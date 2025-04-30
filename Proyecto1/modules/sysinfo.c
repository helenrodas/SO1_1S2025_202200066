#include <linux/module.h>
#include <linux/kernel.h>
#include <linux/init.h>
#include <linux/proc_fs.h>
#include <linux/seq_file.h>
#include <linux/sched.h>
#include <linux/sched/signal.h>
#include <linux/mm.h>
#include <linux/slab.h>
#include <linux/fs.h>
#include <linux/sysinfo.h>
#include <linux/cgroup-defs.h>
#include <linux/cgroup.h>
#include <linux/kernfs.h>
#include <linux/kernel_stat.h>
#include <linux/delay.h>
#include <linux/jiffies.h>

MODULE_LICENSE("GPL");
MODULE_AUTHOR("Helen Rodas");
MODULE_DESCRIPTION("Módulo kernel para métricas memoria y contenedores");
MODULE_VERSION("1.0");

#define PROCFS_NAME "sysinfo_202200066"
#define MAX_CMDLINE_LENGTH 256
#define CONTAINER_ID_LENGTH 64

static unsigned long total_memory_kb = 0;

static u64 prev_cpu_idle_time = 0;
static u64 prev_cpu_total_time = 0;

#define MAX_CONTAINERS 10
static u64 prev_container_cpu_usage[MAX_CONTAINERS] = {0};
static u64 last_update_time = 0;

struct container_info {
    char *container_id;
    char *cgroup_path;
    unsigned long long total_rss_kb;
    unsigned long long disk_usage_kb;
    unsigned long long cpu_usage_percent;
    unsigned long long io_read_ops;  // Nuevo campo para operaciones de lectura
    unsigned long long io_write_ops; // Nuevo campo para operaciones de escritura
    struct list_head list;
};

static char* get_task_cgroup_path(struct task_struct *task) {
    char *path;
    struct cgroup *cgroup;

    path = kmalloc(PATH_MAX, GFP_KERNEL);
    if (!path)
        return NULL;

    rcu_read_lock();
    cgroup = task_cgroup(task, cpu_cgrp_id);
    if (!cgroup || !cgroup->kn) {
        rcu_read_unlock();
        kfree(path);
        return NULL;
    }

    strncpy(path, "/sys/fs/cgroup", PATH_MAX - 1);
    path[PATH_MAX - 1] = '\0';

    if (!kernfs_path(cgroup->kn, path + strlen("/sys/fs/cgroup"), PATH_MAX - strlen("/sys/fs/cgroup"))) {
        rcu_read_unlock();
        kfree(path);
        return NULL;
    }

    rcu_read_unlock();
    return path;
}

static char* get_container_id(const char *cgroup_path) {
    char *id = NULL;
    char *docker_ptr = NULL;

    if (!cgroup_path)
        return NULL;

    docker_ptr = strstr(cgroup_path, "docker-");
    if (docker_ptr) {
        char *id_start = docker_ptr + strlen("docker-");
        char *id_end = strstr(id_start, ".scope");
        size_t id_len = id_end ? (id_end - id_start) : strlen(id_start);

        if (id_len > 0) {
            id = kmalloc(id_len + 1, GFP_KERNEL);
            if (id) {
                strncpy(id, id_start, id_len);
                id[id_len] = '\0';
            }
        }
    }

    return id ? id : kstrdup("unknown", GFP_KERNEL);
}

static void get_container_disk_io(const char *cgroup_path, unsigned long long *disk_usage_kb, unsigned long long *io_read_ops, unsigned long long *io_write_ops) {
    char *io_path;
    struct file *file;
    char *buffer;
    ssize_t bytes_read;
    loff_t pos = 0;
    unsigned long long rbytes = 0, wbytes = 0, rios = 0, wios = 0;

    *disk_usage_kb = 0;
    *io_read_ops = 0;
    *io_write_ops = 0;

    io_path = kmalloc(PATH_MAX, GFP_KERNEL);
    if (!io_path)
        return;

    snprintf(io_path, PATH_MAX, "%s/io.stat", cgroup_path);

    buffer = kmalloc(256, GFP_KERNEL);
    if (!buffer) {
        kfree(io_path);
        return;
    }

    file = filp_open(io_path, O_RDONLY, 0);
    if (IS_ERR(file)) {
        printk(KERN_INFO "Failed to open io.stat file: %s\n", io_path);
        kfree(buffer);
        kfree(io_path);
        return;
    }

    bytes_read = kernel_read(file, buffer, 255, &pos);
    if (bytes_read > 0) {
        buffer[bytes_read] = '\0';
        char *line = buffer;
        while (line) {
            char *next_line = strchr(line, '\n');
            if (next_line) {
                *next_line = '\0';
                next_line++;
            }
            if (strstr(line, "rbytes=")) {
                sscanf(line, "%*s rbytes=%llu wbytes=%llu rios=%llu wios=%llu", &rbytes, &wbytes, &rios, &wios);
                if (rbytes || wbytes) {
                    *disk_usage_kb += (rbytes + wbytes) / 1024;
                }
                *io_read_ops += rios;
                *io_write_ops += wios;
            }
            line = next_line;
        }
    }

    filp_close(file, NULL);
    kfree(buffer);
    kfree(io_path);
}

static unsigned int get_system_cpu_usage(void) {
    u64 cpu_idle_time = 0, cpu_total_time = 0;
    u64 delta_idle, delta_total;
    unsigned int cpu_usage;
    int cpu;

    for_each_possible_cpu(cpu) {
        struct kernel_cpustat *cpu_stats = &kcpustat_cpu(cpu);
        cpu_idle_time += cpu_stats->cpustat[CPUTIME_IDLE];
        cpu_total_time += cpu_stats->cpustat[CPUTIME_USER] +
                          cpu_stats->cpustat[CPUTIME_NICE] +
                          cpu_stats->cpustat[CPUTIME_SYSTEM] +
                          cpu_stats->cpustat[CPUTIME_IDLE] +
                          cpu_stats->cpustat[CPUTIME_IOWAIT] +
                          cpu_stats->cpustat[CPUTIME_IRQ] +
                          cpu_stats->cpustat[CPUTIME_SOFTIRQ];
    }

    delta_idle = cpu_idle_time - prev_cpu_idle_time;
    delta_total = cpu_total_time - prev_cpu_total_time;

    if (delta_total != 0) {
        cpu_usage = 100 - (100 * delta_idle / delta_total);
    } else {
        cpu_usage = 0;
    }

    prev_cpu_idle_time = cpu_idle_time;
    prev_cpu_total_time = cpu_total_time;
    msleep(500);
    return cpu_usage;
}

static u64 get_container_cpu_usage(const char *cgroup_path, int index) {
    char *cpu_path;
    struct file *file;
    char *buffer;
    ssize_t bytes_read;
    u64 usage_usec = 0;
    u64 current_time = get_jiffies_64();
    u64 delta_time, delta_usage;
    u64 cpu_percentage = 0;
    loff_t pos = 0;

    cpu_path = kmalloc(PATH_MAX, GFP_KERNEL);
    if (!cpu_path)
        return 0;

    snprintf(cpu_path, PATH_MAX, "%s/cpu.stat", cgroup_path);

    buffer = kmalloc(256, GFP_KERNEL);
    if (!buffer) {
        kfree(cpu_path);
        return 0;
    }

    file = filp_open(cpu_path, O_RDONLY, 0);
    if (IS_ERR(file)) {
        printk(KERN_INFO "Failed to open cpu.stat file: %s\n", cpu_path);
        kfree(buffer);
        kfree(cpu_path);
        return 0;
    }

    bytes_read = kernel_read(file, buffer, 255, &pos);
    if (bytes_read > 0) {
        buffer[bytes_read] = '\0';
        char *line = buffer;
        while (line) {
            char *next_line = strchr(line, '\n');
            if (next_line) {
                *next_line = '\0';
                next_line++;
            }
            if (sscanf(line, "usage_usec %llu", &usage_usec) == 1) {
                break;
            }
            line = next_line;
        }
    }

    filp_close(file, NULL);
    kfree(buffer);
    kfree(cpu_path);

    if (last_update_time != 0 && usage_usec > prev_container_cpu_usage[index]) {
        delta_time = jiffies_to_usecs(current_time - last_update_time);
        delta_usage = usage_usec - prev_container_cpu_usage[index];
        if (delta_time > 0) {
            cpu_percentage = (delta_usage * 10000) / delta_time;
        }
    }

    prev_container_cpu_usage[index] = usage_usec;
    return cpu_percentage;
}

static void get_memory_info(struct seq_file *m) {
    struct sysinfo info;

    si_meminfo(&info);

    total_memory_kb = info.totalram * info.mem_unit / 1024;
    unsigned long free_kb = info.freeram * info.mem_unit / 1024;
    unsigned long used_kb = total_memory_kb - free_kb;

    seq_printf(m, "\"memory\": {\n");
    seq_printf(m, "    \"total_memory\": %lu,\n", total_memory_kb);
    seq_printf(m, "    \"free_memory\": %lu,\n", free_kb);
    seq_printf(m, "    \"used_memory\": %lu,\n", used_kb);
    seq_printf(m, "    \"cpu_usage_percent\": %u\n", get_system_cpu_usage());
    seq_puts(m, "},\n");
}

static void get_container_processes(struct seq_file *m) {
    struct task_struct *task;
    LIST_HEAD(containers);
    int container_index = 0;

    for_each_process(task) {
        if (strstr(task->comm, "stress") || strstr(task->comm, "docker-")) {
            char *cgroup_path = get_task_cgroup_path(task);
            if (!cgroup_path)
                continue;

            char *container_id = get_container_id(cgroup_path);
            unsigned long rss_kb = task->mm ? get_mm_rss(task->mm) * (PAGE_SIZE / 1024) : 0;

            struct container_info *container = NULL;
            struct container_info *tmp;

            list_for_each_entry(tmp, &containers, list) {
                if (strcmp(tmp->container_id, container_id) == 0) {
                    container = tmp;
                    break;
                }
            }

            if (!container) {
                if (container_index >= MAX_CONTAINERS) {
                    kfree(cgroup_path);
                    kfree(container_id);
                    continue;
                }

                unsigned long long disk_usage_kb = 0;
                unsigned long long io_read_ops = 0;
                unsigned long long io_write_ops = 0;
                get_container_disk_io(cgroup_path, &disk_usage_kb, &io_read_ops, &io_write_ops);
                u64 cpu_usage = get_container_cpu_usage(cgroup_path, container_index);

                container = kmalloc(sizeof(*container), GFP_KERNEL);
                if (!container) {
                    kfree(cgroup_path);
                    kfree(container_id);
                    continue;
                }
                container->container_id = container_id;
                container->cgroup_path = cgroup_path;
                container->total_rss_kb = 0;
                container->disk_usage_kb = disk_usage_kb;
                container->cpu_usage_percent = cpu_usage;
                container->io_read_ops = io_read_ops;
                container->io_write_ops = io_write_ops;
                INIT_LIST_HEAD(&container->list);
                list_add(&container->list, &containers);
                container_index++;
            } else {
                kfree(cgroup_path);
                kfree(container_id);
            }

            container->total_rss_kb += rss_kb;
        }
    }

    int first = 1;
    seq_puts(m, "\"container_processes\": [\n");

    struct container_info *container, *tmp;
    list_for_each_entry_safe(container, tmp, &containers, list) {
        if (!first) {
            seq_puts(m, ",\n");
        }
        first = 0;

        unsigned long long mem_usage_percentage = total_memory_kb ? (container->total_rss_kb * 10000ULL) / total_memory_kb : 0;

        // Modificación: Imprimir memory_usage y cpu_usage como cadenas con comillas
        seq_printf(m, "    {\"container_id\": \"%s\", \"cgroup_path\": \"%s\", "
                      "\"memory_usage\": \"%llu.%02llu%%\", \"disk_usage_kb\": %llu, "
                      "\"cpu_usage\": \"%llu.%02llu%%\", \"io_read_ops\": %llu, \"io_write_ops\": %llu}",
                   container->container_id,
                   container->cgroup_path ? container->cgroup_path : "unknown",
                   mem_usage_percentage / 100, mem_usage_percentage % 100,
                   container->disk_usage_kb,
                   container->cpu_usage_percent / 100, container->cpu_usage_percent % 100,
                   container->io_read_ops,
                   container->io_write_ops);

        kfree(container->container_id);
        kfree(container->cgroup_path);
        list_del(&container->list);
        kfree(container);
    }

    seq_puts(m, "\n]\n");
    last_update_time = get_jiffies_64();
}

static int sysinfo_show(struct seq_file *m, void *v) {
    seq_puts(m, "{\n");
    get_memory_info(m);
    get_container_processes(m);
    seq_puts(m, "}\n");
    return 0;
}

static int sysinfo_open(struct inode *inode, struct file *file) {
    return single_open(file, sysinfo_show, NULL);
}

static const struct proc_ops sysinfo_ops = {
    .proc_open = sysinfo_open,
    .proc_read = seq_read,
    .proc_lseek = seq_lseek,
    .proc_release = single_release,
};

static int __init inicio(void) {
    proc_create(PROCFS_NAME, 0, NULL, &sysinfo_ops);
    printk(KERN_INFO "sysinfo_202200066 cargado correctamente.\n");
    prev_cpu_total_time = get_jiffies_64();
    last_update_time = get_jiffies_64();
    return 0;
}

static void __exit fin(void) {
    remove_proc_entry(PROCFS_NAME, NULL);
    printk(KERN_INFO "sysinfo_202200066 removido.\n");
}

module_init(inicio);
module_exit(fin);
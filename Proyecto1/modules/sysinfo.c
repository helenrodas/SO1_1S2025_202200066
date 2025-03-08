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

MODULE_LICENSE("GPL");
MODULE_AUTHOR("Helen Rodas");
MODULE_DESCRIPTION("Módulo kernel para métricas memoria y contenedores");
MODULE_VERSION("1.0");

#define PROCFS_NAME "sysinfo_202200066"
#define MAX_CMDLINE_LENGTH 256
#define CONTAINER_ID_LENGTH 64

static unsigned long total_memory_kb = 0;

struct container_info {
    char *container_id;
    char *cgroup_path;
    unsigned long long total_rss_kb;
    unsigned long long disk_usage_kb;
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

static void get_container_disk_io(const char *cgroup_path, unsigned long long *disk_usage_kb) {
    char *io_path;
    struct file *file;
    char *buffer;
    ssize_t bytes_read;
    loff_t pos = 0;
    unsigned long long rbytes = 0, wbytes = 0;

    *disk_usage_kb = 0;

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
            // Parsear correctamente todas las líneas que contengan rbytes y wbytes
            if (strstr(line, "rbytes=")) {
                sscanf(line, "%*s rbytes=%llu wbytes=%llu", &rbytes, &wbytes);
                // Sumar solo si la línea tiene ambos valores válidos
                if (rbytes || wbytes) {
                    *disk_usage_kb += (rbytes + wbytes) / 1024;
                }
            }
            line = next_line;
        }
    }

    filp_close(file, NULL);
    kfree(buffer);
    kfree(io_path);
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
    seq_printf(m, "    \"used_memory\": %lu\n", used_kb);
    seq_puts(m, "},\n");
}

static void get_container_processes(struct seq_file *m) {
    struct task_struct *task;
    LIST_HEAD(containers);

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
                unsigned long long disk_usage_kb = 0;
                get_container_disk_io(cgroup_path, &disk_usage_kb);

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
                INIT_LIST_HEAD(&container->list);
                list_add(&container->list, &containers);
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

        unsigned long long usage_percentage = total_memory_kb ? (container->total_rss_kb * 10000ULL) / total_memory_kb : 0;

        seq_printf(m, "    {\"container_id\": \"%s\", \"cgroup_path\": \"%s\", \"memory_usage\": %llu.%02llu%%, \"disk_usage_kb\": %llu}",
                   container->container_id,
                   container->cgroup_path ? container->cgroup_path : "unknown",
                   usage_percentage / 100,
                   usage_percentage % 100,
                   container->disk_usage_kb);

        kfree(container->container_id);
        kfree(container->cgroup_path);
        list_del(&container->list);
        kfree(container);
    }

    seq_puts(m, "\n]\n");
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
    return 0;
}

static void __exit fin(void) {
    remove_proc_entry(PROCFS_NAME, NULL);
    printk(KERN_INFO "sysinfo_202200066 removido.\n");
}

module_init(inicio);
module_exit(fin);
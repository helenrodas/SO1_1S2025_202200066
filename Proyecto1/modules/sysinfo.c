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

// Estructura para rastrear contenedores
struct container_info {
    char *container_id;
    char *cgroup_path;
    unsigned long long total_rss_kb;   // Suma de RSS de todos los procesos
    unsigned long total_cgroup_mem_kb; // Suma de memoria del cgroup
    struct list_head list;
};

// Obtener la ruta del cgroup
static char* get_task_cgroup_path(struct task_struct *task) {
    char *path;
    struct kernfs_node *kn = NULL;
    struct cgroup *cgroup = NULL;

    path = kmalloc(PATH_MAX, GFP_KERNEL);
    if (!path)
        return NULL;

    strncpy(path, "/sys/fs/cgroup", PATH_MAX - 1);
    path[PATH_MAX - 1] = '\0';

    if (task->cgroups && task->cgroups->dfl_cgrp) {
        cgroup = task->cgroups->dfl_cgrp;
        if (cgroup && cgroup->kn) {
            kn = cgroup->kn;
            if (kn && kn->name) {
                strlcat(path, "/", PATH_MAX);
                strlcat(path, kn->name, PATH_MAX);
            }
        }
    }

    return path;
}

// Extraer el ID del contenedor de la ruta del cgroup
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

// Obtener el uso de memoria del cgroup
static unsigned long retrieve_task_mem_usage(struct task_struct *task) {
    char *cgroup_base_path = NULL;
    char *memory_file = NULL;
    struct file *file_handle = NULL;
    char data_buf[64];
    unsigned long mem_value = 0;
    loff_t file_pos = 0;
    ssize_t read_result;

    cgroup_base_path = get_task_cgroup_path(task);
    if (!cgroup_base_path)
        return 0;

    memory_file = kmalloc(PATH_MAX, GFP_KERNEL);
    if (!memory_file) {
        kfree(cgroup_base_path);
        return 0;
    }

    snprintf(memory_file, PATH_MAX, "%s/memory.current", cgroup_base_path);
    file_handle = filp_open(memory_file, O_RDONLY, 0);
    if (IS_ERR(file_handle)) {
        snprintf(memory_file, PATH_MAX, "%s/memory.usage_in_bytes", cgroup_base_path);
        file_handle = filp_open(memory_file, O_RDONLY, 0);
    }

    if (IS_ERR(file_handle)) {
        kfree(memory_file);
        kfree(cgroup_base_path);
        return 0;
    }

    read_result = kernel_read(file_handle, data_buf, sizeof(data_buf) - 1, &file_pos);
    if (read_result > 0) {
        data_buf[read_result] = '\0';
        if (kstrtoul(data_buf, 10, &mem_value) < 0) {
            mem_value = 0;
        }
    }

    filp_close(file_handle, NULL);
    kfree(memory_file);
    kfree(cgroup_base_path);

    return mem_value; // En bytes
}

// Obtener información de memoria del sistema
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

// Mostrar procesos de contenedores agrupados por container_id
static void get_container_processes(struct seq_file *m) {
    struct task_struct *task;
    LIST_HEAD(containers); // Lista para almacenar información de contenedores

    // Primera pasada: recolectar y agrupar datos
    for_each_process(task) {
        if (strstr(task->comm, "stress") || strstr(task->comm, "docker-")) {
            char *cgroup_path = get_task_cgroup_path(task);
            char *container_id = get_container_id(cgroup_path);
            unsigned long mem_usage_bytes = retrieve_task_mem_usage(task);
            unsigned long mem_usage_kb = mem_usage_bytes / 1024;
            unsigned long rss_kb = task->mm ? get_mm_rss(task->mm) * 4 : 0;

            struct container_info *container = NULL;
            struct container_info *tmp;

            // Buscar si el contenedor ya está en la lista
            list_for_each_entry(tmp, &containers, list) {
                if (strcmp(tmp->container_id, container_id) == 0) {
                    container = tmp;
                    break;
                }
            }

            if (!container) {
                // Nuevo contenedor, crear entrada
                container = kmalloc(sizeof(*container), GFP_KERNEL);
                if (!container) {
                    kfree(cgroup_path);
                    kfree(container_id);
                    continue;
                }
                container->container_id = container_id;
                container->cgroup_path = cgroup_path;
                container->total_rss_kb = 0;
                container->total_cgroup_mem_kb = 0;
                INIT_LIST_HEAD(&container->list);
                list_add(&container->list, &containers);
            } else {
                // Contenedor existente, liberar recursos no necesarios
                kfree(cgroup_path);
                kfree(container_id);
            }

            // Acumular métricas
            container->total_rss_kb += rss_kb;
            container->total_cgroup_mem_kb += mem_usage_kb;
        }
    }

    // Segunda pasada: generar JSON
    int first = 1;
    seq_puts(m, "\"container_processes\": [\n");

    struct container_info *container, *tmp;
    list_for_each_entry_safe(container, tmp, &containers, list) {
        if (!first) {
            seq_puts(m, ",\n");
        }
        first = 0;

        unsigned long long usage_percentage = total_memory_kb ? (container->total_rss_kb * 10000ULL) / total_memory_kb : 0;

        seq_printf(m, "    {\"container_id\": \"%s\", \"cgroup_path\": \"%s\", \"memory_usage\": %llu.%02llu%%, \"cgroup_mem_usage_kb\": %lu}",
                   container->container_id,
                   container->cgroup_path ? container->cgroup_path : "unknown",
                   usage_percentage / 100,
                   usage_percentage % 100,
                   container->total_cgroup_mem_kb);

        // Liberar recursos
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
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

// Obtener la ruta del cgroup de manera simplificada
static char* get_task_cgroup_path(struct task_struct *task) {
    char *path;
    struct kernfs_node *kn = NULL;
    struct cgroup *cgroup = NULL;

    path = kmalloc(PATH_MAX, GFP_KERNEL);
    if (!path)
        return NULL;

    // Inicializar con la ruta base
    strncpy(path, "/sys/fs/cgroup", PATH_MAX - 1);
    path[PATH_MAX - 1] = '\0';

    // En versiones recientes del kernel, la obtención de la ruta del cgroup
    // es más complicada. Simplificamos para detectar procesos de contenedores
    // buscando "docker" en la ruta de cgroup del proceso
    if (task->cgroups && task->cgroups->dfl_cgrp) {
        cgroup = task->cgroups->dfl_cgrp;
        if (cgroup && cgroup->kn) {
            kn = cgroup->kn;
            if (kn && kn->name) {
                // Intentamos construir una ruta más descriptiva
                strlcat(path, "/", PATH_MAX);
                strlcat(path, kn->name, PATH_MAX);
            }
        }
    }

    return path;
}

// Nueva versión modificada para obtener el uso de memoria
static unsigned long retrieve_task_mem_usage(struct task_struct *task) {
    char *cgroup_base_path = NULL;
    char *memory_file = NULL;
    struct file *file_handle = NULL;
    char data_buf[64];  // Buffer más grande por seguridad
    unsigned long mem_value = 0;
    loff_t file_pos = 0;
    ssize_t read_result;

    // Obtener la ruta base del cgroup
    cgroup_base_path = get_task_cgroup_path(task);
    if (!cgroup_base_path) {
        printk(KERN_ERR "No se pudo obtener la ruta base del cgroup\n");
        return 0;
    }

    // Asignar memoria dinámica para la ruta del archivo
    memory_file = kmalloc(PATH_MAX, GFP_KERNEL);
    if (!memory_file) {
        printk(KERN_ERR "Fallo al asignar memoria para la ruta del archivo\n");
        kfree(cgroup_base_path);
        return 0;
    }

    // Construir la ruta completa - intenta primero con memory.current (cgroup v2)
    snprintf(memory_file, PATH_MAX, "%s/memory.current", cgroup_base_path);
    
    // Abrir el archivo
    file_handle = filp_open(memory_file, O_RDONLY, 0);
    if (IS_ERR(file_handle)) {
        // Si falla, intenta con memory.usage_in_bytes (cgroup v1)
        snprintf(memory_file, PATH_MAX, "%s/memory.usage_in_bytes", cgroup_base_path);
        file_handle = filp_open(memory_file, O_RDONLY, 0);
    }

    if (IS_ERR(file_handle)) {
        printk(KERN_ERR "Error abriendo archivo de memoria: %ld\n", PTR_ERR(file_handle));
        kfree(memory_file);
        kfree(cgroup_base_path);
        return 0;
    }

    // Leer el contenido del archivo
    read_result = kernel_read(file_handle, data_buf, sizeof(data_buf) - 1, &file_pos);
    if (read_result > 0) {
        data_buf[read_result] = '\0';
        if (kstrtoul(data_buf, 10, &mem_value) < 0) {  // Verificación más estricta
            printk(KERN_ERR "Error al convertir el valor de memoria\n");
            mem_value = 0;
        }
    } else {
        printk(KERN_ERR "Fallo al leer archivo de memoria: %ld\n", read_result);
    }

    // Liberar recursos
    filp_close(file_handle, NULL);
    kfree(memory_file);
    kfree(cgroup_base_path);

    return mem_value;  // Devuelve el uso de memoria en bytes
}

static unsigned long total_memory_kb = 0;

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
    int first = 1;

    seq_puts(m, "\"container_processes\": [\n");

    for_each_process(task) {
        if (strstr(task->comm, "stress") || strstr(task->comm, "docker-")) {
            char *cgroup_path = get_task_cgroup_path(task);
            unsigned long mem_usage_bytes = retrieve_task_mem_usage(task);
            unsigned long mem_usage_kb = mem_usage_bytes / 1024;  // Convertir a KB

            if (!first) {
                seq_puts(m, ",\n");
            }
            first = 0;

            unsigned long rss_kb = 0;
            if (task->mm) {
                rss_kb = get_mm_rss(task->mm) * 4;  // 4KB por página
            }
            
            unsigned long long usage_percentage = total_memory_kb ? (rss_kb * 10000ULL) / total_memory_kb : 0;

            seq_printf(m, "    {\"pid\": %d, \"name\": \"%s\", \"memory_usage\": %llu.%02llu%%, \"cgroup_path\": \"%s\", \"cgroup_mem_usage_kb\": %lu}",
                       task->pid, 
                       task->comm,
                       usage_percentage / 100, 
                       usage_percentage % 100,
                       cgroup_path ? cgroup_path : "unknown",
                       mem_usage_kb);

            if (cgroup_path)
                kfree(cgroup_path);
        }
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
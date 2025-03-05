#include <linux/module.h>
#include <linux/kernel.h>
#include <linux/init.h>
#include <linux/proc_fs.h>
#include <linux/seq_file.h>
#include <linux/sched.h>
#include <linux/sched/signal.h>

MODULE_LICENSE("GPL");
MODULE_AUTHOR("Helen Rodas");
MODULE_DESCRIPTION("Modelo de kernel para mostrar procesos de Docker");
MODULE_VERSION("1.0");

#define PROCFS_NAME "sysinfo_202200066"

// Función para mostrar la información en /proc/docker_process
static int sysinfor_show(struct seq_file *m, void *v) {
    struct task_struct *task;

    for_each_process(task) {
        if (strstr(task->comm, "containerd-shim")) {
            seq_printf(m, "PID: %d, Nombre: %s\n", task->pid, task->comm);
        }
    }

    return 0;
}

static int sysinfo_open(struct inode *inode, struct file *file) {
    return single_open(file, sysinfor_show, NULL);
}

// Estructura para mostrar la información en /proc/docker_process
static const struct proc_ops sysinfo_ops = {
    .proc_open = sysinfo_open,
    .proc_read = seq_read,
    .proc_lseek = seq_lseek,
    .proc_release = single_release,
};

static int __init inicio(void) {
    proc_create(PROCFS_NAME, 0, NULL, &sysinfo_ops);
    printk(KERN_INFO "Módulo cargado correctamente\n");
    return 0;
}

static void __exit fin(void) {
    remove_proc_entry(PROCFS_NAME, NULL);
    printk(KERN_INFO "Fin de modelo\n");
}

module_init(inicio);
module_exit(fin);
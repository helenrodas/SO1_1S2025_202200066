use serde::{Deserialize, Serialize};
use std::fs;
use std::process::Command;
use std::time::Duration;
use std::thread;
use chrono::Utc;
use reqwest::blocking::Client;
use ctrlc;
use std::collections::HashMap;

#[derive(Debug, Serialize, Deserialize)]
struct Memory {
    total_memory: u64,
    free_memory: u64,
    used_memory: u64,
    cpu_usage_percent: u32,
}

#[derive(Debug, Serialize, Deserialize)]
struct Container {
    container_id: String,
    cgroup_path: String,
    memory_usage: String,
    disk_usage_kb: u64,
    cpu_usage: String,
    io_read_ops: u64,
    io_write_ops: u64,
}

#[derive(Debug, Serialize, Deserialize)]
struct SysInfo {
    memory: Memory,
    container_processes: Vec<Container>,
}

#[derive(Debug, Serialize)]
struct LogEntry {
    container_id: String,
    category: String,
    created_at: String,
    deleted_at: Option<String>,
}

fn get_container_names() -> HashMap<String, String> {
    let output = Command::new("docker")
        .args(&["ps", "-a", "--format", "{{.ID}} {{.Names}}"])
        .output()
        .expect("Failed to run docker ps");
    let output_str = String::from_utf8_lossy(&output.stdout);
    let mut name_map = HashMap::new();
    for line in output_str.lines() {
        let parts: Vec<&str> = line.split_whitespace().collect();
        if parts.len() == 2 {
            name_map.insert(parts[0].to_string(), parts[1].to_string());
        }
    }
    name_map
}

fn main() {
    let log_url = "http://localhost:8080/logs";
    let client = Client::new();
    let mut logs: Vec<LogEntry> = Vec::new();

    let running = std::sync::Arc::new(std::sync::atomic::AtomicBool::new(true));
    let r = running.clone();
    ctrlc::set_handler(move || r.store(false, std::sync::atomic::Ordering::SeqCst))
        .expect("Error setting Ctrl+C handler");

    println!("Servicio iniciado. Presiona Ctrl+C para detener.");

    // Mostrar la información de memoria solo una vez al inicio
    let contents = fs::read_to_string("/proc/sysinfo_202200066")
        .expect("Failed to read sysinfo");
    let sysinfo: SysInfo = serde_json::from_str(&contents)
        .expect("Failed to deserialize JSON");

    println!("=== Memoria ===");
    println!("Total: {} KB, Libre: {} KB, Usada: {} KB, CPU: {}%", 
        sysinfo.memory.total_memory, sysinfo.memory.free_memory, 
        sysinfo.memory.used_memory, sysinfo.memory.cpu_usage_percent);

    while running.load(std::sync::atomic::Ordering::SeqCst) {
        let name_map = get_container_names();

        println!("\n=== Contenedores ===");
        for container in &sysinfo.container_processes {
            let short_id = if container.container_id.len() >= 12 { &container.container_id[..12] } else { &container.container_id };
            let name = name_map.get(short_id).unwrap_or(&container.container_id).to_string();

            println!("- {} (Memoria: {}, CPU: {}, Disco: {} KB, I/O Lectura: {}, I/O Escritura: {})", 
                name, container.memory_usage, container.cpu_usage, container.disk_usage_kb, container.io_read_ops, container.io_write_ops);
        }

        thread::sleep(Duration::from_secs(10));
    }

    println!("Finalizando servicio...");
    if let Err(e) = client.post(format!("{}/generate_graphs", log_url)).send() {
        println!("Error generando gráficas: {}", e);
    } else {
        println!("Gráficas generadas");
    }

    Command::new("crontab")
        .arg("-r")
        .output()
        .expect("Failed to remove crontab");

    println!("Servicio detenido.");
}
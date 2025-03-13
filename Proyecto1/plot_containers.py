import matplotlib.pyplot as plt
plt.switch_backend('TkAgg')  # Forzar backend interactivo
import json
from datetime import datetime

# Ruta al archivo logs.json
log_file_path = "/home/helen/Programacion/sopes/SO1_1S2025_202200066/Proyecto1/logs.json"

# Leer el archivo JSON
with open(log_file_path, 'r') as file:
    logs = json.load(file)

# Preparar datos para graficar
timestamps = []
memory_usage = []
cpu_usage = []
container_ids = []

for entry in logs:
    # Ajustar created_at recortando microsegundos a 6 dígitos si es necesario
    created_at = entry['created_at']
    if len(created_at.split('.')[1].split('+')[0]) > 6:
        parts = created_at.split('.')
        microseconds = parts[1][:6]
        created_at = f"{parts[0]}.{microseconds}+00:00"
    
    # Parsear el timestamp
    timestamp = datetime.strptime(created_at, "%Y-%m-%dT%H:%M:%S.%f+00:00")
    timestamps.append(timestamp)
    
    # Extraer métricas (convertir porcentajes a flotantes)
    memory = float(entry['memory_usage'].replace('%', ''))
    cpu = float(entry['cpu_usage'].replace('%', ''))
    
    memory_usage.append(memory)
    cpu_usage.append(cpu)
    container_ids.append(entry['container_id'])

# Crear figura con dos gráficos
plt.figure(figsize=(12, 8))

# Gráfico 1: Uso de memoria
plt.subplot(2, 1, 1)
plt.plot(timestamps, memory_usage, marker='o', label='Memory Usage (%)', color='blue')
plt.title('Memory Usage Over Time')
plt.xlabel('Time')
plt.ylabel('Memory Usage (%)')
plt.xticks(rotation=45)
plt.grid(True)
plt.legend()

# Gráfico 2: Uso de CPU
plt.subplot(2, 1, 2)
plt.plot(timestamps, cpu_usage, marker='o', label='CPU Usage (%)', color='orange')
plt.title('CPU Usage Over Time')
plt.xlabel('Time')
plt.ylabel('CPU Usage (%)')
plt.xticks(rotation=45)
plt.grid(True)
plt.legend()

# Ajustar el diseño y mostrar
plt.tight_layout()
plt.show()
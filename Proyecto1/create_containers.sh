#!/bin/bash

# Imagen base
IMAGE="containerstack/alpine-stress"

# Función para generar un nombre aleatorio
generate_name() {
    TIMESTAMP=$(/usr/bin/date +%s)
    RANDOM_SUFFIX=$(/usr/bin/od -x /dev/urandom | /usr/bin/head -1 | /usr/bin/awk '{print $2$3}')
    echo "${1}_${TIMESTAMP}_${RANDOM_SUFFIX}"
}

# Tipos de contenedores
TYPES=("ram" "cpu" "io" "disk")

# Crear 10 contenedores aleatorios
for ((i=1; i<=10; i++)); do
    TYPE=${TYPES[$RANDOM % ${#TYPES[@]}]}
    CONTAINER_NAME=$(generate_name "$TYPE")
    case $TYPE in
        "ram")
            CMD="stress --vm 1 --vm-bytes 200M --vm-keep"
            ;;
        "cpu")
            CMD="stress --cpu 1"
            ;;
        "io")
            CMD="stress --io 1"
            ;;
        "disk")
            CMD="stress --hdd 1 --hdd-bytes 1G"
            ;;
    esac
    /usr/bin/docker run -d --name "$CONTAINER_NAME" "$IMAGE" sh -c "$CMD"
    /usr/bin/echo "Creado contenedor: $CONTAINER_NAME ($TYPE)"
done
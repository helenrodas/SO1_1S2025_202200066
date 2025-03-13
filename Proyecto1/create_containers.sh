#!/bin/bash

# Imagen base
IMAGE="containerstack/alpine-stress"

# Función para generar un nombre aleatorio
generate_name() {
    # Usamos timestamp + un número aleatorio de /dev/urandom
    TIMESTAMP=$(date +%s)
    RANDOM_SUFFIX=$(od -x /dev/urandom | head -1 | awk '{print $2$3}')
    echo "${1}_${TIMESTAMP}_${RANDOM_SUFFIX}"
}

# Tipos de contenedores
TYPES=("ram" "cpu" "io" "disk")

# Crear 10 contenedores
for ((i=1; i<=3; i++)); do
    # Seleccionar tipo aleatorio
    TYPE=${TYPES[$RANDOM % ${#TYPES[@]}]}

    # Generar nombre único
    CONTAINER_NAME=$(generate_name "$TYPE")

    # Configurar comando según el tipo
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

    # Crear el contenedor
    docker run -d --name "$CONTAINER_NAME" "$IMAGE" sh -c "$CMD" > /dev/null 2>&1
    echo "Creado contenedor: $CONTAINER_NAME ($TYPE)"
done
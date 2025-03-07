#!/bin/bash

# Directorio de logs local (puede cambiar si quieres)
LOG_DIR="./logs"
mkdir -p "$LOG_DIR"

# Imagen base
IMAGE="containerstack/alpine-stress:latest"

# Tipos de contenedores a generar (según las notas del proyecto)
TYPES=("ram" "cpu" "io" "disk")

# Crear 10 contenedores aleatorios
for i in $(seq 1 3); do
    TYPE=${TYPES[$((RANDOM % 4))]}
    #TYPE="ram"


    # Nombre único basado en fecha y random (para evitar colisiones)
    CONTAINER_NAME="${TYPE}_$(date +%s)_$(head /dev/urandom | tr -dc A-Za-z0-9 | head -c 6)"

    case "$TYPE" in
        ram)
            docker run -d --name "$CONTAINER_NAME" "$IMAGE" stress --vm 1 --vm-bytes 200M --vm-hang 0
            ;;
        cpu)
            docker run -d --name "$CONTAINER_NAME" "$IMAGE" stress --cpu-load 0.2
            ;;
        io)
            docker run -d --name "$CONTAINER_NAME" "$IMAGE" stress --io 1
            ;;
        disk)
            docker run -d --name "$CONTAINER_NAME" "$IMAGE" stress --hdd 1 --hdd-bytes 500M
            ;;
    esac

    if [ $? -eq 0 ]; then
        echo "$(date) - Creado: $CONTAINER_NAME (Tipo: $TYPE)" >> "$LOG_DIR/containers.log"
    else
        echo "$(date) - Error al crear: $CONTAINER_NAME (Tipo: $TYPE)" >> "$LOG_DIR/containers.log"
    fi
done

echo "Contenedores creados correctamente. Revisa $LOG_DIR/containers.log para más detalles."
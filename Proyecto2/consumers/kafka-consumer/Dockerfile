# Usar una imagen base de Go con soporte para C
FROM golang:1.22 AS builder

# Instalar dependencias necesarias para librdkafka
RUN apt-get update && apt-get install -y \
    build-essential \
    librdkafka-dev \
    && rm -rf /var/lib/apt/lists/*

# Establecer el directorio de trabajo
WORKDIR /app

# Copiar go.mod y go.sum para instalar dependencias
COPY go.mod ./
COPY go.sum ./
RUN go mod download

# Copiar el código fuente
COPY . .

# Compilar el binario
RUN CGO_ENABLED=1 GOOS=linux go build -o kafka-consumer .

# Usar una imagen de runtime con una versión compatible de glibc
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y librdkafka1 && rm -rf /var/lib/apt/lists/*
WORKDIR /root/
COPY --from=builder /app/kafka-consumer .
CMD ["./kafka-consumer"]
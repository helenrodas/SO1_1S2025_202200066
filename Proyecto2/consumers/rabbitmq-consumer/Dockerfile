# Usar una imagen base de Go
FROM golang:1.21 AS builder

# Establecer el directorio de trabajo
WORKDIR /app

# Copiar go.mod y go.sum para instalar dependencias
COPY go.mod ./
COPY go.sum ./
RUN go mod download

# Copiar el código fuente
COPY . .

# Compilar el binario
RUN CGO_ENABLED=0 GOOS=linux go build -o rabbitmq-consumer .

# Usar una imagen ligera para el runtime
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/rabbitmq-consumer .
CMD ["./rabbitmq-consumer"]
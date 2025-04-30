FROM golang:1.22 AS builder
WORKDIR /app
COPY . .
# Instalar librdkafka
RUN apt-get update && apt-get install -y librdkafka-dev
# Descargar dependencias y construir
RUN go mod download
RUN CGO_ENABLED=1 GOOS=linux go build -o kafka-consumer .

FROM alpine:latest
# Instalar librdkafka en la imagen final
RUN apk add --no-cache librdkafka
COPY --from=builder /app/kafka-consumer /kafka-consumer
CMD ["/kafka-consumer"]
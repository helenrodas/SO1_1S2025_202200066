FROM golang:1.22 AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o rabbitmq-consumer .

FROM alpine:latest
COPY --from=builder /app/rabbitmq-consumer /rabbitmq-consumer
CMD ["/rabbitmq-consumer"]
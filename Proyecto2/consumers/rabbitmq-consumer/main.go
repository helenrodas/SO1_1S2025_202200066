package main

import (
    "log"

    "github.com/streadway/amqp"
)

func failOnError(err error, msg string) {
    if err != nil {
        log.Fatalf("%s: %s", msg, err)
    }
}

func main() {
    conn, err := amqp.Dial("amqp://guest:guest@rabbitmq-service.tweets-clima.svc.cluster.local:5672/")
    failOnError(err, "Failed to connect to RabbitMQ")
    defer conn.Close()

    ch, err := conn.Channel()
    failOnError(err, "Failed to open a channel")
    defer ch.Close()

    q, err := ch.QueueDeclare(
        "message", // name
        false,     // durable
        false,     // auto-deleted
        false,     // exclusive
        false,     // no-wait
        nil,       // args
    )
    failOnError(err, "Failed to declare a queue")

    msgs, err := ch.Consume(
        q.Name, // queue
        "",     // consumer
        true,   // auto-ack
        false,  // exclusive
        false,  // no-local
        false,  // no-wait
        nil,    // args
    )
    failOnError(err, "Failed to register a consumer")

    log.Printf(" [*] Waiting for messages in queue 'message'. To exit press CTRL+C")
    for msg := range msgs {
        log.Printf("Received a message: %s", msg.Body)
    }
}
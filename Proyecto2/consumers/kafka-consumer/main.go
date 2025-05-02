package main

import (
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/confluentinc/confluent-kafka-go/kafka"
)

func main() {
    consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
        "bootstrap.servers": "my-cluster-kafka-bootstrap.tweets-clima.svc.cluster.local:9092",
        "group.id":          "kafka-consumer-group",
        "auto.offset.reset": "earliest",
    })
    if err != nil {
        log.Fatalf("Failed to create consumer: %v", err)
    }
    defer consumer.Close()

    topic := "message"
    err = consumer.SubscribeTopics([]string{topic}, nil)
    if err != nil {
        log.Fatalf("Failed to subscribe to topic: %v", err)
    }

    sigchan := make(chan os.Signal, 1)
    signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

    log.Printf(" [*] Waiting for messages in topic 'message'. To exit press CTRL+C")
    for {
        select {
        case sig := <-sigchan:
            log.Printf("Caught signal %v: terminating", sig)
            return
        default:
            msg, err := consumer.ReadMessage(-1)
            if err == nil {
                log.Printf("Received message: %s", string(msg.Value))
            } else {
                log.Printf("Consumer error: %v (%v)", err, msg)
            }
        }
    }
}
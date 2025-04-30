package main

import (
	"log"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

func main() {
	// Conectar a Kafka
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "kafka:9092",
		"group.id":          "kafka-consumer-group",
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		log.Fatalf("No se pudo conectar a Kafka: %v", err)
	}
	defer consumer.Close()

	// Suscribirse a un tópico
	err = consumer.SubscribeTopics([]string{"message"}, nil)
	if err != nil {
		log.Fatalf("No se pudo suscribir al tópico: %v", err)
	}

	// Consumir mensajes
	for {
		msg, err := consumer.ReadMessage(-1)
		if err != nil {
			log.Printf("Error al leer mensaje: %v", err)
			continue
		}
		log.Printf("Mensaje recibido: %s\n", string(msg.Value))
	}
}

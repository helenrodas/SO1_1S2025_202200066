package main

import (
	"log"

	"github.com/streadway/amqp"
	"github.com/go-redis/redis/v8"
	"context"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func main() {
	// Conectar a RabbitMQ
	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq-service.tweets-clima.svc.cluster.local:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	// Declarar la cola
	q, err := ch.QueueDeclare(
		"message", // name
		false,     // durable
		false,     // auto-deleted
		false,     // exclusive
		false,     // no-wait
		nil,       // args
	)
	failOnError(err, "Failed to declare a queue")

	// Consumir mensajes
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
		tweet := string(msg.Body)
		// Extraer país del mensaje (asumiendo que está en el JSON como "country")
		country := "Guatemala" // Ajusta esto según el parsing real del JSON
		// Incrementar contadores en Valkey
		incrCounterInValkey("valkey-service.tweets-clima.svc.cluster.local:6379", "", "country_counts", country, 1)
		incrCounterInValkey("valkey-service.tweets-clima.svc.cluster.local:6379", "", "message_total", "", 1)
		log.Printf("Received a message: %s", tweet)
	}
}

// Función para incrementar contadores en Valkey usando tablas hash
func incrCounterInValkey(addr, password, hashKey, field string, increment int64) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password, // Vacío, no se usa contraseña
		DB:       0,
	})
	defer client.Close()
	err := client.HIncrBy(context.Background(), hashKey, field, increment).Err()
	if err != nil {
		log.Printf("Error incrementing counter in Valkey: %v", err)
	}
}
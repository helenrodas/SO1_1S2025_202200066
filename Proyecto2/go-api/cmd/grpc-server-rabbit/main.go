package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/redis/go-redis/v9"
	"github.com/streadway/amqp"
)

type Tweet struct {
	Description string `json:"description"`
	Country     string `json:"country"`
	Weather     string `json:"weather"`
}

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	if err != nil {
		log.Fatalf("No se pudo conectar a RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("No se pudo abrir canal: %v", err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare("message", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("No se pudo declarar cola: %v", err)
	}

	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("No se pudo consumir mensajes: %v", err)
	}

	client := redis.NewClient(&redis.Options{Addr: "valkey:6379"})
	defer client.Close()

	for msg := range msgs {
		var tweet Tweet
		if err := json.Unmarshal(msg.Body, &tweet); err != nil {
			log.Printf("Error al decodificar mensaje: %v", err)
			continue
		}

		client.HIncrBy(context.Background(), "tweets:countries", tweet.Country, 1)
		client.Incr(context.Background(), "tweets:total")
	}
}

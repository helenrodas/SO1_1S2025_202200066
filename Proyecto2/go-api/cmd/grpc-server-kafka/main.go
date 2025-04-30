package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/redis/go-redis/v9"
)

type Tweet struct {
	Description string `json:"description"`
	Country     string `json:"country"`
	Weather     string `json:"weather"`
}

func main() {
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "kafka:9092",
		"group.id":          "kafka-consumer",
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		log.Fatalf("No se pudo conectar a Kafka: %v", err)
	}
	defer consumer.Close()

	client := redis.NewClient(&redis.Options{Addr: "redis:6379"})
	defer client.Close()

	consumer.SubscribeTopics([]string{"message"}, nil)
	for {
		msg, err := consumer.ReadMessage(-1)
		if err != nil {
			log.Printf("Error al leer mensaje: %v", err)
			continue
		}

		var tweet Tweet
		if err := json.Unmarshal(msg.Value, &tweet); err != nil {
			log.Printf("Error al decodificar mensaje: %v", err)
			continue
		}

		client.HIncrBy(context.Background(), "tweets:countries", tweet.Country, 1)
		client.Incr(context.Background(), "tweets:total")
	}
}

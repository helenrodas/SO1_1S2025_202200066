package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/go-redis/redis/v8"
)

func main() {
	// Configurar el consumidor de Kafka
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "my-cluster-kafka-bootstrap.tweets-clima.svc.cluster.local:9092",
		"group.id":          "kafka-consumer-group",
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	// Suscribirse al topic
	topic := "message"
	err = consumer.SubscribeTopics([]string{topic}, nil)
	if err != nil {
		log.Fatalf("Failed to subscribe to topic: %v", err)
	}

	// Canal para manejar señales de terminación
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	// Iniciar consumo de mensajes
	log.Printf(" [*] Waiting for messages in topic 'message'. To exit press CTRL+C")
	for {
		select {
		case sig := <-sigchan:
			log.Printf("Caught signal %v: terminating", sig)
			return
		default:
			msg, err := consumer.ReadMessage(-1)
			if err == nil {
				tweet := string(msg.Value)
				// Extraer país del mensaje (asumiendo que está en el JSON como "country")
				country := "Guatemala" // Ajusta esto según el parsing real del JSON
				// Incrementar contadores en Redis
				incrCounterInRedis("redis-service.tweets-clima.svc.cluster.local:6379", "", "country_counts", country, 1)
				incrCounterInRedis("redis-service.tweets-clima.svc.cluster.local:6379", "", "message_total", "", 1)
				log.Printf("Received message: %s", tweet)
			} else {
				log.Printf("Consumer error: %v (%v)", err, msg)
			}
		}
	}
}

// Función para incrementar contadores en Redis usando tablas hash
func incrCounterInRedis(addr, password, hashKey, field string, increment int64) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password, // Vacío, no se usa contraseña
		DB:       0,
	})
	defer client.Close()
	err := client.HIncrBy(context.Background(), hashKey, field, increment).Err()
	if err != nil {
		log.Printf("Error incrementing counter in Redis: %v", err)
	}
}
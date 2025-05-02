package main

import (
    "context"
    "encoding/json"
    "log"

    "github.com/IBM/sarama"
    "github.com/go-redis/redis/v8"
)

func failOnError(err error, msg string) {
    if err != nil {
        log.Fatalf("%s: %s", msg, err)
    }
}

func incrCounterInRedis(addr, password, hashKey, field string, increment int64) {
    client := redis.NewClient(&redis.Options{
        Addr:     addr,
        Password: password,
        DB:       0,
    })
    defer client.Close()
    err := client.HIncrBy(context.Background(), hashKey, field, increment).Err()
    if err != nil {
        log.Printf("Error incrementing counter in Redis: %v", err)
    }
}

func main() {
    // Configuración de Kafka
    brokerList := []string{"my-cluster-kafka-bootstrap.tweets-clima.svc.cluster.local:9092"}
    consumerGroupID := "tweet-group" // ID del grupo de consumidores
    topic := "message"

    // Crear configuración para el consumidor
    config := sarama.NewConfig()
    config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
    config.Consumer.Offsets.Initial = sarama.OffsetOldest

    // Crear el consumidor de grupo
    consumerGroup, err := sarama.NewConsumerGroup(brokerList, consumerGroupID, config)
    failOnError(err, "Failed to create consumer group")
    defer consumerGroup.Close()

    // Contexto para manejar señales de cierre
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Consumir mensajes
    handler := consumerGroupHandler{}
    log.Printf(" [*] Waiting for messages in topic '%s'. To exit press CTRL+C", topic)
    go func() {
        for {
            err := consumerGroup.Consume(ctx, []string{topic}, handler)
            if err != nil {
                log.Printf("Error from consumer: %v", err)
            }
            if ctx.Err() != nil {
                return
            }
        }
    }()

    // Mantener el programa en ejecución hasta que se cancele
    <-ctx.Done()
}

// Handler para procesar mensajes de Kafka
type consumerGroupHandler struct{}

func (h consumerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h consumerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }
func (h consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
    for msg := range claim.Messages() {
        var tweet map[string]string
        err := json.Unmarshal(msg.Value, &tweet)
        if err != nil {
            log.Printf("Error parsing JSON: %v", err)
            continue
        }
        country := tweet["country"]
        // Incrementar contadores en Redis
        incrCounterInRedis("redis-service.tweets-clima.svc.cluster.local:6379", "", "country_counts", country, 1)
        incrCounterInRedis("redis-service.tweets-clima.svc.cluster.local:6379", "", "message_total", "", 1)
        log.Printf("Received message: %s", string(msg.Value))
        session.MarkMessage(msg, "")
    }
    return nil
}
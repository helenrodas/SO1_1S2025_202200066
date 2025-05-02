package main

import (
    "context"
    "encoding/json"
    "log"
    "net"
    "time"

    pb "tweets-clima/proto"

    "google.golang.org/grpc"
    "github.com/confluentinc/confluent-kafka-go/kafka"
)

type server struct {
    pb.UnimplementedTweetServiceServer
}

func (s *server) PublishTweet(ctx context.Context, req *pb.TweetRequest) (*pb.TweetResponse, error) {
    description := req.GetDescription()
    country := req.GetCountry()
    weather := req.GetWeather()

    if description == "" {
        log.Println("El campo description está vacío")
        return nil, nil
    }
    log.Printf("Recibido tweet para publicar en Kafka - Description: %s, Country: %s, Weather: %s", description, country, weather)

    producer, err := kafka.NewProducer(&kafka.ConfigMap{
        "bootstrap.servers": "my-cluster-kafka-bootstrap.tweets-clima.svc.cluster.local:9092",
    })
    if err != nil {
        log.Printf("Error al crear productor de Kafka: %v", err)
        return nil, err
    }
    defer producer.Close()

    tweetJSON, err := json.Marshal(req)
    if err != nil {
        log.Printf("Error al serializar el tweet a JSON: %v", err)
        return nil, err
    }

    topic := "message"
    err = producer.Produce(&kafka.Message{
        TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
        Value:          tweetJSON,
    }, nil)
    if err != nil {
        log.Printf("Error al publicar en Kafka: %v", err)
        return nil, err
    }

    producer.Flush(10 * 1000)
    log.Println("Tweet publicado en Kafka con éxito")
    return &pb.TweetResponse{Message: "Tweet publicado en Kafka"}, nil
}

func main() {
    var lis net.Listener
    var err error
    for {
        lis, err = net.Listen("tcp", ":50053")
        if err != nil {
            log.Printf("No se pudo escuchar en el puerto 50053, reintentando en 5s: %v", err)
            time.Sleep(5 * time.Second)
            continue
        }
        log.Println("Escuchando en el puerto 50053")
        break
    }

    s := grpc.NewServer()
    pb.RegisterTweetServiceServer(s, &server{})
    log.Println("Iniciando servidor gRPC para Kafka en :50053")
    log.Fatal(s.Serve(lis))
}
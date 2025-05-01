// package main

// import (
// 	"context"
// 	"log"
// 	"net"
// 	"time"

// 	pb "tweets-clima/proto"

// 	"github.com/confluentinc/confluent-kafka-go/kafka"
// 	"github.com/streadway/amqp"
// 	"google.golang.org/grpc"
// )

// type server struct {
// 	pb.UnimplementedTweetServiceServer
// 	rabbitConn    *amqp.Connection
// 	kafkaProducer *kafka.Producer
// }

// func (s *server) PublishTweet(ctx context.Context, req *pb.TweetRequest) (*pb.TweetResponse, error) {
// 	// Publicar en RabbitMQ
// 	rabbitCh, err := s.rabbitConn.Channel()
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rabbitCh.Close()
// 	err = rabbitCh.Publish(
// 		"", "message", false, false,
// 		amqp.Publishing{
// 			ContentType: "application/json",
// 			Body:        []byte(req.String()),
// 		},
// 	)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Publicar en Kafka
// 	err = s.kafkaProducer.Produce(&kafka.Message{
// 		TopicPartition: kafka.TopicPartition{Topic: stringPtr("message"), Partition: kafka.PartitionAny},
// 		Value:          []byte(req.String()),
// 	}, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &pb.TweetResponse{Message: "Tweet publicado"}, nil
// }

// func stringPtr(s string) *string {
// 	return &s
// }

// func main() {
// 	var rabbitConn *amqp.Connection
// 	var kafkaProducer *kafka.Producer
// 	var err error

// 	// Reintentar conexión a RabbitMQ
// 	for {
// 		rabbitConn, err = amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
// 		if err != nil {
// 			log.Printf("No se pudo conectar a RabbitMQ, reintentando en 5s: %v", err)
// 			time.Sleep(5 * time.Second)
// 			continue
// 		}
// 		log.Println("Conectado a RabbitMQ")
// 		break
// 	}
// 	defer rabbitConn.Close()

// 	// Reintentar conexión a Kafka
// 	for {
// 		kafkaProducer, err = kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": "kafka:9092"})
// 		if err != nil {
// 			log.Printf("No se pudo conectar a Kafka, reintentando en 5s: %v", err)
// 			time.Sleep(5 * time.Second)
// 			continue
// 		}
// 		log.Println("Conectado a Kafka")
// 		break
// 	}
// 	defer kafkaProducer.Close()

// 	// Reintentar escuchar en el puerto 50051
// 	var lis net.Listener
// 	for {
// 		lis, err = net.Listen("tcp", ":50051")
// 		if err != nil {
// 			log.Printf("No se pudo escuchar en el puerto 50051, reintentando en 5s: %v", err)
// 			time.Sleep(5 * time.Second)
// 			continue
// 		}
// 		log.Println("Escuchando en el puerto 50051")
// 		break
// 	}

// 	// Iniciar servidor gRPC
// 	s := grpc.NewServer()
// 	pb.RegisterTweetServiceServer(s, &server{rabbitConn: rabbitConn, kafkaProducer: kafkaProducer})
// 	log.Println("Iniciando servidor gRPC en :50051")
// 	log.Fatal(s.Serve(lis))
// }



package main

import (
    "context"
    "log"
    "net"

    pb "tweets-clima/proto"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

type server struct {
    pb.UnimplementedTweetServiceServer
}

func (s *server) PublishTweet(ctx context.Context, req *pb.TweetRequest) (*pb.TweetResponse, error) {
    log.Printf("Received tweet: Description=%s, Country=%s, Weather=%s", req.Description, req.Country, req.Weather)

    // Conectar a grpc-server-rabbit
    conn, err := grpc.Dial("grpc-server-rabbit-service:5672", grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Printf("Error connecting to grpc-server-rabbit-service: %v", err)
        return nil, err
    }
    defer conn.Close()

    // Aquí iría la lógica para interactuar con grpc-server-rabbit
    // Por ejemplo, enviar el mensaje a RabbitMQ...

    return &pb.TweetResponse{Message: "Tweet publicado"}, nil
}

func main() {
    lis, err := net.Listen("tcp", ":50051")
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }
    s := grpc.NewServer()
    pb.RegisterTweetServiceServer(s, &server{})
    log.Println("gRPC client server running on :50051")
    if err := s.Serve(lis); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }
}
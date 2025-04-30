package main

import (
	"context"
	"log"
	"net"

	pb "tweets-clima/proto"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/streadway/amqp"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedTweetServiceServer
	rabbitConn    *amqp.Connection
	kafkaProducer *kafka.Producer
}

func (s *server) PublishTweet(ctx context.Context, req *pb.TweetRequest) (*pb.TweetResponse, error) {
	// Publicar en RabbitMQ
	rabbitCh, err := s.rabbitConn.Channel()
	if err != nil {
		return nil, err
	}
	defer rabbitCh.Close()
	err = rabbitCh.Publish(
		"", "message", false, false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        []byte(req.String()),
		},
	)
	if err != nil {
		return nil, err
	}

	// Publicar en Kafka
	err = s.kafkaProducer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: stringPtr("message"), Partition: kafka.PartitionAny},
		Value:          []byte(req.String()),
	}, nil)
	if err != nil {
		return nil, err
	}

	return &pb.TweetResponse{Message: "Tweet publicado"}, nil
}

func stringPtr(s string) *string {
	return &s
}

func main() {
	rabbitConn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	if err != nil {
		log.Fatalf("No se pudo conectar a RabbitMQ: %v", err)
	}
	defer rabbitConn.Close()

	kafkaProducer, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": "kafka:9092"})
	if err != nil {
		log.Fatalf("No se pudo conectar a Kafka: %v", err)
	}
	defer kafkaProducer.Close()

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("No se pudo escuchar en el puerto 50051: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterTweetServiceServer(s, &server{rabbitConn: rabbitConn, kafkaProducer: kafkaProducer})
	log.Fatal(s.Serve(lis))
}

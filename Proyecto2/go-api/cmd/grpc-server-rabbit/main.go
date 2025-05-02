package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"time"

	pb "tweets-clima/proto"

	"github.com/streadway/amqp"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedTweetServiceServer
	rabbitConn *amqp.Connection
}

func (s *server) PublishTweet(ctx context.Context, req *pb.TweetRequest) (*pb.TweetResponse, error) {
	description := req.GetDescription()
	country := req.GetCountry()
	weather := req.GetWeather()

	if description == "" {
		log.Println("El campo description está vacío")
		return nil, nil
	}
	log.Printf("Recibido tweet para publicar en RabbitMQ - Description: %s, Country: %s, Weather: %s", description, country, weather)

	rabbitCh, err := s.rabbitConn.Channel()
	if err != nil {
		log.Printf("Error al abrir canal de RabbitMQ: %v", err)
		return nil, err
	}
	defer rabbitCh.Close()

	tweetJSON, err := json.Marshal(req)
	if err != nil {
		log.Printf("Error al serializar el tweet a JSON: %v", err)
		return nil, err
	}

	err = rabbitCh.Publish(
		"",          // exchange
		"message",   // queue
		false,       // mandatory
		false,       // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        tweetJSON,
		},
	)
	if err != nil {
		log.Printf("Error al publicar en RabbitMQ: %v", err)
		return nil, err
	}

	log.Println("Tweet publicado en RabbitMQ con éxito")
	return &pb.TweetResponse{Message: "Tweet publicado en RabbitMQ"}, nil
}

func main() {
	var rabbitConn *amqp.Connection
	var err error

	for {
		rabbitConn, err = amqp.Dial("amqp://guest:guest@rabbitmq-service:5672/")
		if err != nil {
			log.Printf("No se pudo conectar a RabbitMQ, reintentando en 5s: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}
		log.Println("Conectado a RabbitMQ")
		break
	}
	defer rabbitConn.Close()

	var lis net.Listener
	for {
		lis, err = net.Listen("tcp", ":50052")
		if err != nil {
			log.Printf("No se pudo escuchar en el puerto 50052, reintentando en 5s: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}
		log.Println("Escuchando en el puerto 50052")
		break
	}

	s := grpc.NewServer()
	pb.RegisterTweetServiceServer(s, &server{rabbitConn: rabbitConn})
	log.Println("Iniciando servidor gRPC para RabbitMQ en :50052")
	log.Fatal(s.Serve(lis))
}
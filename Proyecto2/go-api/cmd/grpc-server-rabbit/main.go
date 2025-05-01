// package main

// import (
// 	"context"
// 	"log"
// 	"net"
// 	"time"

// 	pb "tweets-clima/proto"

// 	"github.com/streadway/amqp"
// 	"google.golang.org/grpc"
// )

// type server struct {
// 	pb.UnimplementedTweetServiceServer
// 	rabbitConn *amqp.Connection
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

// 	return &pb.TweetResponse{Message: "Tweet publicado en RabbitMQ"}, nil
// }

// func main() {
// 	var rabbitConn *amqp.Connection
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

// 	// Iniciar servidor gRPC
// 	var lis net.Listener
// 	for {
// 		lis, err = net.Listen("tcp", ":50052")
// 		if err != nil {
// 			log.Printf("No se pudo escuchar en el puerto 50052, reintentando en 5s: %v", err)
// 			time.Sleep(5 * time.Second)
// 			continue
// 		}
// 		log.Println("Escuchando en el puerto 50052")
// 		break
// 	}

// 	s := grpc.NewServer()
// 	pb.RegisterTweetServiceServer(s, &server{rabbitConn: rabbitConn})
// 	log.Println("Iniciando servidor gRPC para RabbitMQ en :50052")
// 	log.Fatal(s.Serve(lis))
// }




package main

import (
	"context"
	"log"
	"net"
	"time"

	pb "tweets-clima/proto"

	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedTweetServiceServer
}

func (s *server) PublishTweet(ctx context.Context, req *pb.TweetRequest) (*pb.TweetResponse, error) {
	// Acceder a los campos usando los getters generados por Protobuf
	description := req.GetDescription()
	country := req.GetCountry()
	weather := req.GetWeather()

	// Verificar y loguear los campos
	if description == "" {
		log.Println("El campo description está vacío")
	} else {
		log.Printf("Recibido tweet para publicar en RabbitMQ - Description: %s, Country: %s, Weather: %s", description, country, weather)
	}
	log.Println("Simulando publicación en RabbitMQ (sin conexión real)")
	return &pb.TweetResponse{Message: "Tweet publicado en RabbitMQ (simulado)"}, nil
}

func main() {
	// Iniciar servidor gRPC
	var lis net.Listener
	var err error
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
	pb.RegisterTweetServiceServer(s, &server{})
	log.Println("Iniciando servidor gRPC para RabbitMQ en :50052")
	log.Fatal(s.Serve(lis))
}
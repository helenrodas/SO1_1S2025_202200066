// package main

// import (
// 	"context"
// 	"encoding/json"
// 	"log"

// 	"github.com/confluentinc/confluent-kafka-go/kafka"
// 	"github.com/redis/go-redis/v9"
// )

// type Tweet struct {
// 	Description string `json:"description"`
// 	Country     string `json:"country"`
// 	Weather     string `json:"weather"`
// }

// func main() {
// 	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
// 		"bootstrap.servers": "kafka:9092",
// 		"group.id":          "kafka-consumer",
// 		"auto.offset.reset": "earliest",
// 	})
// 	if err != nil {
// 		log.Fatalf("No se pudo conectar a Kafka: %v", err)
// 	}
// 	defer consumer.Close()

// 	client := redis.NewClient(&redis.Options{Addr: "redis:6379"})
// 	defer client.Close()

// 	consumer.SubscribeTopics([]string{"message"}, nil)
// 	for {
// 		msg, err := consumer.ReadMessage(-1)
// 		if err != nil {
// 			log.Printf("Error al leer mensaje: %v", err)
// 			continue
// 		}

// 		var tweet Tweet
// 		if err := json.Unmarshal(msg.Value, &tweet); err != nil {
// 			log.Printf("Error al decodificar mensaje: %v", err)
// 			continue
// 		}

// 		client.HIncrBy(context.Background(), "tweets:countries", tweet.Country, 1)
// 		client.Incr(context.Background(), "tweets:total")
// 	}
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
		log.Printf("Recibido tweet para publicar en Kafka - Description: %s, Country: %s, Weather: %s", description, country, weather)
	}
	log.Println("Simulando publicación en Kafka (sin conexión real)")
	return &pb.TweetResponse{Message: "Tweet publicado en Kafka (simulado)"}, nil
}

func main() {
	// Iniciar servidor gRPC
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

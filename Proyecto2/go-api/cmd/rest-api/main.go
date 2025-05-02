package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	pb "tweets-clima/proto"

	"github.com/gorilla/mux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type TweetRequest struct {
	Description string `json:"description"`
	Country     string `json:"country"`
	Weather     string `json:"weather"`
}

type TweetResponse struct {
	Message string `json:"message"`
}

func publishTweet(w http.ResponseWriter, r *http.Request) {
	log.Println("Paso 1: Solicitud recibida en /input")

	var req TweetRequest
	log.Println("Paso 2: Intentando decodificar el payload de la solicitud")
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Paso 2 - Error: Error al decodificar el payload de la solicitud: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	log.Printf("Paso 2 - Éxito: Solicitud decodificada: Description=%s, Country=%s, Weather=%s", req.Description, req.Country, req.Weather)

	// Esperar a que los servicios estén listos
	log.Println("Paso 3: Esperando a que los servicios estén listos")
	time.Sleep(5 * time.Second)

	// Conectar a grpc-server-rabbit
	log.Println("Paso 4: Intentando conectar a grpc-server-rabbit")
	rabbitConn, err := grpc.Dial("grpc-server-rabbit-service:50052", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock(), grpc.WithTimeout(10*time.Second))
	if err != nil {
		log.Printf("Paso 4 - Error: No se pudo conectar a grpc-server-rabbit: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rabbitConn.Close()
	rabbitClient := pb.NewTweetServiceClient(rabbitConn)

	// Conectar a grpc-server-kafka
	log.Println("Paso 5: Intentando conectar a grpc-server-kafka")
	kafkaConn, err := grpc.Dial("grpc-server-kafka-service:50053", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock(), grpc.WithTimeout(10*time.Second))
	if err != nil {
		log.Printf("Paso 5 - Error: No se pudo conectar a grpc-server-kafka: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer kafkaConn.Close()
	kafkaClient := pb.NewTweetServiceClient(kafkaConn)

	// Enviar solicitud a grpc-server-rabbit
	log.Println("Paso 6: Enviando solicitud gRPC a grpc-server-rabbit")
	rabbitResp, err := rabbitClient.PublishTweet(context.Background(), &pb.TweetRequest{
		Description: req.Description,
		Country:     req.Country,
		Weather:     req.Weather,
	})
	if err != nil {
		log.Printf("Paso 6 - Error: Error al enviar solicitud a RabbitMQ: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Enviar solicitud a grpc-server-kafka
	log.Println("Paso 7: Enviando solicitud gRPC a grpc-server-kafka")
	kafkaResp, err := kafkaClient.PublishTweet(context.Background(), &pb.TweetRequest{
		Description: req.Description,
		Country:     req.Country,
		Weather:     req.Weather,
	})
	if err != nil {
		log.Printf("Paso 7 - Error: Error al enviar solicitud a Kafka: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Responder al cliente con el resultado combinado
	log.Println("Paso 8: Publicación exitosa en RabbitMQ y Kafka")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(TweetResponse{Message: "Tweet publicado en RabbitMQ y Kafka: " + rabbitResp.Message + ", " + kafkaResp.Message})
	log.Println("Paso 9: Respuesta enviada al cliente")
}

func main() {
	r := mux.NewRouter()
	log.Println("Iniciando servidor RESTTTTTT")
	r.HandleFunc("/input", publishTweet).Methods("POST")

	var srv *http.Server
	var err error
	for {
		srv = &http.Server{
			Addr:    ":8080",
			Handler: r,
		}
		log.Println("Iniciando servidor REST en :8080")
		err = srv.ListenAndServe()
		if err != nil {
			log.Printf("No se pudo iniciar el servidor REST, reintentando en 5s: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}
		break
	}
}
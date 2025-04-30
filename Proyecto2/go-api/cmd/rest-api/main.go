package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	pb "tweets-clima/proto"

	"github.com/gorilla/mux"
	"google.golang.org/grpc"
)

type Tweet struct {
	Description string `json:"description"`
	Country     string `json:"country"`
	Weather     string `json:"weather"`
}

func main() {
	conn, err := grpc.Dial("grpc-client-service:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("No se pudo conectar a gRPC: %v", err)
	}
	defer conn.Close()
	client := pb.NewTweetServiceClient(conn)

	router := mux.NewRouter()
	router.HandleFunc("/tweet", func(w http.ResponseWriter, r *http.Request) {
		var tweet Tweet
		if err := json.NewDecoder(r.Body).Decode(&tweet); err != nil {
			http.Error(w, "Error al decodificar JSON", http.StatusBadRequest)
			return
		}

		req := &pb.TweetRequest{
			Description: tweet.Description,
			Country:     tweet.Country,
			Weather:     tweet.Weather,
		}
		resp, err := client.PublishTweet(context.Background(), req)
		if err != nil {
			http.Error(w, "Error al procesar tweet", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"message": resp.Message})
	}).Methods("POST")

	log.Fatal(http.ListenAndServe(":8080", router))
}

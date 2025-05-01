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

    // Conectar a grpc-client
    log.Println("Paso 3: Intentando conectar a grpc-client")
    var conn *grpc.ClientConn
    var err error
    for {
        conn, err = grpc.Dial("grpc-client:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
        if err != nil {
            log.Printf("Paso 3 - Error: No se pudo conectar a grpc-client, reintentando en 5s: %v", err)
            time.Sleep(5 * time.Second)
            continue
        }
        log.Println("Paso 3 - Éxito: Conectado a grpc-client")
        break
    }
    defer conn.Close()

    client := pb.NewTweetServiceClient(conn)
    log.Println("Paso 4: Enviando solicitud gRPC a grpc-client:50051")

    // Enviar solicitud gRPC
    log.Println("Paso 5: Intentando enviar solicitud gRPC")
    resp, err := client.PublishTweet(context.Background(), &pb.TweetRequest{
        Description: req.Description,
        Country:     req.Country,
        Weather:     req.Weather,
    })
    if err != nil {
        log.Printf("Paso 5 - Error: Error al enviar solicitud gRPC: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    log.Println("Paso 5 - Éxito: Solicitud gRPC enviada con éxito")

    // Responder al cliente
    log.Println("Paso 6: Tweet publicado con éxito")
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(TweetResponse{Message: resp.Message})
    log.Println("Paso 7: Respuesta enviada al cliente")
}

func main() {
    r := mux.NewRouter()
    log.Println("Iniciando servidor RESTTTTTT")
    r.HandleFunc("/input", publishTweet).Methods("POST")

    var srv *http.Server
    var err error
    for {
        srv = &http.Server{
            Addr:    ":8080", // Cambiado a 8081 para coincidir con rust-api
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
package main

import (
	"context"
	"log"
	"net/http"
	"time"

	redisclient "waitingroom/admissionservice/redis"

	_ "embed"

	"waitingroom/admissionservice/routes"

	"waitingroom/admissionservice/services"

	"github.com/go-chi/chi/v5"
)

//go:embed scripts/waitingroomdecisionworkflow.lua
var redisFunctionString string

func main() {
	waitingRoomService, err := services.NewWaitingRoomService(context.Background(), services.WaitingRoomServiceConfig{CacheTTL: 1 * time.Minute})
	if err != nil {
		log.Printf("Failed to initialize waiting room service: %v", err)
		return
	}

	r := chi.NewRouter()

	err = routes.RegisterRoutes(r, waitingRoomService)
	if err != nil {
		log.Fatalf("Failed to register routes: %v", err)
	}

	redisClientConfig := redisclient.RedisClientConfig{Address: "localhost:6379", Username: "", Password: "", Protocol: 2}
	redisclient.GetRedisClient(redisClientConfig)
	defer redisclient.Close()
	response, err := redisclient.LoadRedisLibrary(redisclient.LoadRedisLibraryRequest{SourceCode: redisFunctionString})
	if err != nil {
		log.Fatalf("Failed to load redis library. Quitting: %v", err)
	}
	log.Printf("Successfully loaded Redis function: %v", response.Response)

	log.Printf("Admission Service listening on port 3333....")

	err = http.ListenAndServe(":3333", r)
	if err != nil {
		log.Fatalf("Server failed to listen: %v", err)
	}
}

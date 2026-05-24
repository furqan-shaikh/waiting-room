package main

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"waitingroom/controlplane/routes"
	"waitingroom/controlplane/services"
)

func main() {
	r := chi.NewRouter()

	waitingRoomService, err := services.NewWaitingRoomService(context.Background())
	if err != nil {
		log.Printf("Failed to initialize waiting room service: %v", err)
		return
	}

	err = routes.RegisterRoutes(r, waitingRoomService)
	if err != nil {
		log.Fatalf("Failed to register routes: %v", err)
	}

	log.Printf("Control Plane Service listening on port 3000....")

	err = http.ListenAndServe(":3000", r)
	if err != nil {
		log.Fatalf("Server failed to listen: %v", err)
	}

}

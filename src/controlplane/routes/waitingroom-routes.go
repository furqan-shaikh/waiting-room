package routes

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"waitingroom/controlplane/services"
	"waitingroom/shared/models"
)

var (
	waitingRoomService *services.WaitingRoomService
)

func RegisterRoutes(r chi.Router, service *services.WaitingRoomService) error {
	r.Post("/waitingRooms", createWaitingRoom)
	r.Get("/waitingRooms/{roomId}", getWaitingRoom)
	r.Delete("/waitingRooms/{roomId}", deleteWaitingRoom)

	waitingRoomService = service

	return nil
}

func createWaitingRoom(w http.ResponseWriter, r *http.Request) {
	var createWaitingRoomRequest models.CreateWaitingRoomRequest
	log.Printf("Received createWaitingRoom request")
	err := json.NewDecoder(r.Body).Decode(&createWaitingRoomRequest)
	if err != nil {
		log.Printf("Failed to decode request json: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	waitingRoom, err := waitingRoomService.CreateWaitingRoom(r.Context(), createWaitingRoomRequest)
	if err != nil {
		if validationError, ok := errors.AsType[*models.ValidationError](err); ok {
			log.Printf("Validation Errors in creating waiting room: %v", validationError)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		log.Printf("Error in creating waiting room: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	log.Printf("Successfully created waiting room: %v", waitingRoom.RoomId)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(waitingRoom)
}

func getWaitingRoom(w http.ResponseWriter, r *http.Request) {
	roomId := chi.URLParam(r, "roomId")
	waitingRoom, err := waitingRoomService.GetWaitingRoom(r.Context(), models.GetWaitingRoomRequest{RoomId: roomId})

	if err != nil {
		if notFoundError, ok := errors.AsType[*models.NotFoundError](err); ok {
			log.Printf("Waiting Room %s not found: %v", roomId, notFoundError)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		log.Printf("Error in fetching waiting room: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(waitingRoom)
}

func deleteWaitingRoom(w http.ResponseWriter, r *http.Request) {
	roomId := chi.URLParam(r, "roomId")
	_, err := waitingRoomService.DeleteWaitingRoom(r.Context(), models.DeleteWaitingRoomRequest{RoomId: roomId, IsSoftDelete: true})
	if err != nil {
		log.Printf("Error in deleting waiting room: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

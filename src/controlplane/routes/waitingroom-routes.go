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
		handleErrorResponse(w, "Failed to decode request json", http.StatusBadRequest, models.BadRequestCode, []models.ResponseErrorDetailItem{})
		return
	}

	log.Printf("CreateWaitingRoom: %v", createWaitingRoomRequest)
	waitingRoom, err := waitingRoomService.CreateWaitingRoom(r.Context(), createWaitingRoomRequest)
	if err != nil {
		if validationError, ok := errors.AsType[*models.ValidationError](err); ok {
			log.Printf("Validation Errors in creating waiting room: %v", validationError)
			messages := validationError.Messages
			responseMessages := []models.ResponseErrorDetailItem{}
			for i := 0; i < len(messages); i++ {
				responseMessages = append(responseMessages, models.ResponseErrorDetailItem{
					Field:   messages[i].Field,
					Message: messages[i].Message,
				})
			}
			handleErrorResponse(w, "Validation failed for request", http.StatusBadRequest, models.BadRequestCode, responseMessages)
			return
		}
		log.Printf("Error in creating waiting room: %v", err)
		handleErrorResponse(w, "Error in creating waiting room", http.StatusInternalServerError, models.InternalServerErrorCode, []models.ResponseErrorDetailItem{})
		return
	}
	log.Printf("Successfully created waiting room: %v", waitingRoom.RoomId)
	w.Header().Set("Content-Type", "application/json")
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

func handleErrorResponse(w http.ResponseWriter, message string, statusCode int, errorCode string, details []models.ResponseErrorDetailItem) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.Encode(models.ResponseError{
		Code:    errorCode,
		Message: message,
		Details: details,
	})
}

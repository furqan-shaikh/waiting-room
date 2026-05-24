package routes

import (
	"encoding/json"
	"log"
	"net/http"

	redisclient "waitingroom/admissionservice/redis"
	"waitingroom/admissionservice/services"
	"waitingroom/shared/models"

	"github.com/go-chi/chi/v5"
)

var (
	waitingRoomService *services.WaitingRoomService
)

func RegisterWaitingRoomRoutes(r chi.Router, service *services.WaitingRoomService) error {
	r.Get("/{roomId}/status", getStatus)
	waitingRoomService = service
	return nil
}

func getStatus(w http.ResponseWriter, r *http.Request) {
	roomId := chi.URLParam(r, "roomId")
	sessionToken, err := services.HandleSessionToken(w, r)
	if err != nil {
		log.Printf("Error in handling session token: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var waitingRoom models.WaitingRoom
	waitingRoom, err = waitingRoomService.GetWaitingRoom(r.Context(), models.GetWaitingRoomRequest{RoomId: roomId})
	if err != nil {
		log.Printf("Failed to fetch waiting room config: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	maxActiveUserCount := waitingRoom.MaxActiveUsersCount

	ttlInSeconds := 5
	origin := waitingRoom.OriginApplication

	redisFunctionResponse, err := redisclient.InvokeRedisLibrary(redisclient.RedisFunctionRequest{
		RoomId:             roomId,
		MaxActiveUserCount: maxActiveUserCount,
		SessionToken:       sessionToken,
		TTLInSeconds:       ttlInSeconds,
	})
	if err != nil {
		log.Printf("Error in invoking redis function: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]string{"roomId": roomId, "decision": redisFunctionResponse.Decision, "origin": origin}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

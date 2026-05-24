package routes

import (
	_ "embed"
	"log"
	"net/http"

	"waitingroom/admissionservice/services"

	"github.com/go-chi/chi/v5"
)

//go:embed static/waitingroom.html
var waitingRoomAppHtml string

//go:embed static/waitingroom.js
var waitingRoomAppJS string

func RegisterStaticRoutes(r chi.Router) error {
	r.Get("/assets/waitingroom.js", getWaitingRoomAppJS)
	r.Get("/{roomId}", getWaitingRoomApp)

	return nil
}

func getWaitingRoomApp(w http.ResponseWriter, r *http.Request) {
	_, err := services.HandleSessionToken(w, r)
	if err != nil {
		log.Printf("Error in handling session token: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(waitingRoomAppHtml))
}

func getWaitingRoomAppJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Write([]byte(waitingRoomAppJS))
}

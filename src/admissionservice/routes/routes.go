package routes

import (
	"waitingroom/admissionservice/services"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, service *services.WaitingRoomService) error {
	r.Route("/waitingRooms", func(r chi.Router) {
		RegisterStaticRoutes(r)
		RegisterWaitingRoomRoutes(r, service)
	})
	return nil
}

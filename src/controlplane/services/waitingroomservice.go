package services

import (
	"context"
	"log"
	"time"

	"waitingroom/controlplane/authn"
	"waitingroom/shared/models"
	"waitingroom/shared/pg"

	"github.com/google/uuid"
)

type WaitingRoomService struct {
	pgWaitingRoomRepository *pg.PgWaitingRoomRepository
}

func NewWaitingRoomService(ctx context.Context) (*WaitingRoomService, error) {
	pgWaitingRoomRepository, err := pg.NewPgWaitingRoomRepository()
	if err != nil {
		log.Printf("Failed to initialize NewPgWaitingRoomRepository: %v", err)
		return nil, err
	}
	err = pgWaitingRoomRepository.Ping(ctx)
	if err != nil {
		log.Printf("Failed to initialize NewPgWaitingRoomRepository as Ping call failed: %v", err)
		return nil, err
	}
	// check if the table schema exists
	_, err = pgWaitingRoomRepository.SchemaExists(ctx)
	if err != nil {
		log.Printf("%v. Run migrations script to setup waitingrooms schema", err)
		return nil, err
	}
	log.Printf("Successfully created Waiting Room Service")
	return &WaitingRoomService{pgWaitingRoomRepository: pgWaitingRoomRepository}, nil
}

func (svc *WaitingRoomService) CreateWaitingRoom(ctx context.Context, request models.CreateWaitingRoomRequest, userPrincipal authn.UserPrincipal) (models.WaitingRoom, error) {
	// Validate the request
	validationMessages := validateWaitingRoomRequest(request)
	if len(validationMessages) > 0 {
		return models.WaitingRoom{}, &models.ValidationError{Messages: validationMessages}
	}
	log.Printf("Successfully validated the WaitingRoomRequest")
	waitingRoom := getWaitingRoomEntity(request, userPrincipal)
	_, err := svc.pgWaitingRoomRepository.CreateWaitingRoom(ctx, waitingRoom)
	if err != nil {
		log.Printf("Failed to create waiting room: %v", err)
		return models.WaitingRoom{}, err
	}
	return waitingRoom, nil
}

func getWaitingRoomEntity(request models.CreateWaitingRoomRequest, userPrincipal authn.UserPrincipal) models.WaitingRoom {
	room_id := uuid.New().String()
	nowUTC := time.Now().UTC()
	activeSessionTtlInSeconds := models.DefaultActiveSessionTtlInSeconds
	if request.ActiveSessionTtlSeconds != 0 {
		activeSessionTtlInSeconds = request.ActiveSessionTtlSeconds
	}

	waitingSessionTtlInSeconds := models.DefaultWaitingSessionTtlInSeconds
	if request.WaitingSessionTtlSeconds != 0 {
		waitingSessionTtlInSeconds = request.WaitingSessionTtlSeconds
	}

	pollingIntervalSeconds := models.DefaultPollingIntervalInSeconds
	if request.PollingIntervalSeconds != 0 {
		pollingIntervalSeconds = request.PollingIntervalSeconds
	}

	return models.WaitingRoom{
		RoomId:                   room_id,
		CreatedAt:                nowUTC,
		UpdatedAt:                nowUTC,
		MaxActiveUsersCount:      request.MaxActiveUsersCount,
		OriginApplication:        request.OriginApplication,
		Status:                   models.StatusActive,
		ActiveSessionTtlSeconds:  activeSessionTtlInSeconds,
		WaitingSessionTtlSeconds: waitingSessionTtlInSeconds,
		PollingIntervalSeconds:   pollingIntervalSeconds,
		OwnerId:                  userPrincipal.Id,
	}
}

func (svc *WaitingRoomService) GetWaitingRoom(ctx context.Context, request models.GetWaitingRoomRequest, userPrincipal authn.UserPrincipal) (models.WaitingRoom, error) {
	waitingRoom, err := svc.pgWaitingRoomRepository.GetWaitingRoom(ctx, request, userPrincipal.Id)
	if err != nil {
		log.Printf("Failed to get waiting room: %v", err)
		return models.WaitingRoom{}, err
	}
	activeSessionTtlInSeconds := models.DefaultActiveSessionTtlInSeconds
	if waitingRoom.ActiveSessionTtlSeconds != 0 {
		activeSessionTtlInSeconds = waitingRoom.ActiveSessionTtlSeconds
	}

	waitingSessionTtlInSeconds := models.DefaultWaitingSessionTtlInSeconds
	if waitingRoom.WaitingSessionTtlSeconds != 0 {
		waitingSessionTtlInSeconds = waitingRoom.WaitingSessionTtlSeconds
	}

	pollingIntervalSeconds := models.DefaultPollingIntervalInSeconds
	if waitingRoom.PollingIntervalSeconds != 0 {
		pollingIntervalSeconds = waitingRoom.PollingIntervalSeconds
	}
	newWaitingRoomModel := models.WaitingRoom{
		RoomId:                   waitingRoom.RoomId,
		CreatedAt:                waitingRoom.CreatedAt,
		UpdatedAt:                waitingRoom.UpdatedAt,
		MaxActiveUsersCount:      waitingRoom.MaxActiveUsersCount,
		Status:                   waitingRoom.Status,
		OriginApplication:        waitingRoom.OriginApplication,
		ActiveSessionTtlSeconds:  activeSessionTtlInSeconds,
		WaitingSessionTtlSeconds: waitingSessionTtlInSeconds,
		PollingIntervalSeconds:   pollingIntervalSeconds,
		OwnerId:                  userPrincipal.Id,
	}
	return newWaitingRoomModel, nil
}

func (svc *WaitingRoomService) DeleteWaitingRoom(ctx context.Context, request models.DeleteWaitingRoomRequest, userPrincipal authn.UserPrincipal) (bool, error) {
	status, err := svc.pgWaitingRoomRepository.DeleteWaitingRoom(ctx, request, userPrincipal.Id)
	if err != nil {
		log.Printf("Failed to delete waiting room: %v", err)
		return status, err
	}
	return status, nil
}

func validateWaitingRoomRequest(request models.CreateWaitingRoomRequest) []models.ValidationErrorItem {
	messages := []models.ValidationErrorItem{}
	if request.MaxActiveUsersCount <= 0 {
		messages = append(messages, models.ValidationErrorItem{
			Field:   "maxActiveUsersCount",
			Message: "maxActiveUsersCount must be greater than 0",
		})
	}
	if request.OriginApplication == "" {
		messages = append(messages, models.ValidationErrorItem{
			Field:   "originApplication",
			Message: "originApplication is a required field",
		})
	}
	if request.ActiveSessionTtlSeconds < 0 {
		messages = append(messages, models.ValidationErrorItem{
			Field:   "activeSessionTtlSeconds",
			Message: "activeSessionTtlSeconds must be greater than 0 when provided",
		})
	}
	if request.WaitingSessionTtlSeconds < 0 {
		messages = append(messages, models.ValidationErrorItem{
			Field:   "waitingSessionTtlSeconds",
			Message: "waitingSessionTtlSeconds must be greater than 0 when provided",
		})
	}

	if request.PollingIntervalSeconds < 0 {
		messages = append(messages, models.ValidationErrorItem{
			Field:   "pollingIntervalSeconds",
			Message: "pollingIntervalSeconds must be greater than 0 when provided",
		})
	}
	return messages
}

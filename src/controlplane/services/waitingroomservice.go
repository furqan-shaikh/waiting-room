package services

import (
	"context"
	"log"
	"time"

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

func (svc *WaitingRoomService) CreateWaitingRoom(ctx context.Context, request models.CreateWaitingRoomRequest) (models.WaitingRoom, error) {
	// Validate the request

	validationMessages := validateWaitingRoomRequest(request)
	if len(validationMessages) > 0 {
		return models.WaitingRoom{}, &models.ValidationError{Messages: validationMessages}
	}
	log.Printf("Successfully validated the WaitingRoomRequest")
	room_id := uuid.New().String()
	nowUTC := time.Now().UTC()

	waitingRoom := models.WaitingRoom{
		RoomId:              room_id,
		CreatedAt:           nowUTC,
		UpdatedAt:           nowUTC,
		MaxActiveUsersCount: request.MaxActiveUsersCount,
		OriginApplication:   request.OriginApplication,
		Status:              models.StatusActive,
	}
	_, err := svc.pgWaitingRoomRepository.CreateWaitingRoom(ctx, waitingRoom)
	if err != nil {
		log.Printf("Failed to create waiting room: %v", err)
		return models.WaitingRoom{}, err
	}
	return waitingRoom, nil
}

func (svc *WaitingRoomService) GetWaitingRoom(ctx context.Context, request models.GetWaitingRoomRequest) (models.WaitingRoom, error) {
	waitingRoom, err := svc.pgWaitingRoomRepository.GetWaitingRoom(ctx, request)
	if err != nil {
		log.Printf("Failed to get waiting room: %v", err)
		return models.WaitingRoom{}, err
	}
	return waitingRoom, nil
}

func (svc *WaitingRoomService) DeleteWaitingRoom(ctx context.Context, request models.DeleteWaitingRoomRequest) (bool, error) {
	status, err := svc.pgWaitingRoomRepository.DeleteWaitingRoom(ctx, request)
	if err != nil {
		log.Printf("Failed to delete waiting room: %v", err)
		return status, err
	}
	return status, nil
}

func validateWaitingRoomRequest(request models.CreateWaitingRoomRequest) []string {
	messages := []string{}
	if request.MaxActiveUsersCount <= 0 {
		messages = append(messages, "MaxActiveUsersCount must be > 0")
	}
	if request.OriginApplication == "" {
		messages = append(messages, "Origin Application is a required field")
	}
	return messages
}

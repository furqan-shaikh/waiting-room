package services

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"waitingroom/shared/models"
	"waitingroom/shared/pg"

	"waitingroom/admissionservice/cache"
)

type WaitingRoomServiceConfig struct {
	CacheTTL time.Duration
}
type WaitingRoomService struct {
	pgWaitingRoomRepository *pg.PgWaitingRoomRepository
	roomCacheManager        *cache.RoomCacheManager
}

func NewWaitingRoomService(ctx context.Context, config WaitingRoomServiceConfig) (*WaitingRoomService, error) {
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

	roomCacheManager := cache.NewRoomCacheManager(cache.RoomCacheManagerConfig{
		TTLInMinutes: config.CacheTTL,
	})

	log.Printf("Successfully created Waiting Room Service")
	return &WaitingRoomService{pgWaitingRoomRepository: pgWaitingRoomRepository, roomCacheManager: roomCacheManager}, nil
}

func (svc *WaitingRoomService) GetWaitingRoom(ctx context.Context, request models.GetWaitingRoomRequest) (models.WaitingRoom, error) {
	// Pattern is as below:
	// 1. Fetch the room config from cache
	// 2. If present, return
	// 3. If not present, fetch from database
	// 4. Set in cache
	// 5. return
	var waitingRoomFromDb models.WaitingRoom

	// 1. Fetch the room config from cache
	roomConfigFromCacheString, err := svc.roomCacheManager.Get(request.RoomId)
	if err == nil {
		// 2. If present, return
		var roomConfigFromCache models.WaitingRoom
		if err := json.Unmarshal([]byte(roomConfigFromCacheString), &roomConfigFromCache); err != nil {
			log.Printf("Failed to decode waiting room config from cache: %v", err)
		} else {
			return roomConfigFromCache, nil
		}
	}

	// 3. If not present, fetch from database
	waitingRoomFromDb, err = svc.pgWaitingRoomRepository.GetWaitingRoom(ctx, request)
	if err != nil {
		log.Printf("Failed to get waiting room: %v", err)
		return models.WaitingRoom{}, err
	}

	// 4. Set in cache
	waitingRoomB, marshalErr := json.Marshal(waitingRoomFromDb)
	if marshalErr == nil {
		svc.roomCacheManager.Set(request.RoomId, string(waitingRoomB))
	} else {
		log.Printf("Failed to encode waiting room config for cache: %v", marshalErr)
	}

	// 5. return
	return waitingRoomFromDb, nil
}

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
			if roomConfigFromCache.ActiveSessionTtlSeconds == 0 {
				roomConfigFromCache.ActiveSessionTtlSeconds = models.DefaultActiveSessionTtlInSeconds
			}

			if roomConfigFromCache.WaitingSessionTtlSeconds == 0 {
				roomConfigFromCache.WaitingSessionTtlSeconds = models.DefaultWaitingSessionTtlInSeconds
			}

			if roomConfigFromCache.PollingIntervalSeconds == 0 {
				roomConfigFromCache.PollingIntervalSeconds = models.DefaultPollingIntervalInSeconds
			}
			return roomConfigFromCache, nil
		}
	}

	// 3. If not present, fetch from database
	waitingRoomFromDb, err = svc.pgWaitingRoomRepository.GetWaitingRoom(ctx, request)
	if err != nil {
		log.Printf("Failed to get waiting room: %v", err)
		return models.WaitingRoom{}, err
	}
	activeSessionTtlInSeconds := models.DefaultActiveSessionTtlInSeconds
	if waitingRoomFromDb.ActiveSessionTtlSeconds != 0 {
		activeSessionTtlInSeconds = waitingRoomFromDb.ActiveSessionTtlSeconds
	}

	waitingSessionTtlInSeconds := models.DefaultWaitingSessionTtlInSeconds
	if waitingRoomFromDb.WaitingSessionTtlSeconds != 0 {
		waitingSessionTtlInSeconds = waitingRoomFromDb.WaitingSessionTtlSeconds
	}

	pollingIntervalSeconds := models.DefaultPollingIntervalInSeconds
	if waitingRoomFromDb.PollingIntervalSeconds != 0 {
		pollingIntervalSeconds = waitingRoomFromDb.PollingIntervalSeconds
	}
	newWaitingRoomFromDb := models.WaitingRoom{
		RoomId:                   waitingRoomFromDb.RoomId,
		CreatedAt:                waitingRoomFromDb.CreatedAt,
		UpdatedAt:                waitingRoomFromDb.UpdatedAt,
		MaxActiveUsersCount:      waitingRoomFromDb.MaxActiveUsersCount,
		Status:                   waitingRoomFromDb.Status,
		OriginApplication:        waitingRoomFromDb.OriginApplication,
		ActiveSessionTtlSeconds:  activeSessionTtlInSeconds,
		WaitingSessionTtlSeconds: waitingSessionTtlInSeconds,
		PollingIntervalSeconds:   pollingIntervalSeconds,
	}

	// 4. Set in cache
	waitingRoomB, marshalErr := json.Marshal(newWaitingRoomFromDb)
	if marshalErr == nil {
		svc.roomCacheManager.Set(request.RoomId, string(waitingRoomB))
	} else {
		log.Printf("Failed to encode waiting room config for cache: %v", marshalErr)
	}

	// 5. return
	return newWaitingRoomFromDb, nil
}

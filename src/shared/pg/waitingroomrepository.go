package pg

import (
	"context"
	"waitingroom/shared/models"
)

type WaitingRoomRepository interface {
	CreateWaitingRoom(ctx context.Context, request models.WaitingRoom) (bool, error)
	GetWaitingRoom(ctx context.Context, request models.GetWaitingRoomRequest) (models.WaitingRoom, error)
	DeleteWaitingRoom(ctx context.Context, request models.DeleteWaitingRoomRequest) (bool, error)
	Ping(ctx context.Context) error
	Close()
}

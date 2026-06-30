package pg

import (
	"context"
	"waitingroom/shared/models"
)

type WaitingRoomRepository interface {
	CreateWaitingRoom(ctx context.Context, request models.WaitingRoom) (bool, error)
	GetWaitingRoom(ctx context.Context, request models.GetWaitingRoomRequest, userPrincipal string) (models.WaitingRoom, error)
	DeleteWaitingRoom(ctx context.Context, request models.DeleteWaitingRoomRequest, userPrincipal string) (bool, error)
	Ping(ctx context.Context) error
	SchemaExists(ctx context.Context) (bool, error)
	Close()
}

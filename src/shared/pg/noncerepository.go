package pg

import (
	"context"
	"waitingroom/shared/models"
)

type NonceRepository interface {
	TryUseNonce(ctx context.Context, request models.Nonce) (bool, error)
}

package pg

import (
	"context"
	"errors"
	"log"
	"waitingroom/shared/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PgNonceRepository struct {
	pgConnectionPool *pgxpool.Pool
}

func NewPgNonceRepository() (*PgNonceRepository, error) {
	pgConnectionPool = GetPostgreslient()
	if pgConnectionPool == nil {
		message := "Unable to create Pg Nonce Repository"
		log.Printf(message)
		return nil, errors.New(message)
	}
	log.Printf("Successfully created Pg Nonce Repository")
	return &PgNonceRepository{pgConnectionPool: pgConnectionPool}, nil
}

func (pgNonceRepository *PgNonceRepository) TryUseNonce(ctx context.Context, request models.Nonce) (bool, error) {
	if pgNonceRepository.pgConnectionPool == nil {
		return false, errors.New("Call NewPgNonceRepository before invoking repository methods")
	}
	// construct insert parameterized query
	query := `INSERT INTO nonces (key_id, nonce_value, created_at) VALUES (@key_id, @nonce_value, @created_at)`
	args := pgx.NamedArgs{
		"key_id":      request.KeyId,
		"nonce_value": request.NonceValue,
		"created_at":  request.CreatedAt,
	}
	log.Printf("Inserting nonce into pg table: %v", request.KeyId)
	_, err := pgNonceRepository.pgConnectionPool.Exec(ctx, query, args)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			// 23505 is the unique_violation error code in PostgreSQL
			if pgErr.Code == "23505" {
				return false, errors.New("Nonce insertion failed as duplicate row error caught: nonce already exists")
			}
		}
		log.Printf("Unable to insert row in nonces: %v", err)
		return false, err
	}
	log.Printf("Successfully inserted nonce into pg table: %v", request.KeyId)
	return true, nil
}

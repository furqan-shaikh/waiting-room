package dao

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"waitingroom/shared/models"
	"waitingroom/shared/pg"
)

type PgWaitingRoomRepositoryDp struct {
	pgConnectionPool *pgxpool.Pool
}

func NewPgWaitingRoomRepositoryDp() (*PgWaitingRoomRepositoryDp, error) {
	connectionPool := pg.GetPostgreslient()
	if connectionPool == nil {
		message := "Unable to create Pg WaitingRoom Repository"
		log.Printf(message)
		return nil, errors.New(message)
	}
	log.Printf("Successfully created Pg WaitingRoom Repository")
	return &PgWaitingRoomRepositoryDp{pgConnectionPool: connectionPool}, nil
}

func (pg *PgWaitingRoomRepositoryDp) Ping(ctx context.Context) error {
	return pg.pgConnectionPool.Ping(ctx)
}

func (pgrepository *PgWaitingRoomRepositoryDp) SchemaExists(ctx context.Context) (bool, error) {
	query := `SELECT TO_REGCLASS('public.waitingrooms') IS NOT NULL`
	var exists bool
	err := pgrepository.pgConnectionPool.QueryRow(ctx, query).Scan(&exists)

	if err != nil {
		log.Printf("Failed to validate db schema: %v", err)
		return false, err
	}
	if !exists {
		return false, errors.New("waitingrooms schema doesn't exist")
	}
	return exists, nil
}

func (pgrepository *PgWaitingRoomRepositoryDp) GetWaitingRoom(ctx context.Context, request models.GetWaitingRoomRequest) (models.WaitingRoom, error) {
	if pgrepository.pgConnectionPool == nil {
		return models.WaitingRoom{}, errors.New("Call NewRepository before invoking repository methods")
	}

	// since the 2 ttl columns can have null values, we need to return 0 else scanning NULL into int would fail.
	// The COALESCE() function accepts a list of arguments and returns the first non-null argument.
	query := `SELECT room_id,
					created_at,
  					updated_at,
  					max_active_users_count,
  					origin_application,
  					status, 
					COALESCE(active_session_ttl_seconds, 0) AS active_session_ttl_seconds,
					COALESCE(waiting_session_ttl_seconds, 0) AS waiting_session_ttl_seconds,
					COALESCE(polling_interval_seconds, 0) AS polling_interval_seconds
			  FROM waitingrooms WHERE room_id = @room_id AND status = @status`
	args := pgx.NamedArgs{
		"room_id": request.RoomId,
		"status":  models.StatusActive,
	}

	rows, err := pgrepository.pgConnectionPool.Query(ctx, query, args)
	if err != nil {
		log.Printf("Failed to get waiting room: %v", err)
		return models.WaitingRoom{}, err
	}
	waitingRoom, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.WaitingRoom])
	if errors.Is(err, pgx.ErrNoRows) {
		return models.WaitingRoom{}, &models.NotFoundError{Message: fmt.Sprintf("Waiting Room %s not found", request.RoomId)}
	}

	if err != nil {
		log.Printf("Failed to get waiting room: %v", err)
		return models.WaitingRoom{}, err
	}
	return waitingRoom, nil
}

func (pg *PgWaitingRoomRepositoryDp) Close() {
	pg.pgConnectionPool.Close()
}

package pg

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"waitingroom/shared/models"
)

type PgWaitingRoomRepository struct {
	pgConnectionPool *pgxpool.Pool
}

func NewPgWaitingRoomRepository() (*PgWaitingRoomRepository, error) {
	// databaseUrl := os.Getenv("PG_DATABASE_URL")
	// if databaseUrl == "" {
	// 	errorStr := "Environment Variable PG_DATABASE_URL is not set"
	// 	log.Printf("%s", errorStr)
	// 	return nil, errors.New(errorStr)
	// }
	// pgConnectionPool, err := pgxpool.New(context.Background(), databaseUrl)
	// if err != nil {
	// 	log.Printf("Unable to create connection pool: %v\n", err)
	// 	return nil, err
	// }
	pgConnectionPool = GetPostgreslient()
	if pgConnectionPool == nil {
		message := "Unable to create Pg WaitingRoom Repository"
		log.Printf(message)
		return nil, errors.New(message)
	}
	log.Printf("Successfully created Pg WaitingRoom Repository")
	return &PgWaitingRoomRepository{pgConnectionPool: pgConnectionPool}, nil
}

func (pg *PgWaitingRoomRepository) Ping(ctx context.Context) error {
	return pg.pgConnectionPool.Ping(ctx)
}

func (pgrepository *PgWaitingRoomRepository) SchemaExists(ctx context.Context) (bool, error) {
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

func (pgrepository *PgWaitingRoomRepository) CreateWaitingRoom(ctx context.Context, request models.WaitingRoom) (bool, error) {
	if pgrepository.pgConnectionPool == nil {
		return false, errors.New("Call NewRepository before invoking repository methods")
	}
	// construct insert parameterized query
	query := `INSERT INTO waitingrooms (room_id, created_at, updated_at, max_active_users_count, origin_application, status, active_session_ttl_seconds, waiting_session_ttl_seconds, polling_interval_seconds) VALUES (@room_id, @created_at, @updated_at, @max_active_users_count, @origin_application, @status, @active_session_ttl_seconds, @waiting_session_ttl_seconds, @polling_interval_seconds)`
	args := pgx.NamedArgs{
		"room_id":                     request.RoomId,
		"max_active_users_count":      request.MaxActiveUsersCount,
		"origin_application":          request.OriginApplication,
		"status":                      request.Status,
		"created_at":                  request.CreatedAt,
		"updated_at":                  request.UpdatedAt,
		"active_session_ttl_seconds":  request.ActiveSessionTtlSeconds,
		"waiting_session_ttl_seconds": request.WaitingSessionTtlSeconds,
		"polling_interval_seconds":    request.PollingIntervalSeconds,
	}
	log.Printf("Inserting waiting room into pg table: %v", request.RoomId)
	_, err := pgrepository.pgConnectionPool.Exec(ctx, query, args)
	if err != nil {
		log.Printf("Unable to insert row in waiting room: %v", err)
		return false, err
	}
	log.Printf("Successfully inserted waiting room into pg table: %v", request.RoomId)
	return true, nil
}

func (pgrepository *PgWaitingRoomRepository) GetWaitingRoom(ctx context.Context, request models.GetWaitingRoomRequest) (models.WaitingRoom, error) {
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

func (pgrepository *PgWaitingRoomRepository) DeleteWaitingRoom(ctx context.Context, request models.DeleteWaitingRoomRequest) (bool, error) {
	if pgrepository.pgConnectionPool == nil {
		return false, errors.New("Call NewRepository before invoking repository methods")
	}
	var (
		query string
		args  pgx.NamedArgs
	)
	if request.IsSoftDelete {
		query = `UPDATE waitingrooms SET status = @status, updated_at = @updated_at WHERE room_id = @room_id`
		args = pgx.NamedArgs{
			"status":     models.StatusDeleted,
			"updated_at": time.Now().UTC(),
			"room_id":    request.RoomId,
		}

	} else {
		query = `DELETE FROM waitingrooms WHERE room_id = @room_id`
		args = pgx.NamedArgs{
			"room_id": request.RoomId,
		}
	}

	commandTag, err := pgrepository.pgConnectionPool.Exec(ctx, query, args)
	if err != nil {
		log.Printf("Unable to delete waiting room: %v", err)
		return false, err
	}
	if commandTag.RowsAffected() == 0 {
		return false, fmt.Errorf("Failed to delete waiting room as room id  %s not found", request.RoomId)
	}
	return true, nil
}

func (pg *PgWaitingRoomRepository) Close() {
	pg.pgConnectionPool.Close()
}

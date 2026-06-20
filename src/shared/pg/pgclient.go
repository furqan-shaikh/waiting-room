package pg

import (
	"context"
	"log"
	"os"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	pgConnectionPool *pgxpool.Pool
	once             sync.Once
)

// This function creates a postgres client. Ensure this is called just once in the application
func _newInstance() *pgxpool.Pool {
	databaseUrl := os.Getenv("PG_DATABASE_URL")
	if databaseUrl == "" {
		errorStr := "Environment Variable PG_DATABASE_URL is not set"
		log.Printf("%s", errorStr)
		return nil
	}
	pgConnectionPool, err := pgxpool.New(context.Background(), databaseUrl)
	if err != nil {
		log.Printf("Unable to create connection pool: %v\n", err)
		return nil
	}
	log.Printf("Successfully created Pg Connection Pool")
	return pgConnectionPool
}

// sync.Once ensures a function is executed exactly once, even when called from multiple goroutines
func GetPostgreslient() *pgxpool.Pool {
	once.Do(func() {
		log.Println("Initializing Postgres Connection Pool...")
		pgConnectionPool = _newInstance()
	})
	return pgConnectionPool
}

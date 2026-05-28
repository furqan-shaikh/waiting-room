package redisclient

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/redis/go-redis/v9"
)

const DefaultAddress = "localhost:6379"

type RedisClientConfig struct {
	Address  string
	Username string
	Password string
	Protocol int
}

type LoadRedisLibraryRequest struct {
	SourceCode string
}

type LoadRedisLibraryResponse struct {
	Response string
}

type RedisFunctionRequest struct {
	RoomId             string
	MaxActiveUserCount int
	SessionToken       string
	TTLInSeconds       int
}

type RedisFunctionResponse struct {
	Decision            string
	NumberOfActiveUsers int64
}

var (
	redisClient *redis.Client
	once        sync.Once
)

// This function creates a redis client. Ensure this is called just once in the application
func _newInstance(redisClientConfig RedisClientConfig) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisClientConfig.Address,
		Username: redisClientConfig.Username,
		Password: redisClientConfig.Password,
		DB:       0, // use default DB
		Protocol: redisClientConfig.Protocol,
	})
	return rdb
}

// sync.Once ensures a function is executed exactly once, even when called from multiple goroutines
func GetRedisClient(redisClientConfig RedisClientConfig) *redis.Client {
	once.Do(func() {
		log.Println("Initializing Redis Singleton Instance...")
		redisClient = _newInstance(redisClientConfig)
	})
	return redisClient
}

func LoadRedisLibrary(request LoadRedisLibraryRequest) (LoadRedisLibraryResponse, error) {
	log.Println("Loading Redis Library...")
	ctx := context.Background()
	if redisClient == nil {
		return LoadRedisLibraryResponse{}, errors.New("Redis Client is not initialized. Call GetRedisClient first")
	}
	cmd := redisClient.FunctionLoadReplace(ctx, request.SourceCode)
	libraryName, err := cmd.Result()
	if err != nil {
		return LoadRedisLibraryResponse{}, err
	}
	return LoadRedisLibraryResponse{Response: libraryName}, nil
}

func InvokeRedisLibrary(args RedisFunctionRequest) (RedisFunctionResponse, error) {
	ctx := context.Background()
	result, err := redisClient.FCall(ctx, "waitingroomdecisionworkflow", []string{},
		args.RoomId,
		args.MaxActiveUserCount,
		args.SessionToken,
		args.TTLInSeconds,
	).Result()

	if err != nil {
		log.Printf("Redis Function execution failed: %v", err)
		return RedisFunctionResponse{}, err
	}

	// The Lua function returns a flat array: ["decision", "admit", "currentActiveUsersCount", 1]
	// go-redis reads this as a generic slice
	sliceResult, ok := result.([]interface{})
	if !ok {
		log.Printf("Unexpected response format from Redis")
		return RedisFunctionResponse{}, errors.New("Unexpected response format from Redis")
	}
	log.Printf("Raw Redis Output: %v\n", sliceResult)
	return RedisFunctionResponse{Decision: sliceResult[1].(string), NumberOfActiveUsers: sliceResult[3].(int64)}, nil
}

func Close() error {
	if redisClient != nil {
		log.Println("Closing Redis Client connection pool...")
		return redisClient.Close()
	}
	return nil
}

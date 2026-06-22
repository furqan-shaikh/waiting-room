package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"

	"waitingroom/controlplane/authn"
	"waitingroom/controlplane/authn/keyrepository"
	"waitingroom/controlplane/routes"
	"waitingroom/controlplane/services"
	"waitingroom/shared/pg"

	"github.com/joho/godotenv"
)

func main() {
	// Load the default .env file in the current directory
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file")
	}

	r := chi.NewRouter()
	keyRepository, keyLookupErr := getKeyLookupRepository()
	if keyLookupErr != nil {
		log.Printf("Failed to initialize Key Lookup Repository: %v", keyLookupErr)
		return
	}

	nonceRepository, nonceRepoError := pg.NewPgNonceRepository()
	if nonceRepoError != nil {
		log.Printf("Failed to initialize Nonce Repository: %v", nonceRepoError)
		return
	}
	r.Use(authn.ApiAuthn(authn.ApiAuthnConfig{KeyLookUpRepository: keyRepository,
		NonceRepository:   nonceRepository,
		AllowedClockSkew:  getAllowedClockSkew(),
		MaxExpiryDuration: getMaxExpiryDuration(),
	}))

	waitingRoomService, err := services.NewWaitingRoomService(context.Background())
	if err != nil {
		log.Printf("Failed to initialize waiting room service: %v", err)
		return
	}

	err = routes.RegisterRoutes(r, waitingRoomService)
	if err != nil {
		log.Fatalf("Failed to register routes: %v", err)
	}

	log.Printf("Control Plane Service listening on port 3000....")

	err = http.ListenAndServe(":3000", r)
	if err != nil {
		log.Fatalf("Server failed to listen: %v", err)
	}
}

func getKeyLookupRepository() (keyrepository.KeyLookup, error) {
	keyLookupType := os.Getenv("KEY_LOOKUP_TYPE")
	if keyLookupType == "" {
		return nil, errors.New("Key Lookup Type not configured")
	}
	if !isSupportedKeyLookupType() {
		return nil, errors.New("Invalid Key Lookup Repository not configured")
	}

	if keyLookupType == "local" {
		return &keyrepository.LocalFileSystemKeyRepository{
			FilePath:  os.Getenv("KEY_LOOKUP_TYPE_LOCAL_PATH"),
			Extension: os.Getenv("KEY_LOOKUP_TYPE_LOCAL_EXTENSION"),
		}, nil
	}

	return nil, errors.New("Unsupported Key Lookup Type")
}

func isSupportedKeyLookupType() bool {
	return os.Getenv("KEY_LOOKUP_TYPE") == "local"
}

func getAllowedClockSkew() int {
	allowedClockSkew := os.Getenv("ALLOWED_CLOCK_SKEW")
	if allowedClockSkew == "" {
		log.Printf("ALLOWED_CLOCK_SKEW is empty. Setting to default value")
		return authn.MaxAllowedClockSkew
	}
	return getValidClockSkew(allowedClockSkew)
}

func getValidClockSkew(allowedClockSkew string) int {
	// if not a valid number or greater than max allowed clock skew
	num, err := strconv.Atoi(allowedClockSkew)
	if err != nil {
		log.Printf("ALLOWED_CLOCK_SKEW is not a valid number. Setting to default value")
		return authn.MaxAllowedClockSkew
	}
	if num > authn.MaxAllowedClockSkew {
		log.Printf("ALLOWED_CLOCK_SKEW is not in a  valid value range")
		return authn.MaxAllowedClockSkew
	}
	if num <= 0 {
		log.Printf("ALLOWED_CLOCK_SKEW must be greater than 0")
		return authn.MaxAllowedClockSkew
	}
	return num
}

func getMaxExpiryDuration() int {
	maxExpiryDuration := os.Getenv("MAX_EXPIRY_DURATION")
	if maxExpiryDuration == "" {
		log.Printf("MAX_EXPIRY_DURATION is empty. Setting to default value")
		return authn.MaxExpiryDuration
	}
	return getValidMaxExpiryDuration(maxExpiryDuration)
}

func getValidMaxExpiryDuration(maxExpiryDuration string) int {
	// if not a valid number or greater than max Expiry Duration
	num, err := strconv.Atoi(maxExpiryDuration)
	if err != nil {
		log.Printf("MAX_EXPIRY_DURATION is not a valid number. Setting to default value")
		return authn.MaxExpiryDuration
	}
	if num > authn.MaxExpiryDuration {
		log.Printf("MAX_EXPIRY_DURATION is not in a  valid value range")
		return authn.MaxExpiryDuration
	}

	if num <= 0 {
		log.Printf("MAX_EXPIRY_DURATION must be greater than 0")
		return authn.MaxExpiryDuration
	}
	return num
}

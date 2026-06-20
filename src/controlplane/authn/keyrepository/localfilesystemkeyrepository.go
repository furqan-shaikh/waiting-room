package keyrepository

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type LocalFileSystemKeyRepository struct {
	FilePath  string
	Extension string
}

func (repository *LocalFileSystemKeyRepository) GetKey(keyId string) (*rsa.PublicKey, error) {
	fullPath := filepath.Join(repository.FilePath, fmt.Sprintf("%v%v", keyId, repository.Extension))
	bytes, readErr := os.ReadFile(fullPath)
	if readErr != nil {
		log.Printf("Failed to read key file: %v . Error: %v", fullPath, readErr)
		return nil, readErr
	}
	if repository.Extension == ".key" {
		key, decodeErr := JsonDecoder(bytes)
		if decodeErr != nil {
			log.Printf("Failed to decode key file: %v . Error: %v", fullPath, decodeErr)
			return nil, decodeErr
		}
		return key, nil
	}
	return nil, errors.New("Unsupported key file extension")
}

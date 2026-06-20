package keyrepository

import (
	"crypto/rsa"
	"encoding/json"
	"log"
)

func JsonDecoder(input []byte) (*rsa.PublicKey, error) {
	var key rsa.PublicKey
	err := json.Unmarshal(input, &key)
	if err != nil {
		log.Printf("Failed to decode request json: %v", err)
		return nil, err
	}
	return &key, nil
}

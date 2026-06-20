package keyrepository

import "crypto/rsa"

type Key struct {
	key *rsa.PublicKey
}

type KeyLookup interface {
	GetKey(keyId string) (*rsa.PublicKey, error)
}

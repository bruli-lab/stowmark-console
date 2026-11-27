package encryption

import (
	"errors"
	"path/filepath"
)

var ErrKeyPairIncomplete = errors.New("a key pair needs a private and a public key")

type KeyPair struct {
	privateKey string
	publicKey  string
}

func NewKeyPair(privateKey, publicKey string) (KeyPair, error) {
	switch {
	case privateKey == "" || publicKey == "":
		return KeyPair{}, ErrKeyPairIncomplete
	case !filepath.IsAbs(privateKey) || !filepath.IsAbs(publicKey):
		return KeyPair{}, ErrKeyPathNotAbsolute
	}

	return KeyPair{privateKey: privateKey, publicKey: publicKey}, nil
}

func (k KeyPair) PrivateKey() string {
	return k.privateKey
}

func (k KeyPair) PublicKey() string {
	return k.publicKey
}

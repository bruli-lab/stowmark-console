package encryption

import (
	"errors"
	"path/filepath"

	"github.com/bruli-lab/go-core/event"
)

var (
	ErrPrivateKeyPathRequired = errors.New("the private key is required")
	ErrPublicKeyPathRequired  = errors.New("the public key is required")
)

type Rewrap struct {
	event.BasicAggregateRoot
	oldPrivateKey string
	newPublicKey  string
}

func NewRewrap(oldPrivateKey, newPublicKey string) (*Rewrap, error) {
	if err := validateKeyPaths(oldPrivateKey, newPublicKey); err != nil {
		return nil, err
	}

	return &Rewrap{
		BasicAggregateRoot: event.NewBasicAggregateRoot(),
		oldPrivateKey:      oldPrivateKey,
		newPublicKey:       newPublicKey,
	}, nil
}

func (r *Rewrap) OldPrivateKey() string {
	return r.oldPrivateKey
}

func (r *Rewrap) NewPublicKey() string {
	return r.newPublicKey
}

func (r *Rewrap) Rewrapped() {
	r.Record(NewKeyRewrapped(r.oldPrivateKey, r.newPublicKey))
}

func validateKeyPaths(privateKey, publicKey string) error {
	switch {
	case privateKey == "":
		return ErrPrivateKeyPathRequired
	case publicKey == "":
		return ErrPublicKeyPathRequired
	case !filepath.IsAbs(privateKey) || !filepath.IsAbs(publicKey):
		return ErrKeyPathNotAbsolute
	}

	return nil
}

package store

import (
	"context"

	"github.com/bruli-lab/stowmark-console/internal/domain/encryption"
)

type Keys struct {
	session *Session
}

func NewKeys(session *Session) *Keys {
	return &Keys{session: session}
}

func (k *Keys) Generate(ctx context.Context, generation *encryption.KeyGeneration) (encryption.KeyPair, error) {
	handler, err := k.session.current()
	if err != nil {
		return encryption.KeyPair{}, err
	}

	generated, err := handler.KeyGenerate(ctx, generation.Folder())
	if err != nil {
		return encryption.KeyPair{}, translate(err)
	}

	return encryption.NewKeyPair(generated.PrivateKey(), generated.PublicKey())
}

func (k *Keys) Rewrap(ctx context.Context, rewrap *encryption.Rewrap) error {
	handler, err := k.session.current()
	if err != nil {
		return err
	}

	return translate(handler.Rewrap(ctx, rewrap.OldPrivateKey(), rewrap.NewPublicKey()))
}

func (k *Keys) Rotate(ctx context.Context, rotation *encryption.Rotation) error {
	handler, err := k.session.current()
	if err != nil {
		return err
	}

	return translate(handler.ReKey(ctx, rotation.PrivateKey(), rotation.PublicKey()))
}

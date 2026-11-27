package app

import (
	"context"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli-lab/stowmark-console/internal/domain/encryption"
)

const RotateKeyCommandName = "rotate-key"

type RotateKeyCommand struct {
	PrivateKey string
	PublicKey  string
}

func (RotateKeyCommand) Name() string {
	return RotateKeyCommandName
}

type RotateKey struct {
	keys encryption.Keys
}

func NewRotateKey(keys encryption.Keys) RotateKey {
	return RotateKey{keys: keys}
}

func (h RotateKey) Handle(ctx context.Context, command cqs.Command) ([]event.Event, error) {
	cmd, ok := command.(RotateKeyCommand)
	if !ok {
		return nil, cqs.NewInvalidCommandError(RotateKeyCommandName, command.Name())
	}

	rotation, err := encryption.NewRotation(cmd.PrivateKey, cmd.PublicKey)
	if err != nil {
		return nil, err
	}

	if err := h.keys.Rotate(ctx, rotation); err != nil {
		return nil, err
	}

	rotation.Rotated()

	return rotation.Events(), nil
}

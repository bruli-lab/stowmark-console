package app

import (
	"context"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli-lab/stowmark-console/internal/domain/encryption"
)

const RewrapKeyCommandName = "rewrap-key"

type RewrapKeyCommand struct {
	OldPrivateKey string
	NewPublicKey  string
}

func (RewrapKeyCommand) Name() string {
	return RewrapKeyCommandName
}

type RewrapKey struct {
	keys encryption.Keys
}

func NewRewrapKey(keys encryption.Keys) RewrapKey {
	return RewrapKey{keys: keys}
}

func (h RewrapKey) Handle(ctx context.Context, command cqs.Command) ([]event.Event, error) {
	cmd, ok := command.(RewrapKeyCommand)
	if !ok {
		return nil, cqs.NewInvalidCommandError(RewrapKeyCommandName, command.Name())
	}

	rewrap, err := encryption.NewRewrap(cmd.OldPrivateKey, cmd.NewPublicKey)
	if err != nil {
		return nil, err
	}

	if err := h.keys.Rewrap(ctx, rewrap); err != nil {
		return nil, err
	}

	rewrap.Rewrapped()

	return rewrap.Events(), nil
}

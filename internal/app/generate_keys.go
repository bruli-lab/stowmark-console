package app

import (
	"context"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli-lab/stowmark-console/internal/domain/encryption"
)

const GenerateKeysCommandName = "generate-keys"

type GenerateKeysCommand struct {
	Folder string
}

func (GenerateKeysCommand) Name() string {
	return GenerateKeysCommandName
}

type GenerateKeys struct {
	keys encryption.Keys
}

func NewGenerateKeys(keys encryption.Keys) GenerateKeys {
	return GenerateKeys{keys: keys}
}

func (h GenerateKeys) Handle(ctx context.Context, command cqs.Command) ([]event.Event, error) {
	cmd, ok := command.(GenerateKeysCommand)
	if !ok {
		return nil, cqs.NewInvalidCommandError(GenerateKeysCommandName, command.Name())
	}

	generation, err := encryption.NewKeyGeneration(cmd.Folder)
	if err != nil {
		return nil, err
	}

	keys, err := h.keys.Generate(ctx, generation)
	if err != nil {
		return nil, err
	}

	if err := generation.Generated(keys); err != nil {
		return nil, err
	}

	return generation.Events(), nil
}

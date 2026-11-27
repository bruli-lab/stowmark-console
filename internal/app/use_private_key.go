package app

import (
	"context"
	"errors"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
)

const UsePrivateKeyCommandName = "use-private-key"

var ErrPrivateKeyPathRequired = errors.New("private key file is required")

type UsePrivateKeyCommand struct {
	Profile    repository.Profile
	PrivateKey string
}

func (UsePrivateKeyCommand) Name() string {
	return UsePrivateKeyCommandName
}

type UsePrivateKey struct {
	catalog repository.Catalog
}

func NewUsePrivateKey(catalog repository.Catalog) UsePrivateKey {
	return UsePrivateKey{catalog: catalog}
}

func (h UsePrivateKey) Handle(ctx context.Context, command cqs.Command) ([]event.Event, error) {
	cmd, ok := command.(UsePrivateKeyCommand)
	if !ok {
		return nil, cqs.NewInvalidCommandError(UsePrivateKeyCommandName, command.Name())
	}

	if cmd.Profile.Name() == "" {
		return nil, ErrRepositoryNameRequired
	}

	if cmd.PrivateKey == "" {
		return nil, ErrPrivateKeyPathRequired
	}

	opened, err := h.catalog.Update(ctx, cmd.Profile.WithPrivateKey(&cmd.PrivateKey))
	if err != nil {
		return nil, err
	}

	return []event.Event{repository.NewOpened(opened)}, nil
}

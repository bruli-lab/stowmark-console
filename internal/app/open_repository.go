package app

import (
	"context"
	"errors"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
)

const OpenRepositoryCommandName = "open-repository"

var (
	ErrNoRepositories         = errors.New("no repositories configured")
	ErrRepositoryNameRequired = errors.New("repository name is required")
)

type OpenRepositoryCommand struct {
	RepositoryName string
}

func (OpenRepositoryCommand) Name() string {
	return OpenRepositoryCommandName
}

type OpenRepository struct {
	catalog repository.Catalog
}

func NewOpenRepository(catalog repository.Catalog) OpenRepository {
	return OpenRepository{catalog: catalog}
}

func (h OpenRepository) Handle(ctx context.Context, command cqs.Command) ([]event.Event, error) {
	cmd, ok := command.(OpenRepositoryCommand)
	if !ok {
		return nil, cqs.NewInvalidCommandError(OpenRepositoryCommandName, command.Name())
	}

	if cmd.RepositoryName == "" {
		return nil, ErrRepositoryNameRequired
	}

	profile, err := h.catalog.Open(ctx, cmd.RepositoryName)
	if err != nil {
		return nil, err
	}

	return []event.Event{repository.NewOpened(profile)}, nil
}

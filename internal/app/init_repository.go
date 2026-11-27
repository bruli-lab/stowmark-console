package app

import (
	"context"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
)

const InitRepositoryCommandName = "init-repository"

type InitRepositoryCommand struct {
	Options repository.InitOptions
}

func (InitRepositoryCommand) Name() string {
	return InitRepositoryCommandName
}

type InitRepository struct {
	repository repository.Repository
}

func NewInitRepository(repo repository.Repository) InitRepository {
	return InitRepository{repository: repo}
}

func (h InitRepository) Handle(ctx context.Context, command cqs.Command) ([]event.Event, error) {
	cmd, ok := command.(InitRepositoryCommand)
	if !ok {
		return nil, cqs.NewInvalidCommandError(InitRepositoryCommandName, command.Name())
	}

	if err := cmd.Options.Validate(); err != nil {
		return nil, err
	}

	if err := h.repository.Initialize(ctx, cmd.Options); err != nil {
		return nil, err
	}

	return []event.Event{repository.NewInitialized(cmd.Options)}, nil
}

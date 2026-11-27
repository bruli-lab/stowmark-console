package app

import (
	"context"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
)

const UnlockRepositoryCommandName = "unlock-repository"

type UnlockRepositoryCommand struct {
	All bool
}

func (UnlockRepositoryCommand) Name() string {
	return UnlockRepositoryCommandName
}

type UnlockRepository struct {
	repository repository.Repository
}

func NewUnlockRepository(repo repository.Repository) UnlockRepository {
	return UnlockRepository{repository: repo}
}

func (h UnlockRepository) Handle(ctx context.Context, command cqs.Command) ([]event.Event, error) {
	cmd, ok := command.(UnlockRepositoryCommand)
	if !ok {
		return nil, cqs.NewInvalidCommandError(UnlockRepositoryCommandName, command.Name())
	}

	report, err := h.repository.Unlock(ctx, cmd.All)
	if err != nil {
		return nil, err
	}

	return []event.Event{repository.NewUnlocked(cmd.All, report)}, nil
}

package app

import (
	"context"

	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
)

type InitRepositoryCommand struct {
	Options repository.InitOptions
}

type InitRepository struct {
	repository repository.Repository
}

func NewInitRepository(repo repository.Repository) *InitRepository {
	return &InitRepository{repository: repo}
}

func (h *InitRepository) Handle(ctx context.Context, cmd InitRepositoryCommand) (repository.Initialized, error) {
	if err := h.repository.Initialize(ctx, cmd.Options); err != nil {
		return repository.Initialized{}, err
	}

	return repository.Initialized{Options: cmd.Options}, nil
}

package app

import (
	"context"
	"errors"

	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
)

var (
	ErrNoRepositories         = errors.New("no repositories configured")
	ErrRepositoryNameRequired = errors.New("repository name is required")
)

type OpenRepositoryCommand struct {
	Name string
}

type OpenRepository struct {
	catalog repository.Catalog
}

func NewOpenRepository(catalog repository.Catalog) *OpenRepository {
	return &OpenRepository{catalog: catalog}
}

func (h *OpenRepository) Handle(ctx context.Context, cmd OpenRepositoryCommand) (repository.Opened, error) {
	if cmd.Name == "" {
		return repository.Opened{}, ErrRepositoryNameRequired
	}

	profile, err := h.catalog.Open(ctx, cmd.Name)
	if err != nil {
		return repository.Opened{}, err
	}

	return repository.Opened{Profile: profile}, nil
}

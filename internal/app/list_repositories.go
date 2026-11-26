package app

import (
	"context"

	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
)

type ListRepositoriesQuery struct{}

type ListRepositories struct {
	catalog repository.Catalog
}

func NewListRepositories(catalog repository.Catalog) *ListRepositories {
	return &ListRepositories{catalog: catalog}
}

func (h *ListRepositories) Handle(_ context.Context, _ ListRepositoriesQuery) ([]repository.Profile, error) {
	profiles := h.catalog.Profiles()
	if len(profiles) == 0 {
		return nil, ErrNoRepositories
	}

	return profiles, nil
}

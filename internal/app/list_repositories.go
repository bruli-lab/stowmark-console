package app

import (
	"context"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
)

const ListRepositoriesQueryName = "list-repositories"

type ListRepositoriesQuery struct{}

func (ListRepositoriesQuery) Name() string {
	return ListRepositoriesQueryName
}

type ListRepositories struct {
	catalog repository.Catalog
}

func NewListRepositories(catalog repository.Catalog) ListRepositories {
	return ListRepositories{catalog: catalog}
}

func (h ListRepositories) Handle(_ context.Context, query cqs.Query) (any, error) {
	if _, ok := query.(ListRepositoriesQuery); !ok {
		return nil, cqs.NewInvalidQueryError(ListRepositoriesQueryName, query.Name())
	}

	profiles := h.catalog.Profiles()
	if len(profiles) == 0 {
		return nil, ErrNoRepositories
	}

	return profiles, nil
}

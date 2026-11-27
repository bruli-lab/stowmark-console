package app

import (
	"context"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
)

const RepositoryStatusQueryName = "repository-status"

type RepositoryStatusQuery struct{}

func (RepositoryStatusQuery) Name() string {
	return RepositoryStatusQueryName
}

type RepositoryStatus struct {
	repository repository.Repository
}

func NewRepositoryStatus(repo repository.Repository) RepositoryStatus {
	return RepositoryStatus{repository: repo}
}

func (h RepositoryStatus) Handle(ctx context.Context, query cqs.Query) (any, error) {
	if _, ok := query.(RepositoryStatusQuery); !ok {
		return nil, cqs.NewInvalidQueryError(RepositoryStatusQueryName, query.Name())
	}

	return h.repository.Status(ctx)
}

package app

import (
	"context"

	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
)

type RepositoryStatusQuery struct{}

type RepositoryStatus struct {
	repository repository.Repository
}

func NewRepositoryStatus(repo repository.Repository) *RepositoryStatus {
	return &RepositoryStatus{repository: repo}
}

func (h *RepositoryStatus) Handle(ctx context.Context, _ RepositoryStatusQuery) (bool, error) {
	return h.repository.Initialized(ctx)
}

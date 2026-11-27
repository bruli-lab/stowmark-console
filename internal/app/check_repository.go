package app

import (
	"context"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/stowmark-console/internal/domain/integrity"
)

const CheckRepositoryQueryName = "check-repository"

type CheckRepositoryQuery struct {
	Scope integrity.Scope
}

func (CheckRepositoryQuery) Name() string {
	return CheckRepositoryQueryName
}

type CheckRepository struct {
	checker integrity.Checker
}

func NewCheckRepository(checker integrity.Checker) CheckRepository {
	return CheckRepository{checker: checker}
}

func (h CheckRepository) Handle(ctx context.Context, query cqs.Query) (any, error) {
	q, ok := query.(CheckRepositoryQuery)
	if !ok {
		return nil, cqs.NewInvalidQueryError(CheckRepositoryQueryName, query.Name())
	}

	report, err := h.checker.Check(ctx, q.Scope)
	if err != nil {
		return nil, err
	}

	report.Scope = q.Scope

	return report, nil
}

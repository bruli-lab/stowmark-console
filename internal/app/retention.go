package app

import (
	"context"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli-lab/stowmark-console/internal/domain/retention"
)

const (
	PlanForgetQueryName        = "plan-forget"
	ForgetSnapshotsCommandName = "forget-snapshots"
	PlanPruneQueryName         = "plan-prune"
	PruneRepositoryCommandName = "prune-repository"
)

type PlanForgetQuery struct {
	Rules retention.Rules
}

func (PlanForgetQuery) Name() string {
	return PlanForgetQueryName
}

type PlanForget struct {
	retention retention.Repository
}

func NewPlanForget(repo retention.Repository) PlanForget {
	return PlanForget{retention: repo}
}

func (h PlanForget) Handle(ctx context.Context, query cqs.Query) (any, error) {
	q, ok := query.(PlanForgetQuery)
	if !ok {
		return nil, cqs.NewInvalidQueryError(PlanForgetQueryName, query.Name())
	}

	policy, err := retention.NewPolicy(q.Rules)
	if err != nil {
		return nil, err
	}

	return h.retention.PlanForget(ctx, policy)
}

type ForgetSnapshotsCommand struct {
	Rules retention.Rules
}

func (ForgetSnapshotsCommand) Name() string {
	return ForgetSnapshotsCommandName
}

type ForgetSnapshots struct {
	retention retention.Repository
}

func NewForgetSnapshots(repo retention.Repository) ForgetSnapshots {
	return ForgetSnapshots{retention: repo}
}

func (h ForgetSnapshots) Handle(ctx context.Context, command cqs.Command) ([]event.Event, error) {
	cmd, ok := command.(ForgetSnapshotsCommand)
	if !ok {
		return nil, cqs.NewInvalidCommandError(ForgetSnapshotsCommandName, command.Name())
	}

	forgetting, err := retention.NewForgetting(cmd.Rules)
	if err != nil {
		return nil, err
	}

	result, err := h.retention.Forget(ctx, forgetting)
	if err != nil {
		return nil, err
	}

	forgetting.Forgot(result)

	return forgetting.Events(), nil
}

type PlanPruneQuery struct{}

func (PlanPruneQuery) Name() string {
	return PlanPruneQueryName
}

type PlanPrune struct {
	retention retention.Repository
}

func NewPlanPrune(repo retention.Repository) PlanPrune {
	return PlanPrune{retention: repo}
}

func (h PlanPrune) Handle(ctx context.Context, query cqs.Query) (any, error) {
	if _, ok := query.(PlanPruneQuery); !ok {
		return nil, cqs.NewInvalidQueryError(PlanPruneQueryName, query.Name())
	}

	return h.retention.PlanPrune(ctx)
}

type PruneRepositoryCommand struct{}

func (PruneRepositoryCommand) Name() string {
	return PruneRepositoryCommandName
}

type PruneRepository struct {
	retention retention.Repository
}

func NewPruneRepository(repo retention.Repository) PruneRepository {
	return PruneRepository{retention: repo}
}

func (h PruneRepository) Handle(ctx context.Context, command cqs.Command) ([]event.Event, error) {
	if _, ok := command.(PruneRepositoryCommand); !ok {
		return nil, cqs.NewInvalidCommandError(PruneRepositoryCommandName, command.Name())
	}

	pruning := retention.NewPruning()

	result, err := h.retention.Prune(ctx, pruning)
	if err != nil {
		return nil, err
	}

	pruning.Pruned(&result)

	return pruning.Events(), nil
}

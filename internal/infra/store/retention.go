package store

import (
	"context"

	"github.com/bruli-lab/stowmark-console/internal/domain/retention"
	"github.com/bruli-lab/stowmark/pkg/stowmark"
)

type Retention struct {
	session *Session
}

func NewRetention(session *Session) *Retention {
	return &Retention{session: session}
}

func (r *Retention) PlanForget(ctx context.Context, policy retention.Policy) (retention.Plan, error) {
	handler, err := r.session.current()
	if err != nil {
		return nil, err
	}

	decisions, err := handler.PlanForget(ctx, retentionPolicy(policy))
	if err != nil {
		return nil, translate(err)
	}

	plan := make(retention.Plan, len(decisions))
	for i := range decisions {
		plan[i] = retention.Decision{
			SnapshotID: decisions[i].ID,
			CreatedAt:  decisions[i].CreatedAt,
			Source:     decisions[i].Source,
			Reasons:    decisions[i].Reasons,
		}
	}

	return plan, nil
}

func (r *Retention) Forget(ctx context.Context, forgetting *retention.Forgetting) (retention.ForgetResult, error) {
	handler, err := r.session.current()
	if err != nil {
		return retention.ForgetResult{}, err
	}

	result, err := handler.ForgetSnapshots(ctx, retentionPolicy(forgetting.Policy()))
	if err != nil {
		return retention.ForgetResult{}, translate(err)
	}

	forgotten := make([]retention.ForgottenSnapshot, len(result.Forgotten))
	for i, item := range result.Forgotten {
		forgotten[i] = retention.ForgottenSnapshot{ID: item.ID, CreatedAt: item.CreatedAt, Source: item.Source}
	}

	postponed := make([]retention.PostponedSnapshot, len(result.Postponed))
	for i, item := range result.Postponed {
		postponed[i] = retention.PostponedSnapshot{
			ID:            item.ID,
			CreatedAt:     item.CreatedAt,
			Source:        item.Source,
			RetainedUntil: item.RetainedUntil,
		}
	}

	return retention.ForgetResult{Forgotten: forgotten, Postponed: postponed}, nil
}

func (r *Retention) PlanPrune(ctx context.Context) (retention.PrunePlan, error) {
	handler, err := r.session.current()
	if err != nil {
		return retention.PrunePlan{}, err
	}

	plan, err := handler.PlanPrune(ctx)
	if err != nil {
		return retention.PrunePlan{}, translate(err)
	}

	return retention.PrunePlan{
		StoredObjects:       plan.StoredObjects,
		ReferencedObjects:   plan.ReferencedObjects,
		UnreferencedObjects: plan.UnreferencedObjects,
		UnreferencedBytes:   plan.UnreferencedBytes,
		MissingObjects:      plan.MissingObjects,
		OldKeyGenerations:   plan.OldKeyGenerations,
	}, nil
}

func (r *Retention) Prune(ctx context.Context, _ *retention.Pruning) (retention.PruneResult, error) {
	handler, err := r.session.current()
	if err != nil {
		return retention.PruneResult{}, err
	}

	result, err := handler.Prune(ctx)
	if err != nil {
		return retention.PruneResult{}, translate(err)
	}

	return retention.PruneResult{
		RemovedObjects:          result.RemovedObjects,
		FreedBytes:              result.FreedBytes,
		MissingObjects:          result.MissingObjects,
		RemovedKeyGenerations:   result.RemovedKeyGenerations,
		RemovedTemporaryFiles:   result.RemovedTemporaryFiles,
		PostponedObjects:        result.PostponedObjects,
		PostponedBytes:          result.PostponedBytes,
		PostponedKeyGenerations: result.PostponedKeyGenerations,
		RetainedUntil:           result.RetainedUntil,
	}, nil
}

func retentionPolicy(policy retention.Policy) stowmark.RetentionPolicy {
	rules := policy.Rules()

	return stowmark.RetentionPolicy{
		KeepLast:    rules.Last,
		KeepDaily:   rules.Daily,
		KeepWeekly:  rules.Weekly,
		KeepMonthly: rules.Monthly,
		KeepYearly:  rules.Yearly,
		KeepWithin:  rules.Within,
	}
}

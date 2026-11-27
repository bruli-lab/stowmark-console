package retention

import "context"

type Repository interface {
	PlanForget(ctx context.Context, policy Policy) (Plan, error)
	Forget(ctx context.Context, forgetting *Forgetting) (ForgetResult, error)
	PlanPrune(ctx context.Context) (PrunePlan, error)
	Prune(ctx context.Context, pruning *Pruning) (PruneResult, error)
}

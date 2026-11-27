package app_test

import (
	"context"
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/app"
	"github.com/bruli-lab/stowmark-console/internal/domain/retention"
	"github.com/bruli-lab/stowmark-console/internal/mocks"
	"github.com/stretchr/testify/require"
)

func TestPlanForget(t *testing.T) {
	t.Parallel()

	t.Run(`Given a query that is not a plan forget query
	When it is handled
	Then an invalid query error is returned`, func(t *testing.T) {
		t.Parallel()

		requireInvalidQuery(t, app.NewPlanForget(&mocks.RetentionRepositoryMock{}))
	})

	t.Run(`Given keep rules
	When forgetting is planned
	Then the policy reaches the repository and its plan is returned`, func(t *testing.T) {
		t.Parallel()
		plan := retention.Plan{{SnapshotID: "a", Reasons: []string{"last"}}, {SnapshotID: "b"}}
		repo := &mocks.RetentionRepositoryMock{
			PlanForgetFunc: func(context.Context, retention.Policy) (retention.Plan, error) { return plan, nil },
		}

		result, err := app.NewPlanForget(repo).Handle(t.Context(), app.PlanForgetQuery{Rules: retention.Rules{Last: 1}})

		require.NoError(t, err)
		require.Equal(t, plan, result)
		require.Equal(t, 1, repo.PlanForgetCalls()[0].Policy.Rules().Last)
	})

	t.Run(`Given no keep rule
	When forgetting is planned
	Then it is rejected before reaching the repository`, func(t *testing.T) {
		t.Parallel()
		repo := &mocks.RetentionRepositoryMock{}

		_, err := app.NewPlanForget(repo).Handle(t.Context(), app.PlanForgetQuery{})

		require.ErrorIs(t, err, retention.ErrEmptyPolicy)
		require.Empty(t, repo.PlanForgetCalls())
	})
}

func TestForgetSnapshots(t *testing.T) {
	t.Parallel()

	t.Run(`Given a command that is not a forget snapshots command
	When it is handled
	Then an invalid command error is returned`, func(t *testing.T) {
		t.Parallel()

		requireInvalidCommand(t, app.NewForgetSnapshots(&mocks.RetentionRepositoryMock{}))
	})

	t.Run(`Given keep rules
	When the snapshots are forgotten
	Then a snapshots forgotten event carries the result`, func(t *testing.T) {
		t.Parallel()
		result := retention.ForgetResult{Forgotten: []retention.ForgottenSnapshot{{ID: "b"}}}
		repo := &mocks.RetentionRepositoryMock{
			ForgetFunc: func(context.Context, *retention.Forgetting) (retention.ForgetResult, error) { return result, nil },
		}

		events, err := app.NewForgetSnapshots(repo).Handle(t.Context(), app.ForgetSnapshotsCommand{Rules: retention.Rules{Daily: 7}})

		require.NoError(t, err)
		forgotten := singleEvent[retention.SnapshotsForgotten](t, events, retention.SnapshotsForgottenEventName)
		require.Equal(t, result, forgotten.Result)
		require.Equal(t, 7, forgotten.Policy.Rules().Daily)
	})

	t.Run(`Given no keep rule
	When the snapshots are forgotten
	Then it is rejected before reaching the repository`, func(t *testing.T) {
		t.Parallel()
		repo := &mocks.RetentionRepositoryMock{}

		_, err := app.NewForgetSnapshots(repo).Handle(t.Context(), app.ForgetSnapshotsCommand{})

		require.ErrorIs(t, err, retention.ErrEmptyPolicy)
		require.Empty(t, repo.ForgetCalls())
	})

	t.Run(`Given a repository that fails to forget
	When the snapshots are forgotten
	Then the error is returned`, func(t *testing.T) {
		t.Parallel()
		repo := &mocks.RetentionRepositoryMock{
			ForgetFunc: func(context.Context, *retention.Forgetting) (retention.ForgetResult, error) {
				return retention.ForgetResult{}, errBackend
			},
		}

		_, err := app.NewForgetSnapshots(repo).Handle(t.Context(), app.ForgetSnapshotsCommand{Rules: retention.Rules{Last: 1}})

		require.ErrorIs(t, err, errBackend)
	})
}

func TestPrune(t *testing.T) {
	t.Parallel()

	t.Run(`Given a query that is not a plan prune query
	When it is handled
	Then an invalid query error is returned`, func(t *testing.T) {
		t.Parallel()

		requireInvalidQuery(t, app.NewPlanPrune(&mocks.RetentionRepositoryMock{}))
	})

	t.Run(`Given a command that is not a prune repository command
	When it is handled
	Then an invalid command error is returned`, func(t *testing.T) {
		t.Parallel()

		requireInvalidCommand(t, app.NewPruneRepository(&mocks.RetentionRepositoryMock{}))
	})

	t.Run(`Given a repository with unreferenced objects
	When pruning is planned
	Then its plan is returned`, func(t *testing.T) {
		t.Parallel()
		plan := retention.PrunePlan{UnreferencedObjects: 3, UnreferencedBytes: 300}
		repo := &mocks.RetentionRepositoryMock{
			PlanPruneFunc: func(context.Context) (retention.PrunePlan, error) { return plan, nil },
		}

		result, err := app.NewPlanPrune(repo).Handle(t.Context(), app.PlanPruneQuery{})

		require.NoError(t, err)
		require.Equal(t, plan, result)
	})

	t.Run(`Given a repository with unreferenced objects
	When it is pruned
	Then a repository pruned event carries the result`, func(t *testing.T) {
		t.Parallel()
		result := retention.PruneResult{RemovedObjects: 3, FreedBytes: 300}
		repo := &mocks.RetentionRepositoryMock{
			PruneFunc: func(context.Context, *retention.Pruning) (retention.PruneResult, error) { return result, nil },
		}

		events, err := app.NewPruneRepository(repo).Handle(t.Context(), app.PruneRepositoryCommand{})

		require.NoError(t, err)
		pruned := singleEvent[retention.RepositoryPruned](t, events, retention.RepositoryPrunedEventName)
		require.Equal(t, result, pruned.Result)
	})

	t.Run(`Given a repository that fails to prune
	When it is pruned
	Then the error is returned`, func(t *testing.T) {
		t.Parallel()
		repo := &mocks.RetentionRepositoryMock{
			PruneFunc: func(context.Context, *retention.Pruning) (retention.PruneResult, error) {
				return retention.PruneResult{}, errBackend
			},
		}

		_, err := app.NewPruneRepository(repo).Handle(t.Context(), app.PruneRepositoryCommand{})

		require.ErrorIs(t, err, errBackend)
	})
}

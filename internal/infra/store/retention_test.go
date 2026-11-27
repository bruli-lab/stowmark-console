package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/domain/retention"
	"github.com/bruli-lab/stowmark-console/internal/infra/store"
	"github.com/stretchr/testify/require"
)

func TestStoreRetention(t *testing.T) {
	t.Setenv("STOWMARK_LOG_LEVEL", "error")
	source := writeSource(t)

	session, err := store.Open(t.Context(), filepath.Join(t.TempDir(), "repository"), nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, session.Close()) })
	require.NoError(t, store.NewRepositories(session).Initialize(t.Context(), repository.NewInitOptions("none", nil, nil)))
	snapshots := store.NewSnapshots(session)
	repo := store.NewRetention(session)

	var newest string
	for _, content := range []string{"first version", "second version", "third version"} {
		require.NoError(t, os.WriteFile(filepath.Join(source, "README.txt"), []byte(content), 0o600))
		created, err := snapshots.Create(t.Context(), creation(t, source))
		require.NoError(t, err)
		newest = created.ID()
	}
	keepLast := retention.Rules{Last: 1}

	t.Run(`Given three snapshots of a source
	When forgetting is planned keeping the last one
	Then the newest is kept by the last rule and the others would be forgotten`, func(t *testing.T) {
		policy, err := retention.NewPolicy(keepLast)
		require.NoError(t, err)

		plan, err := repo.PlanForget(t.Context(), policy)

		require.NoError(t, err)
		require.Len(t, plan, 3)
		require.Equal(t, 2, plan.Forgotten())
		for _, decision := range plan {
			require.Equal(t, decision.SnapshotID == newest, decision.Keep())
			require.Equal(t, source, decision.Source)
		}
	})

	t.Run(`Given three snapshots of a source
	When the snapshots are forgotten keeping the last one
	Then two are forgotten and only the newest is listed`, func(t *testing.T) {
		forgetting, err := retention.NewForgetting(keepLast)
		require.NoError(t, err)

		result, err := repo.Forget(t.Context(), forgetting)

		require.NoError(t, err)
		require.Len(t, result.Forgotten, 2)
		require.Empty(t, result.Postponed)
		listed, err := snapshots.List(t.Context())
		require.NoError(t, err)
		require.Len(t, listed, 1)
		require.Equal(t, newest, listed[0].ID())
	})

	var plan retention.PrunePlan

	t.Run(`Given forgotten snapshots with their own contents
	When pruning is planned
	Then the unreferenced objects and their size are reported`, func(t *testing.T) {
		plan, err = repo.PlanPrune(t.Context())

		require.NoError(t, err)
		require.Positive(t, plan.UnreferencedObjects)
		require.Positive(t, plan.UnreferencedBytes)
		require.Zero(t, plan.MissingObjects)
		require.False(t, plan.Empty())
	})

	t.Run(`Given unreferenced objects
	When the repository is pruned
	Then they are removed, nothing is left to prune and the kept snapshot is still intact`, func(t *testing.T) {
		result, err := repo.Prune(t.Context(), retention.NewPruning())

		require.NoError(t, err)
		require.Equal(t, plan.UnreferencedObjects, result.RemovedObjects)
		require.Equal(t, plan.UnreferencedBytes, result.FreedBytes)
		after, err := repo.PlanPrune(t.Context())
		require.NoError(t, err)
		require.True(t, after.Empty())
		failures, err := snapshots.Verify(t.Context(), verification(t, newest))
		require.NoError(t, err)
		require.Empty(t, failures)
	})
}

func TestStoreRetentionWithoutRepository(t *testing.T) {
	t.Parallel()
	repo := store.NewRetention(store.NewSession())
	policy, err := retention.NewPolicy(retention.Rules{Last: 1})
	require.NoError(t, err)
	forgetting, err := retention.NewForgetting(retention.Rules{Last: 1})
	require.NoError(t, err)

	_, planErr := repo.PlanForget(t.Context(), policy)
	_, forgetErr := repo.Forget(t.Context(), forgetting)
	_, prunePlanErr := repo.PlanPrune(t.Context())
	_, pruneErr := repo.Prune(t.Context(), retention.NewPruning())

	for _, err := range []error{planErr, forgetErr, prunePlanErr, pruneErr} {
		require.ErrorIs(t, err, store.ErrNotConnected)
	}
}

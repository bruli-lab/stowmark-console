package retention_test

import (
	"testing"
	"time"

	"github.com/bruli-lab/stowmark-console/internal/domain/retention"
	"github.com/stretchr/testify/require"
)

func TestPolicy(t *testing.T) {
	t.Parallel()

	t.Run(`Given rules with at least one keep rule
	When a policy is built
	Then it keeps the rules`, func(t *testing.T) {
		t.Parallel()
		rules := retention.Rules{Last: 3, Within: 48 * time.Hour}

		policy, err := retention.NewPolicy(rules)

		require.NoError(t, err)
		require.Equal(t, rules, policy.Rules())
	})

	t.Run(`Given rules without any keep rule
	When a policy is built
	Then it returns an empty policy error`, func(t *testing.T) {
		t.Parallel()

		_, err := retention.NewPolicy(retention.Rules{})

		require.ErrorIs(t, err, retention.ErrEmptyPolicy)
	})

	t.Run(`Given a negative rule
	When a policy is built
	Then it returns a negative rule error`, func(t *testing.T) {
		t.Parallel()

		_, err := retention.NewPolicy(retention.Rules{Last: 1, Daily: -1})

		require.ErrorIs(t, err, retention.ErrNegativeRule)
	})

	t.Run(`Given rules
	When they are described
	Then only the set rules are listed in order`, func(t *testing.T) {
		t.Parallel()

		description := retention.Rules{Last: 7, Monthly: 12, Within: 14 * 24 * time.Hour}.String()

		require.Equal(t, "last 7, monthly 12, within 2w", description)
	})
}

func TestWithin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value    string
		expected time.Duration
		err      error
	}{
		{value: "", expected: 0},
		{value: "30d", expected: 30 * 24 * time.Hour},
		{value: "2w", expected: 14 * 24 * time.Hour},
		{value: " 36h ", expected: 36 * time.Hour},
		{value: "xd", err: retention.ErrInvalidDuration},
		{value: "-1d", err: retention.ErrInvalidDuration},
		{value: "-2h", err: retention.ErrInvalidDuration},
		{value: "3y", err: retention.ErrInvalidDuration},
	}

	for _, tt := range tests {
		t.Run(`Given the keep within value "`+tt.value+`"
		When it is parsed
		Then it returns its duration or an invalid duration error`, func(t *testing.T) {
			t.Parallel()

			duration, err := retention.ParseWithin(tt.value)

			require.ErrorIs(t, err, tt.err)
			require.Equal(t, tt.expected, duration)
		})
	}

	t.Run(`Given durations in weeks, days and hours
	When they are formatted
	Then they use the largest whole unit`, func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "2w", retention.FormatWithin(14*24*time.Hour))
		require.Equal(t, "30d", retention.FormatWithin(30*24*time.Hour))
		require.Equal(t, "36h", retention.FormatWithin(36*time.Hour))
		require.Equal(t, "1h30m0s", retention.FormatWithin(90*time.Minute))
		require.Empty(t, retention.FormatWithin(0))
	})
}

func TestPlan(t *testing.T) {
	t.Parallel()

	t.Run(`Given a plan with kept and forgotten snapshots
	When it is counted
	Then it reports both`, func(t *testing.T) {
		t.Parallel()
		plan := retention.Plan{
			{SnapshotID: "a", Reasons: []string{"last"}},
			{SnapshotID: "b"},
			{SnapshotID: "c"},
		}

		require.Equal(t, 1, plan.Kept())
		require.Equal(t, 2, plan.Forgotten())
		require.True(t, plan[0].Keep())
		require.False(t, plan[1].Keep())
	})
}

func TestForgetting(t *testing.T) {
	t.Parallel()

	t.Run(`Given valid rules
	When a forgetting finishes
	Then it records a snapshots forgotten event with the policy and the result`, func(t *testing.T) {
		t.Parallel()
		forgetting, err := retention.NewForgetting(retention.Rules{Last: 1})
		require.NoError(t, err)
		result := retention.ForgetResult{Forgotten: []retention.ForgottenSnapshot{{ID: "old"}}}

		forgetting.Forgot(result)

		events := forgetting.Events()
		require.Len(t, events, 1)
		forgotten, ok := events[0].(retention.SnapshotsForgotten)
		require.True(t, ok)
		require.Equal(t, retention.SnapshotsForgottenEventName, forgotten.EventName())
		require.Equal(t, result, forgotten.Result)
		require.Equal(t, 1, forgotten.Policy.Rules().Last)
	})

	t.Run(`Given empty rules
	When a forgetting is built
	Then it is rejected`, func(t *testing.T) {
		t.Parallel()

		_, err := retention.NewForgetting(retention.Rules{})

		require.ErrorIs(t, err, retention.ErrEmptyPolicy)
	})
}

func TestPruning(t *testing.T) {
	t.Parallel()

	t.Run(`Given a pruning
	When it finishes
	Then it records a repository pruned event with the result`, func(t *testing.T) {
		t.Parallel()
		pruning := retention.NewPruning()
		result := retention.PruneResult{RemovedObjects: 3, FreedBytes: 1024}

		pruning.Pruned(&result)

		pruned, ok := pruning.Events()[0].(retention.RepositoryPruned)
		require.True(t, ok)
		require.Equal(t, retention.RepositoryPrunedEventName, pruned.EventName())
		require.Equal(t, result, pruned.Result)
	})

	t.Run(`Given prune plans
	When they are checked
	Then a plan is empty only without unreferenced objects and old key generations`, func(t *testing.T) {
		t.Parallel()

		require.True(t, retention.PrunePlan{StoredObjects: 5, ReferencedObjects: 5}.Empty())
		require.False(t, retention.PrunePlan{UnreferencedObjects: 1}.Empty())
		require.False(t, retention.PrunePlan{OldKeyGenerations: []uint64{1}}.Empty())
	})
}

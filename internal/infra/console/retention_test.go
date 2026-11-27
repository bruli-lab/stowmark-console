package console

import (
	"context"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/bruli-lab/stowmark-console/internal/domain/retention"
	"github.com/stretchr/testify/require"
)

var forgetPlan = retention.Plan{
	{SnapshotID: "newest", CreatedAt: time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC), Source: "/data", Reasons: []string{"last"}},
	{SnapshotID: "oldest", CreatedAt: time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC), Source: "/data"},
}

func (h *harness) retentionForm() *retentionForm {
	form, _ := h.model.dialog.(*retentionForm)
	return form
}

func (h *harness) forgetScreen() *forgetScreen {
	screen, _ := h.model.top().(*forgetScreen)
	return screen
}

func (h *harness) planForget(t *testing.T, plan retention.Plan) {
	t.Helper()
	h.retention.PlanForgetFunc = func(context.Context, retention.Policy) (retention.Plan, error) { return plan, nil }
	h.command("forget")
	h.typeText("1")
	h.pressAndRun(key(tea.KeyEnter))
	require.NotNil(t, h.forgetScreen())
}

func TestConsoleForget(t *testing.T) {
	t.Parallel()

	t.Run(`Given a repository with a retention in the configuration
	When the user enters the forget command
	Then the form is prefilled with its rules`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true)
		rules := retention.Rules{Last: 7, Daily: 14, Within: 30 * 24 * time.Hour}
		h.profiles[0] = h.profiles[0].WithRetention(&rules)
		h.started()

		h.command("forget")

		form := h.retentionForm()
		require.NotNil(t, form)
		require.Equal(t, "7", form.fields[0].input.Value())
		require.Equal(t, "14", form.fields[1].input.Value())
		require.Equal(t, "30d", form.fields[5].input.Value())
		require.Contains(t, form.view(), "Prefilled with the retention of the repositories file.")
	})

	t.Run(`Given the forget form without rules
	When the user submits it
	Then it asks for at least one rule and stays open`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.command("forget")

		h.pressAndRun(key(tea.KeyEnter))

		require.NotNil(t, h.retentionForm())
		require.ErrorIs(t, h.retentionForm().err, retention.ErrEmptyPolicy)
		require.Empty(t, h.retention.PlanForgetCalls())
	})

	t.Run(`Given the forget form with a rule that is not a number
	When the user submits it
	Then it names the rule`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.command("forget")
		h.pressAndRun(key(tea.KeyDown))
		h.typeText("x")

		h.pressAndRun(key(tea.KeyEnter))

		require.EqualError(t, h.retentionForm().err, "Daily must be a positive whole number")
	})

	t.Run(`Given the forget form with an invalid keep within
	When the user submits it
	Then it explains the expected format`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.command("forget")
		h.pressAndRun(key(tea.KeyUp))
		h.typeText("3y")

		h.pressAndRun(key(tea.KeyEnter))

		require.ErrorIs(t, h.retentionForm().err, retention.ErrInvalidDuration)
	})

	t.Run(`Given the forget form
	When the user cancels it
	Then nothing is planned`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.command("forget")

		h.pressAndRun(key(tea.KeyEscape))

		require.Nil(t, h.model.dialog)
		require.Equal(t, "Forget cancelled", h.model.message)
	})

	t.Run(`Given the forget form with keep rules
	When the user submits it
	Then the plan screen lists what is kept, why, and what is forgotten`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()

		h.planForget(t, forgetPlan)

		require.Equal(t, 1, h.retention.PlanForgetCalls()[0].Policy.Rules().Last)
		view := h.view()
		require.Contains(t, view, "Forget plan (last 1): keep 1, forget 1")
		require.Contains(t, view, "✓ keep")
		require.Contains(t, view, "✗ forget")
		require.Contains(t, view, "newest")
	})

	t.Run(`Given the forget plan
	When the user forgets and confirms
	Then the snapshots are forgotten, the plan is closed and the snapshots are reloaded`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.retention.ForgetFunc = func(context.Context, *retention.Forgetting) (retention.ForgetResult, error) {
			return retention.ForgetResult{Forgotten: []retention.ForgottenSnapshot{{ID: "oldest"}}}, nil
		}
		h.planForget(t, forgetPlan)
		listed := len(h.snapshots.ListCalls())

		h.pressAndRun(runes("f"))
		require.Equal(t, "Forget 1 snapshot of default?", h.confirmation().title)
		h.pressAndRun(runes("y"))

		require.Len(t, h.retention.ForgetCalls(), 1)
		require.Empty(t, h.retention.PruneCalls())
		require.Nil(t, h.forgetScreen())
		require.Equal(t, "Forgot 1 snapshot", h.model.message)
		require.Len(t, h.snapshots.ListCalls(), listed+1)
	})

	t.Run(`Given the forget plan
	When the user forgets and prunes
	Then both run and the message reports the freed space`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.retention.ForgetFunc = func(context.Context, *retention.Forgetting) (retention.ForgetResult, error) {
			return retention.ForgetResult{Forgotten: []retention.ForgottenSnapshot{{ID: "oldest"}}}, nil
		}
		h.retention.PruneFunc = func(context.Context, *retention.Pruning) (retention.PruneResult, error) {
			return retention.PruneResult{RemovedObjects: 3, FreedBytes: 2048}, nil
		}
		h.planForget(t, forgetPlan)

		h.pressAndRun(runes("p"))
		require.Contains(t, h.confirmation().body, "Then prune frees the space")
		h.pressAndRun(runes("y"))

		require.Len(t, h.retention.PruneCalls(), 1)
		require.Equal(t, "Forgot 1 snapshot. Pruned: removed 3 objects and freed 2.0 KiB", h.model.message)
	})

	t.Run(`Given snapshots retained by object lock
	When they are forgotten
	Then the message says until when they are kept`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.retention.ForgetFunc = func(context.Context, *retention.Forgetting) (retention.ForgetResult, error) {
			return retention.ForgetResult{Postponed: []retention.PostponedSnapshot{
				{ID: "oldest", RetainedUntil: time.Date(2026, 12, 24, 0, 0, 0, 0, time.UTC)},
			}}, nil
		}
		h.planForget(t, forgetPlan)

		h.pressAndRun(runes("f"))
		h.pressAndRun(runes("y"))

		require.Equal(t, "Forgot 0 snapshots; 1 snapshot kept by object lock until 2026-12-24", h.model.message)
	})

	t.Run(`Given a plan that keeps every snapshot
	When the user tries to forget
	Then nothing is asked nor forgotten`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.planForget(t, forgetPlan[:1])

		h.pressAndRun(runes("f"))

		require.Nil(t, h.model.dialog)
		require.Empty(t, h.retention.ForgetCalls())
		require.Equal(t, "Nothing to forget: the policy keeps every snapshot", h.model.message)
	})

	t.Run(`Given the forget plan
	When the user goes back
	Then the snapshots list is shown again`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.planForget(t, forgetPlan)

		h.pressAndRun(key(tea.KeyEscape))

		require.Equal(t, h.model.snapshots, h.model.top())
	})
}

func TestConsolePrune(t *testing.T) {
	t.Parallel()

	t.Run(`Given a repository with unreferenced objects and old key generations
	When the user enters the prune command
	Then a confirmation shows the plan`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.retention.PlanPruneFunc = func(context.Context) (retention.PrunePlan, error) {
			return retention.PrunePlan{
				StoredObjects: 10, ReferencedObjects: 7, UnreferencedObjects: 3, UnreferencedBytes: 300,
				OldKeyGenerations: []uint64{1, 2},
			}, nil
		}

		h.command("prune")

		body := h.confirmation().body
		require.Equal(t, "Prune repository default?", h.confirmation().title)
		require.Contains(t, body, "Unreferenced objects  3 (300 B to free)")
		require.Contains(t, body, "Old key generations   1, 2")
		require.Contains(t, body, "exclusive lock")
		require.Empty(t, h.confirmation().warning)
	})

	t.Run(`Given the prune confirmation
	When the user confirms
	Then the repository is pruned and the message summarizes it`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.retention.PlanPruneFunc = func(context.Context) (retention.PrunePlan, error) {
			return retention.PrunePlan{UnreferencedObjects: 1}, nil
		}
		h.retention.PruneFunc = func(context.Context, *retention.Pruning) (retention.PruneResult, error) {
			return retention.PruneResult{
				RemovedObjects: 1, FreedBytes: 1024, RemovedKeyGenerations: []uint64{1},
				PostponedObjects: 2, PostponedBytes: 10, RetainedUntil: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
			}, nil
		}
		h.command("prune")

		h.pressAndRun(runes("y"))

		require.Len(t, h.retention.PruneCalls(), 1)
		require.Equal(t,
			"Pruned: removed 1 object and freed 1.0 KiB; removed key generations 1; 2 objects (10 B) kept by object lock until 2027-01-01",
			h.model.message)
	})

	t.Run(`Given the prune confirmation
	When the user declines
	Then nothing is pruned`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.retention.PlanPruneFunc = func(context.Context) (retention.PrunePlan, error) {
			return retention.PrunePlan{UnreferencedObjects: 1}, nil
		}
		h.command("prune")

		h.pressAndRun(runes("n"))

		require.Empty(t, h.retention.PruneCalls())
		require.Equal(t, "Prune cancelled", h.model.message)
	})

	t.Run(`Given a repository where every object is referenced but some are missing
	When the user enters the prune command
	Then nothing is asked and the message warns about the missing objects`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.retention.PlanPruneFunc = func(context.Context) (retention.PrunePlan, error) {
			return retention.PrunePlan{StoredObjects: 4, ReferencedObjects: 4, MissingObjects: 1}, nil
		}

		h.command("prune")

		require.Nil(t, h.model.dialog)
		require.Equal(t,
			"Nothing to prune: every stored object is referenced (4). Missing objects: 1 (the snapshots that use them are damaged).",
			h.model.message)
	})

	t.Run(`Given no open repository
	When the user enters the prune command
	Then it asks to choose a repository first`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).withRepository("second", t.TempDir()).started()

		h.command("prune")

		require.Empty(t, h.retention.PlanPruneCalls())
		require.Equal(t, "Choose a repository first", h.model.message)
	})

	t.Run(`Given an open repository
	When the console is drawn
	Then the panel lists the forget and prune commands`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true)
		h.model.width = 160
		h.started()

		view := h.view()

		require.Contains(t, view, ":forget")
		require.Contains(t, view, ":prune")
	})
}

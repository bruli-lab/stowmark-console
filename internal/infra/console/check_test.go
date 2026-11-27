package console

import (
	"context"
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/bruli-lab/stowmark-console/internal/domain/encryption"
	"github.com/bruli-lab/stowmark-console/internal/domain/integrity"
	"github.com/bruli-lab/stowmark-console/internal/domain/progress"
	"github.com/stretchr/testify/require"
)

func (h *harness) choice() *choiceDialog {
	choice, _ := h.model.dialog.(*choiceDialog)
	return choice
}

func (h *harness) checkScreen() *checkScreen {
	screen, _ := h.model.top().(*checkScreen)
	return screen
}

func (h *harness) checkedScopes() []string {
	scopes := make([]string, len(h.checker.CheckCalls()))
	for i, call := range h.checker.CheckCalls() {
		scopes[i] = call.Scope.String()
	}
	return scopes
}

func damagedReport() integrity.Report {
	return integrity.Report{
		Snapshots:           3,
		CorruptManifests:    []integrity.Problem{{ID: "broken", Reason: "unexpected EOF"}},
		StoredObjects:       10,
		ReferencedObjects:   9,
		UnreferencedObjects: 2,
		MissingObjects:      []integrity.MissingObject{{Hash: "abc123", Snapshots: []string{"second", "first"}}},
		StaleLocks:          1,
	}
}

func TestConsoleCheck(t *testing.T) {
	t.Parallel()

	t.Run(`Given an open repository
	When the user enters the check command
	Then it offers the structure, all the data or a part of it`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()

		h.command("check")

		require.NotNil(t, h.choice())
		view := h.view()
		require.Contains(t, view, "Structure only")
		require.Contains(t, view, "All data")
		require.Contains(t, view, "Part of the data")
		require.NotContains(t, view, "needs its private key")
	})

	t.Run(`Given an encrypted repository without its private key
	When the user enters the check command
	Then the choice warns that reading the data needs the key`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true)
		h.encrypted(t, 2, false)
		h.started()

		h.command("check")

		require.Contains(t, h.view(), "needs its private key")
	})

	t.Run(`Given the check choice
	When the user cancels it
	Then nothing is checked`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.command("check")

		h.pressAndRun(key(tea.KeyEscape))

		require.Nil(t, h.model.dialog)
		require.Empty(t, h.checker.CheckCalls())
		require.Contains(t, h.view(), "Check cancelled")
	})

	t.Run(`Given the check choice
	When the user chooses the structure
	Then it checks without reading data and shows a healthy repository`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.checkReport = integrity.Report{Snapshots: 2, StoredObjects: 5, ReferencedObjects: 5}
		h.command("check")

		h.pressAndRun(runes("s"))

		require.Equal(t, []string{"structure"}, h.checkedScopes())
		require.NotNil(t, h.checkScreen())
		view := h.view()
		require.Contains(t, view, "✓ Repository OK")
		require.Contains(t, view, "No problems found.")
		require.NotContains(t, view, "Read objects")
	})

	t.Run(`Given the check choice
	When the user chooses all the data
	Then it reads every object and shows the read and corrupt objects`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.checkReport = integrity.Report{
			ReadObjects:    4,
			CorruptObjects: []integrity.Problem{{ID: "def456", Reason: "hash mismatch"}},
		}
		h.command("check")

		h.pressAndRun(runes("d"))

		require.Equal(t, []string{"structure and all data"}, h.checkedScopes())
		view := h.view()
		require.Contains(t, view, "✗ Repository has errors")
		require.Contains(t, view, "Read objects")
		require.Contains(t, view, "object def456 cannot be read: hash mismatch")
	})

	t.Run(`Given the check choice
	When the user chooses a part of the data and accepts the default
	Then it reads ten per cent of the data`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.command("check")

		h.pressAndRun(runes("p"))
		require.NotNil(t, h.prompt())
		h.pressAndRun(key(tea.KeyEnter))

		require.Equal(t, []string{"structure and 10% of the data"}, h.checkedScopes())
	})

	t.Run(`Given the data subset prompt
	When the user enters an invalid subset
	Then the prompt stays open with the error`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.command("check")
		h.pressAndRun(runes("p"))
		h.clearInput()
		h.typeText("3/2")

		h.pressAndRun(key(tea.KeyEnter))

		require.NotNil(t, h.prompt())
		require.Contains(t, h.prompt().err, integrity.ErrInvalidSubset.Error())
		require.Empty(t, h.checker.CheckCalls())
	})

	t.Run(`Given a repository with damaged snapshots and warnings
	When it is checked
	Then the screen lists the damaged snapshots, the errors and the warnings with their shortcuts`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.checkReport = damagedReport()
		h.command("check")

		h.pressAndRun(runes("s"))

		view := h.view()
		require.Contains(t, view, "Damaged snapshots")
		require.Contains(t, view, "  broken")
		require.Contains(t, view, "  first")
		require.Contains(t, view, "manifest of snapshot broken cannot be read: unexpected EOF")
		require.Contains(t, view, "object abc123 is missing, used by second, first")
		require.Contains(t, view, "2 objects not used by any snapshot: [p] prune frees them")
		require.Contains(t, view, "1 stale lock: [u] unlock removes them")
		require.Contains(t, h.checkScreen().help(), "[p] Prune")
		require.Contains(t, h.checkScreen().help(), "[u] Unlock")
	})

	t.Run(`Given the result of a check with unreferenced objects
	When the user presses p
	Then it plans the prune`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.checkReport = damagedReport()
		h.command("check")
		h.pressAndRun(runes("s"))

		h.pressAndRun(runes("p"))

		require.Len(t, h.retention.PlanPruneCalls(), 1)
	})

	t.Run(`Given the result of a check with stale locks
	When the user presses u
	Then it removes the stale locks`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.checkReport = damagedReport()
		h.command("check")
		h.pressAndRun(runes("s"))

		h.pressAndRun(runes("u"))

		calls := h.repositories.UnlockCalls()
		require.Len(t, calls, 1)
		require.False(t, calls[0].All)
	})

	t.Run(`Given the result of a healthy check
	When the user presses p or u
	Then nothing happens`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.command("check")
		h.pressAndRun(runes("s"))

		h.pressAndRun(runes("p"))
		h.pressAndRun(runes("u"))

		require.Empty(t, h.retention.PlanPruneCalls())
		require.Empty(t, h.repositories.UnlockCalls())
		require.NotContains(t, h.checkScreen().help(), "[p] Prune")
	})

	t.Run(`Given the result of a check
	When the user presses r
	Then it checks again with the same scope and replaces the screen`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.command("check")
		h.pressAndRun(runes("d"))
		depth := len(h.model.screens)

		h.pressAndRun(runes("r"))

		require.Equal(t, []string{"structure and all data", "structure and all data"}, h.checkedScopes())
		require.Len(t, h.model.screens, depth)
	})

	t.Run(`Given the result of a check
	When the user presses esc
	Then it goes back to the snapshots`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.command("check")
		h.pressAndRun(runes("s"))

		h.pressAndRun(key(tea.KeyEscape))

		require.Nil(t, h.checkScreen())
	})

	t.Run(`Given an encrypted repository without its private key
	When the data check fails for the key
	Then it explains which key is missing`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true)
		h.encrypted(t, 2, false)
		h.started()
		h.checker.CheckFunc = func(context.Context, integrity.Scope) (integrity.Report, error) {
			return integrity.Report{}, encryption.ErrPrivateKeyRequired
		}
		h.command("check")

		h.pressAndRun(runes("d"))

		require.Nil(t, h.checkScreen())
		require.ErrorContains(t, h.model.err, "private key")
	})

	t.Run(`Given a check that fails
	When the result arrives
	Then the error is shown`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.checker.CheckFunc = func(context.Context, integrity.Scope) (integrity.Report, error) {
			return integrity.Report{}, errors.New("boom")
		}
		h.command("check")

		h.pressAndRun(runes("s"))

		require.ErrorContains(t, h.model.err, "check repository default")
	})

	t.Run(`Given a running data check
	When the reading progress is reported
	Then the footer shows the reading stage`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.withClock()
		h.command("check")
		intent := h.press(runes("d"))
		h.model.Update(intent())

		h.report(progress.Started(progress.StageReadingData, 8, 4096).Advanced(2, 1024))

		footer := h.model.renderFooter()
		require.Contains(t, footer, "Reading data")
		require.Contains(t, footer, " 25%")
	})
}

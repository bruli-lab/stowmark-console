package console

import (
	"context"
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
	"github.com/stretchr/testify/require"
)

func TestConsoleLocks(t *testing.T) {
	t.Parallel()

	t.Run(`Given an open repository with a stale and an active lock
	When the user enters the unlock command
	Then only the stale locks are removed and the message reports the active ones`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.lockReport = repository.LockReport{Removed: 1, Active: 1}

		h.command("unlock")

		require.Len(t, h.repositories.UnlockCalls(), 1)
		require.False(t, h.repositories.UnlockCalls()[0].All)
		require.Equal(t,
			"Removed 1 stale lock from repository default; 1 lock still held by running processes (:unlock! removes them)",
			h.model.message)
	})

	t.Run(`Given an open repository without locks
	When the user enters the unlock command
	Then the message says that no lock was removed`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()

		h.command("unlock")

		require.Equal(t, "Removed 0 stale locks from repository default", h.model.message)
	})

	t.Run(`Given an open repository
	When the user enters the unlock every lock command
	Then a confirmation warns about breaking running commands`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()

		h.command("unlock!")

		require.NotNil(t, h.confirmation())
		require.Equal(t, "Remove every lock of default?", h.confirmation().title)
		require.Contains(t, h.confirmation().warning, "can\ncorrupt the repository")
		require.Empty(t, h.repositories.UnlockCalls())
	})

	t.Run(`Given the unlock every lock confirmation
	When the user confirms
	Then every lock is removed`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.lockReport = repository.LockReport{Removed: 2}
		h.command("unlock!")

		h.pressAndRun(runes("y"))

		require.True(t, h.repositories.UnlockCalls()[0].All)
		require.Equal(t, "Removed 2 locks from repository default", h.model.message)
	})

	t.Run(`Given the unlock every lock confirmation
	When the user declines
	Then no lock is removed`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.command("unlock!")

		h.pressAndRun(key(tea.KeyEscape))

		require.Empty(t, h.repositories.UnlockCalls())
		require.Equal(t, "Unlock cancelled", h.model.message)
	})

	t.Run(`Given no open repository
	When the user enters the unlock command
	Then it asks to choose a repository first`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).withRepository("second", t.TempDir()).started()

		h.command("unlock")

		require.Empty(t, h.repositories.UnlockCalls())
		require.Equal(t, "Choose a repository first", h.model.message)
	})

	t.Run(`Given a repository locked by another process
	When the user creates a snapshot
	Then the error names the holder and suggests the unlock commands`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.snapshots.CreateFunc = func(context.Context, *snapshot.Creation) (snapshot.Snapshot, []snapshot.SkippedFile, error) {
			return snapshot.Snapshot{}, nil, fmt.Errorf("%w: exclusive lock held by nas (pid 42)", repository.ErrLocked)
		}
		h.pressAndRun(runes("c"))
		h.clearInput()
		h.typeText(t.TempDir())
		h.pressAndRun(key(tea.KeyEnter))

		h.pressAndRun(key(tea.KeyEnter))

		require.ErrorIs(t, h.model.err, repository.ErrLocked)
		require.ErrorContains(t, h.model.err, "held by nas (pid 42)")
		require.ErrorContains(t, h.model.err, ":unlock removes the stale locks and :unlock! every lock")
	})

	t.Run(`Given an open repository
	When the console is drawn
	Then the panel lists the unlock command but not the one that removes every lock`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()

		view := h.view()

		require.Contains(t, view, ":unlock")
		require.NotContains(t, view, ":unlock!")
	})
}

package console

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
	"github.com/bruli-lab/stowmark-console/internal/fixtures"
	"github.com/stretchr/testify/require"
)

func (h *harness) createOptions() *createOptionsDialog {
	dialog, _ := h.model.dialog.(*createOptionsDialog)
	return dialog
}

func (h *harness) chooseOrigin(t *testing.T, source string) {
	t.Helper()
	h.pressAndRun(runes("c"))
	h.clearInput()
	h.typeText(source)
	h.pressAndRun(key(tea.KeyEnter))
	require.NotNil(t, h.createOptions())
}

func (h *harness) createdWith(t *testing.T) snapshot.Exclusions {
	t.Helper()
	calls := h.snapshots.CreateCalls()
	require.Len(t, calls, 1)
	return calls[0].Creation.Exclusions()
}

func space() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
}

func TestConsoleCreateOptions(t *testing.T) {
	t.Parallel()

	t.Run(`Given a repository without configured exclusions
	When the user chooses the origin
	Then the options are empty and read the ignore files`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()

		h.chooseOrigin(t, t.TempDir())

		view := h.view()
		require.Contains(t, view, "Exclude")
		require.Contains(t, view, "[x] Read .stowmarkignore files")
		require.NotContains(t, view, "Prefilled")
		require.Empty(t, h.snapshots.CreateCalls())
	})

	t.Run(`Given a repository with exclusions in the configuration
	When the user chooses the origin
	Then the options are prefilled with them`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true)
		h.profiles[0] = h.profiles[0].WithExclusions(snapshot.NewExclusions([]string{"*.tmp", "cache/"}, true))
		h.started()

		h.chooseOrigin(t, t.TempDir())

		require.Equal(t, "*.tmp, cache/", h.createOptions().exclude.Value())
		view := h.view()
		require.Contains(t, view, "[ ] Read .stowmarkignore files")
		require.Contains(t, view, "Prefilled with the exclusions of the repositories file.")
	})

	t.Run(`Given an origin with a .stowmarkignore file
	When the options are shown
	Then they say so`, func(t *testing.T) {
		t.Parallel()
		source := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(source, snapshot.IgnoreFileName), []byte("*.log\n"), 0o600))
		h := newHarness(t, true).started()

		h.chooseOrigin(t, source)

		require.Contains(t, h.view(), "the origin has a .stowmarkignore file")
	})

	t.Run(`Given the create options
	When the user types patterns, skips the ignore files and creates
	Then the snapshot is created with those exclusions`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.chooseOrigin(t, t.TempDir())
		h.typeText("*.tmp, node_modules/")
		h.pressAndRun(key(tea.KeyTab))
		h.pressAndRun(space())

		h.pressAndRun(key(tea.KeyEnter))

		exclusions := h.createdWith(t)
		require.Equal(t, []string{"*.tmp", "node_modules/"}, exclusions.Patterns())
		require.True(t, exclusions.SkipsIgnoreFiles())
	})

	t.Run(`Given the create options with the toggle focused
	When the user types
	Then the patterns do not change`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.chooseOrigin(t, t.TempDir())
		h.pressAndRun(key(tea.KeyDown))

		h.typeText("x")

		require.Empty(t, h.createOptions().exclude.Value())
	})

	t.Run(`Given the create options
	When the user creates without changes
	Then the snapshot is created without exclusions`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.chooseOrigin(t, t.TempDir())

		h.pressAndRun(key(tea.KeyEnter))

		require.True(t, h.createdWith(t).Empty())
	})

	t.Run(`Given the create options
	When the user cancels them
	Then nothing is created`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.chooseOrigin(t, t.TempDir())

		h.pressAndRun(key(tea.KeyEsc))

		require.Nil(t, h.model.dialog)
		require.Empty(t, h.snapshots.CreateCalls())
		require.Equal(t, "Snapshot creation cancelled", h.model.message)
	})

	t.Run(`Given a snapshot whose origin has entries Stowmark skips
	When it is created
	Then the message counts them by kind and a dialog lists them`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		created := fixtures.Snapshot().WithID("new-snapshot").Build()
		h.snapshots.CreateFunc = func(context.Context, *snapshot.Creation) (snapshot.Snapshot, []snapshot.SkippedFile, error) {
			return created, []snapshot.SkippedFile{
				{Path: "run/app.sock", Kind: "socket"},
				{Path: "run/db.sock", Kind: "socket"},
				{Path: "dev/tty", Kind: "device"},
			}, nil
		}
		h.chooseOrigin(t, t.TempDir())

		h.pressAndRun(key(tea.KeyEnter))

		require.Contains(t, h.model.message, "3 skipped (1 device, 2 sockets)")
		dialog, ok := h.model.dialog.(*infoDialog)
		require.True(t, ok)
		require.Contains(t, dialog.view(), "run/app.sock")
		require.Contains(t, dialog.view(), "dev/tty")
	})

	t.Run(`Given a snapshot with more skipped entries than the dialog lists
	When it is created
	Then the dialog says how many more there are`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		skipped := make([]snapshot.SkippedFile, maxListedSkipped+3)
		for i := range skipped {
			skipped[i] = snapshot.SkippedFile{Path: fmt.Sprintf("pipe-%d", i), Kind: "named pipe"}
		}
		h.snapshots.CreateFunc = func(context.Context, *snapshot.Creation) (snapshot.Snapshot, []snapshot.SkippedFile, error) {
			return fixtures.Snapshot().WithID("new-snapshot").Build(), skipped, nil
		}
		h.chooseOrigin(t, t.TempDir())

		h.pressAndRun(key(tea.KeyEnter))

		require.Contains(t, h.model.dialog.view(), "… and 3 more")
	})

	t.Run(`Given an invalid exclude pattern
	When the snapshot is created
	Then the error is shown`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.snapshots.CreateFunc = func(context.Context, *snapshot.Creation) (snapshot.Snapshot, []snapshot.SkippedFile, error) {
			return snapshot.Snapshot{}, nil, fmt.Errorf("%w: \"a//b\"", snapshot.ErrInvalidExcludePattern)
		}
		h.chooseOrigin(t, t.TempDir())
		h.typeText("a//b")

		h.pressAndRun(key(tea.KeyEnter))

		require.ErrorIs(t, h.model.err, snapshot.ErrInvalidExcludePattern)
	})
}

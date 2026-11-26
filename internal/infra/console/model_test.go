package console

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

func runes(value string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)}
}

func key(keyType tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: keyType}
}

func modelAt(t *testing.T, repositoryPath string) *Model {
	t.Helper()
	model := New(t.Context(), repositoryPath)
	return &model
}

func snapshotsModel(t *testing.T) *Model {
	t.Helper()
	root := t.TempDir()
	model := modelAt(t, root)
	model.loading = false
	model.currentPath = filepath.Join(root, "snapshots")
	model.entries = []repository.Entry{*repository.NewEntry("snapshot-1.json", false, 128)}
	return model
}

func isQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

func TestModelInit(t *testing.T) {
	t.Parallel()

	t.Run(`Given a path without a Stowmark repository
	When the console starts
	Then it asks for confirmation before initializing anything`, func(t *testing.T) {
		t.Parallel()
		model := modelAt(t, filepath.Join(t.TempDir(), "missing"))

		cmd := model.Init()

		require.Nil(t, cmd)
		require.True(t, model.confirmingInit)
		require.False(t, model.loading)
		require.Empty(t, model.initWarning)
	})

	t.Run(`Given a non empty directory without a Stowmark repository
	When the console starts
	Then the confirmation warns that the directory is not empty`, func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(root, "file.txt"), []byte("data"), 0o600))
		model := modelAt(t, root)

		model.Init()

		require.True(t, model.confirmingInit)
		require.NotEmpty(t, model.initWarning)
	})

	t.Run(`Given an existing Stowmark repository
	When the console starts
	Then it loads the repository without asking`, func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(root, "config.json"), []byte("{}"), 0o600))
		model := modelAt(t, root)

		cmd := model.Init()

		require.NotNil(t, cmd)
		require.False(t, model.confirmingInit)
	})
}

func TestModelInitConfirmation(t *testing.T) {
	t.Parallel()

	t.Run(`Given the initialization confirmation
	When the user rejects it
	Then the console quits`, func(t *testing.T) {
		t.Parallel()
		model := modelAt(t, filepath.Join(t.TempDir(), "missing"))
		model.Init()

		_, cmd := model.Update(runes("n"))

		require.True(t, isQuit(cmd))
	})

	t.Run(`Given the initialization confirmation
	When the user accepts it
	Then the repository initialization starts`, func(t *testing.T) {
		t.Parallel()
		model := modelAt(t, filepath.Join(t.TempDir(), "missing"))
		model.Init()

		_, cmd := model.Update(runes("y"))

		require.NotNil(t, cmd)
		require.False(t, model.confirmingInit)
		require.True(t, model.loading)
	})

	t.Run(`Given the initialization confirmation
	When the user presses enter
	Then nothing is initialized`, func(t *testing.T) {
		t.Parallel()
		model := modelAt(t, filepath.Join(t.TempDir(), "missing"))
		model.Init()

		_, cmd := model.Update(key(tea.KeyEnter))

		require.Nil(t, cmd)
		require.True(t, model.confirmingInit)
	})
}

func TestModelRestore(t *testing.T) {
	t.Parallel()

	t.Run(`Given a selected snapshot
	When the user asks to restore it
	Then a destination form is shown with a default path that is not the origin`, func(t *testing.T) {
		t.Parallel()
		model := snapshotsModel(t)

		_, cmd := model.Update(runes("x"))

		require.Nil(t, cmd)
		require.NotNil(t, model.restoreForm)
		require.Equal(t, "snapshot-1", model.restore.snapshotID)
		require.True(t, filepath.IsAbs(model.restoreForm.value))
		require.Equal(t, "snapshot-1", filepath.Base(model.restoreForm.value))
	})

	t.Run(`Given the destination form with an empty destination
	When the user submits it
	Then an error is shown and the restore is not confirmed`, func(t *testing.T) {
		t.Parallel()
		model := snapshotsModel(t)
		model.Update(runes("x"))
		model.restoreForm.value = "  "

		model.Update(key(tea.KeyEnter))

		require.NotNil(t, model.restoreForm)
		require.NotEmpty(t, model.restoreForm.err)
		require.False(t, model.confirmingRestore())
	})

	t.Run(`Given the destination form with a non empty directory
	When the user submits it
	Then the confirmation shows the absolute destination and an overwrite warning`, func(t *testing.T) {
		t.Parallel()
		destination := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(destination, "file.txt"), []byte("data"), 0o600))
		model := snapshotsModel(t)
		model.Update(runes("x"))
		model.restoreForm.value = destination

		model.Update(key(tea.KeyEnter))

		require.True(t, model.confirmingRestore())
		require.Equal(t, destination, model.restore.destination)
		require.NotEmpty(t, model.restore.warning)
	})

	t.Run(`Given the restore confirmation
	When the user presses enter
	Then the restore does not start`, func(t *testing.T) {
		t.Parallel()
		model := snapshotsModel(t)
		model.Update(runes("x"))
		model.Update(key(tea.KeyEnter))

		_, cmd := model.Update(key(tea.KeyEnter))

		require.Nil(t, cmd)
		require.True(t, model.confirmingRestore())
	})

	t.Run(`Given the restore confirmation
	When the user confirms with y
	Then the restore starts`, func(t *testing.T) {
		t.Parallel()
		model := snapshotsModel(t)
		model.Update(runes("x"))
		model.Update(key(tea.KeyEnter))

		_, cmd := model.Update(runes("y"))

		require.NotNil(t, cmd)
		require.Nil(t, model.restore)
		require.True(t, model.loading)
	})

	t.Run(`Given the restore confirmation
	When the user cancels it
	Then no restore is pending`, func(t *testing.T) {
		t.Parallel()
		model := snapshotsModel(t)
		model.Update(runes("x"))
		model.Update(key(tea.KeyEnter))

		_, cmd := model.Update(runes("n"))

		require.Nil(t, cmd)
		require.Nil(t, model.restore)
		require.Equal(t, "Restore cancelled", model.message)
	})
}

func TestModelBrowser(t *testing.T) {
	t.Parallel()

	t.Run(`Given an operation in progress
	When the user asks to restore a snapshot
	Then the action is ignored`, func(t *testing.T) {
		t.Parallel()
		model := snapshotsModel(t)
		model.loading = true

		_, cmd := model.Update(runes("x"))

		require.Nil(t, cmd)
		require.Nil(t, model.restoreForm)
	})

	t.Run(`Given the repository browser
	When the user presses e
	Then no editor is opened`, func(t *testing.T) {
		t.Parallel()
		model := snapshotsModel(t)

		_, cmd := model.Update(runes("e"))

		require.Nil(t, cmd)
	})

	t.Run(`Given the repository root
	When the user goes back
	Then the console does not get stuck loading`, func(t *testing.T) {
		t.Parallel()
		model := modelAt(t, t.TempDir())
		model.loading = false

		_, cmd := model.Update(key(tea.KeyEsc))

		require.Nil(t, cmd)
		require.False(t, model.loading)
	})
}

func TestTextPrompt(t *testing.T) {
	t.Parallel()

	t.Run(`Given an empty prompt
	When the user types a path with spaces
	Then the spaces are kept`, func(t *testing.T) {
		t.Parallel()
		prompt := newTextPrompt("title", "label", "")

		prompt.handleKey(runes("my"))
		prompt.handleKey(key(tea.KeySpace))
		prompt.handleKey(runes("dir"))

		require.Equal(t, "my dir", prompt.value)
	})
}

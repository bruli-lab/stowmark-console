package console

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/app"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
	"github.com/bruli-lab/stowmark-console/internal/fixtures"
	"github.com/bruli-lab/stowmark-console/internal/mocks"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

const snapshotID = "20261126T160034Z-f482b24d"

type harness struct {
	model        *Model
	repositories *mocks.RepositoryMock
	snapshots    *mocks.SnapshotRepositoryMock
}

func newHarness(t *testing.T, initialized bool) *harness {
	t.Helper()
	repositories := &mocks.RepositoryMock{
		InitializedFunc: func(context.Context) (bool, error) { return initialized, nil },
		InitializeFunc:  func(context.Context, repository.InitOptions) error { return nil },
	}
	snapshots := &mocks.SnapshotRepositoryMock{
		ListFunc: func(context.Context) ([]snapshot.Snapshot, error) {
			return []snapshot.Snapshot{fixtures.Snapshot().WithID(snapshotID).Build()}, nil
		},
		ManifestFunc: func(context.Context, string) (snapshot.Manifest, error) {
			return fixtures.Manifest().WithSnapshotID(snapshotID).Build(), nil
		},
		FileContentFunc: func(context.Context, string, string, int64) ([]byte, error) {
			return []byte("line 1\nline 2"), nil
		},
		VerifyFunc: func(ctx context.Context, _ string) ([]snapshot.Failure, error) {
			return nil, ctx.Err()
		},
		RestoreFunc: func(context.Context, string, string) ([]snapshot.Failure, error) { return nil, nil },
	}
	model := New(t.Context(), Settings{
		RepositoryPath: filepath.Join(t.TempDir(), "repository"),
		Init:           repository.NewInitOptions("zstd", nil, nil),
	}, app.NewHandlers(repositories, snapshots))
	model.width, model.height = 120, 40

	return &harness{model: &model, repositories: repositories, snapshots: snapshots}
}

func (h *harness) run(cmd tea.Cmd) {
	for cmd != nil {
		msg := cmd()
		if _, quit := msg.(tea.QuitMsg); quit {
			return
		}
		_, cmd = h.model.Update(msg)
	}
}

func (h *harness) press(msg tea.KeyMsg) tea.Cmd {
	_, cmd := h.model.Update(msg)
	return cmd
}

func (h *harness) pressAndRun(msg tea.KeyMsg) {
	h.run(h.press(msg))
}

func (h *harness) started() *harness {
	h.run(h.model.Init())
	return h
}

func runes(value string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)}
}

func key(keyType tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: keyType}
}

func isQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

func TestConsoleStartup(t *testing.T) {
	t.Parallel()

	t.Run(`Given a repository that is not initialized
	When the console starts
	Then it asks for confirmation showing the init options`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, false).started()

		require.True(t, h.model.confirmingInit)
		require.Contains(t, h.model.View(), "zstd")
		require.Empty(t, h.repositories.InitializeCalls())
	})

	t.Run(`Given the initialization confirmation
	When the user rejects it
	Then the console quits without initializing`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, false).started()

		cmd := h.press(runes("n"))

		require.True(t, isQuit(cmd))
		require.Empty(t, h.repositories.InitializeCalls())
	})

	t.Run(`Given the initialization confirmation
	When the user presses enter
	Then nothing is initialized`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, false).started()

		h.pressAndRun(key(tea.KeyEnter))

		require.True(t, h.model.confirmingInit)
		require.Empty(t, h.repositories.InitializeCalls())
	})

	t.Run(`Given the initialization confirmation
	When the user accepts it
	Then the repository is initialized with the configured options and the snapshots are listed`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, false).started()

		h.pressAndRun(runes("y"))

		require.Len(t, h.repositories.InitializeCalls(), 1)
		require.Equal(t, "zstd", h.repositories.InitializeCalls()[0].Options.Compression())
		require.Len(t, h.snapshots.ListCalls(), 1)
		require.Equal(t, "Repository initialized successfully", h.model.message)
	})

	t.Run(`Given an initialized repository
	When the console starts
	Then the snapshots are listed`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()

		require.False(t, h.model.confirmingInit)
		require.Len(t, h.model.snapshots, 1)
		require.Contains(t, h.model.View(), snapshotID)
	})

	t.Run(`Given a repository that cannot be opened
	When the console starts
	Then the error is shown`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true)
		h.repositories.InitializedFunc = func(context.Context) (bool, error) { return false, errors.New("unreachable") }

		h.started()

		require.ErrorContains(t, h.model.err, "unreachable")
		require.False(t, h.model.loading)
	})
}

func TestConsoleBrowsing(t *testing.T) {
	t.Parallel()

	t.Run(`Given the snapshots list
	When the user opens a snapshot
	Then its root files are shown`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()

		h.pressAndRun(key(tea.KeyEnter))

		require.Equal(t, filesView, h.model.view)
		require.Equal(t, snapshotID, h.snapshots.ManifestCalls()[0].SnapshotID)
		require.Equal(t, "notes", h.model.entries[0].Name())
		require.Equal(t, "README.txt", h.model.entries[1].Name())
	})

	t.Run(`Given a nested directory of a snapshot
	When the user goes back
	Then the parent is shown with the directory selected, and back again returns to the snapshots`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.pressAndRun(key(tea.KeyEnter))
		h.pressAndRun(key(tea.KeyEnter))
		require.Equal(t, "notes", h.model.directory)

		h.pressAndRun(key(tea.KeyEsc))

		require.Equal(t, "", h.model.directory)
		require.Equal(t, "notes", h.model.entries[h.model.fileCursor].Name())

		h.pressAndRun(key(tea.KeyEsc))

		require.Equal(t, snapshotsView, h.model.view)
	})

	t.Run(`Given a text file of a snapshot
	When the user opens it
	Then its content is previewed from the snapshot`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.pressAndRun(key(tea.KeyEnter))
		h.pressAndRun(runes("j"))

		h.pressAndRun(key(tea.KeyEnter))

		require.NotNil(t, h.model.preview)
		require.Equal(t, []string{"line 1", "line 2"}, h.model.preview.lines)
		call := h.snapshots.FileContentCalls()[0]
		require.Equal(t, "/home/user/documents/README.txt", call.Path)
		require.Equal(t, int64(maxPreviewSize), call.MaxSize)
	})

	t.Run(`Given a binary file of a snapshot
	When the user opens it
	Then an error is shown instead of a preview`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.snapshots.FileContentFunc = func(context.Context, string, string, int64) ([]byte, error) {
			return []byte{0x89, 'P', 'N', 'G', 0x00, 0x01}, nil
		}
		h.pressAndRun(key(tea.KeyEnter))
		h.pressAndRun(runes("j"))

		h.pressAndRun(key(tea.KeyEnter))

		require.Nil(t, h.model.preview)
		require.ErrorContains(t, h.model.err, "not a text file")
	})
}

func TestConsoleRestore(t *testing.T) {
	t.Parallel()

	t.Run(`Given a selected snapshot
	When the user asks to restore it
	Then a destination form is shown with a default path that is not the origin`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()

		h.pressAndRun(runes("x"))

		require.NotNil(t, h.model.restoreForm)
		require.True(t, filepath.IsAbs(h.model.restoreForm.value))
		require.Equal(t, snapshotID, filepath.Base(h.model.restoreForm.value))
	})

	t.Run(`Given the destination form with an empty destination
	When the user submits it
	Then an error is shown and nothing is restored`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.pressAndRun(runes("x"))
		h.model.restoreForm.value = "  "

		h.pressAndRun(key(tea.KeyEnter))

		require.NotEmpty(t, h.model.restoreForm.err)
		require.False(t, h.model.confirmingRestore())
	})

	t.Run(`Given the destination form with a non empty directory
	When the user submits it
	Then the confirmation warns that files will be overwritten`, func(t *testing.T) {
		t.Parallel()
		destination := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(destination, "file.txt"), []byte("data"), 0o600))
		h := newHarness(t, true).started()
		h.pressAndRun(runes("x"))
		h.model.restoreForm.value = destination

		h.pressAndRun(key(tea.KeyEnter))

		require.True(t, h.model.confirmingRestore())
		require.Equal(t, destination, h.model.restore.destination)
		require.NotEmpty(t, h.model.restore.warning)
	})

	t.Run(`Given the restore confirmation
	When the user presses enter
	Then the restore does not start`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.pressAndRun(runes("x"))
		h.pressAndRun(key(tea.KeyEnter))

		h.pressAndRun(key(tea.KeyEnter))

		require.True(t, h.model.confirmingRestore())
		require.Empty(t, h.snapshots.RestoreCalls())
	})

	t.Run(`Given the restore confirmation
	When the user confirms with y
	Then the snapshot is restored to the chosen destination`, func(t *testing.T) {
		t.Parallel()
		destination := filepath.Join(t.TempDir(), "restore")
		h := newHarness(t, true).started()
		h.pressAndRun(runes("x"))
		h.model.restoreForm.value = destination
		h.pressAndRun(key(tea.KeyEnter))

		h.pressAndRun(runes("y"))

		require.Len(t, h.snapshots.RestoreCalls(), 1)
		require.Equal(t, destination, h.snapshots.RestoreCalls()[0].Destination)
		require.Contains(t, h.model.message, "restored to "+destination)
	})

	t.Run(`Given a restore that fails for some files
	When it finishes
	Then the failures are shown as an error`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.snapshots.RestoreFunc = func(context.Context, string, string) ([]snapshot.Failure, error) {
			return []snapshot.Failure{{Path: "/data/a.txt", Reason: "permission denied"}}, nil
		}
		h.pressAndRun(runes("x"))
		h.pressAndRun(key(tea.KeyEnter))

		h.pressAndRun(runes("y"))

		require.ErrorContains(t, h.model.err, "permission denied")
	})

	t.Run(`Given the restore confirmation
	When the user cancels it
	Then nothing is restored`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.pressAndRun(runes("x"))
		h.pressAndRun(key(tea.KeyEnter))

		h.pressAndRun(runes("n"))

		require.Nil(t, h.model.restore)
		require.Empty(t, h.snapshots.RestoreCalls())
		require.Equal(t, "Restore cancelled", h.model.message)
	})
}

func TestConsoleOperations(t *testing.T) {
	t.Parallel()

	t.Run(`Given the create form with a source
	When the user submits it
	Then the snapshot is created from the absolute source and selected in the list`, func(t *testing.T) {
		t.Parallel()
		source := t.TempDir()
		h := newHarness(t, true).started()
		h.snapshots.CreateFunc = func(context.Context, string) (snapshot.Snapshot, error) {
			return fixtures.Snapshot().WithID("new-snapshot").Build(), nil
		}
		h.pressAndRun(runes("c"))
		h.model.snapshotForm.value = source

		h.pressAndRun(key(tea.KeyEnter))

		require.Equal(t, source, h.snapshots.CreateCalls()[0].Source)
		require.Contains(t, h.model.message, "Snapshot new-snapshot created")
		require.Len(t, h.snapshots.ListCalls(), 2)
	})

	t.Run(`Given an operation in progress
	When the user asks to restore a snapshot
	Then the action is ignored`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.press(runes("v"))

		h.press(runes("x"))

		require.True(t, h.model.loading)
		require.Nil(t, h.model.restoreForm)
	})

	t.Run(`Given a verification in progress
	When the user presses escape
	Then the operation is cancelled`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		cmd := h.press(runes("v"))

		h.press(key(tea.KeyEsc))
		h.run(cmd)

		require.False(t, h.model.loading)
		require.NoError(t, h.model.err)
		require.Contains(t, h.model.message, "Cancelled")
	})

	t.Run(`Given a verification that succeeds
	When it finishes
	Then a success message is shown`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()

		h.pressAndRun(runes("v"))

		require.Equal(t, snapshotID, h.snapshots.VerifyCalls()[0].SnapshotID)
		require.Contains(t, h.model.message, "verified successfully")
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

func TestIsText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		content  []byte
		expected bool
	}{
		{name: "empty content, then it is text", content: nil, expected: true},
		{name: "utf8 text, then it is text", content: []byte("àèéíòóú €"), expected: true},
		{name: "utf8 text cut in the middle of a rune, then it is text", content: []byte("àèé")[:5], expected: true},
		{name: "content with null bytes, then it is not text", content: []byte("a\x00b"), expected: false},
		{name: "a png header, then it is not text", content: []byte("\x89PNG\r\n\x1a\n"), expected: false},
	}

	for _, tt := range tests {
		t.Run(`Given `+tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.expected, isText(tt.content))
		})
	}
}

func TestTruncate(t *testing.T) {
	t.Parallel()

	t.Run(`Given a string with multibyte characters wider than the limit
	When it is truncated
	Then the result keeps valid characters and fits the width`, func(t *testing.T) {
		t.Parallel()

		truncated := truncate("àèéíòóú", 4)

		require.Equal(t, "àèé…", truncated)
	})
}

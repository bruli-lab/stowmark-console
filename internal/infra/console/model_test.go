package console

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf8"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bruli-lab/stowmark-console/internal/app"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
	"github.com/bruli-lab/stowmark-console/internal/fixtures"
	"github.com/bruli-lab/stowmark-console/internal/mocks"
	"github.com/stretchr/testify/require"
)

const snapshotID = "20261126T160034Z-f482b24d"

type harness struct {
	model        *Model
	profiles     []repository.Profile
	catalog      *mocks.CatalogMock
	repositories *mocks.RepositoryMock
	snapshots    *mocks.SnapshotRepositoryMock
}

var errUnreachable = errors.New("unreachable")

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
	h := &harness{
		profiles: []repository.Profile{
			repository.NewProfile("default", filepath.Join(t.TempDir(), "repository"), repository.NewInitOptions("zstd", nil, nil)),
		},
		repositories: repositories,
		snapshots:    snapshots,
	}
	h.catalog = &mocks.CatalogMock{
		ProfilesFunc: func() []repository.Profile { return h.profiles },
		OpenFunc: func(_ context.Context, name string) (repository.Profile, error) {
			for _, profile := range h.profiles {
				if profile.Name() == name && profile.Kind() != "broken" {
					return profile, nil
				}
			}
			return repository.Profile{}, errUnreachable
		},
	}
	handlers := app.NewHandlers(h.catalog, repositories, snapshots)
	model := New(t.Context(), &handlers)
	model.width, model.height = 120, 40
	h.model = &model

	return h
}

func (h *harness) withRepository(name, location string) *harness {
	h.profiles = append(h.profiles, repository.NewProfile(name, location, repository.NewInitOptions("", nil, nil)))
	return h
}

func (h *harness) repositoriesScreen() *repositoriesScreen {
	screen, _ := h.model.top().(*repositoriesScreen)
	return screen
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

func (h *harness) press(msg tea.KeyPressMsg) tea.Cmd {
	_, cmd := h.model.Update(msg)
	return cmd
}

func (h *harness) pressAndRun(msg tea.KeyPressMsg) {
	h.run(h.press(msg))
}

func (h *harness) typeText(value string) {
	for _, char := range value {
		h.pressAndRun(runes(string(char)))
	}
}

func (h *harness) started() *harness {
	h.run(h.model.Init())
	return h
}

func (h *harness) view() string {
	return h.model.View().Content
}

func (h *harness) files() *filesScreen {
	files, _ := h.model.top().(*filesScreen)
	return files
}

func (h *harness) preview() *previewScreen {
	preview, _ := h.model.top().(*previewScreen)
	return preview
}

func (h *harness) prompt() *promptDialog {
	prompt, _ := h.model.dialog.(*promptDialog)
	return prompt
}

func (h *harness) confirmation() *confirmDialog {
	confirmation, _ := h.model.dialog.(*confirmDialog)
	return confirmation
}

func (h *harness) start(msg tea.KeyPressMsg) tea.Cmd {
	intent := h.press(msg)
	_, operation := h.model.Update(intent())
	return operation
}

func runes(value string) tea.KeyPressMsg {
	code, _ := utf8.DecodeRuneInString(value)
	return tea.KeyPressMsg{Code: code, Text: value}
}

func key(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
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

		require.NotNil(t, h.confirmation())
		require.Contains(t, h.view(), "zstd")
		require.Empty(t, h.repositories.InitializeCalls())
	})

	t.Run(`Given the initialization confirmation
	When the user rejects it
	Then the console quits without initializing`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, false).started()

		_, cmd := h.model.Update(h.press(runes("n"))())

		require.True(t, isQuit(cmd))
		require.Empty(t, h.repositories.InitializeCalls())
	})

	t.Run(`Given the initialization confirmation
	When the user presses enter
	Then nothing is initialized`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, false).started()

		h.pressAndRun(key(tea.KeyEnter))

		require.NotNil(t, h.confirmation())
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

		require.Nil(t, h.model.dialog)
		require.Len(t, h.model.snapshots.snapshots, 1)
		require.Contains(t, h.view(), snapshotID)
	})

	t.Run(`Given a repository that cannot be opened
	When the console starts
	Then the error is shown`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true)
		h.repositories.InitializedFunc = func(context.Context) (bool, error) { return false, errUnreachable }

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

		require.NotNil(t, h.files())
		require.Equal(t, snapshotID, h.snapshots.ManifestCalls()[0].SnapshotID)
		require.Equal(t, "notes", h.files().entries[0].Name())
		require.Equal(t, "README.txt", h.files().entries[1].Name())
	})

	t.Run(`Given a nested directory of a snapshot
	When the user goes back
	Then the parent is shown with the directory selected, and back again returns to the snapshots`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.pressAndRun(key(tea.KeyEnter))
		h.pressAndRun(key(tea.KeyEnter))
		require.Equal(t, "notes", h.files().directory)

		h.pressAndRun(key(tea.KeyEsc))

		require.Equal(t, "", h.files().directory)
		selected, _ := h.files().selected()
		require.Equal(t, "notes", selected.Name())

		h.pressAndRun(key(tea.KeyEsc))

		require.Equal(t, h.model.snapshots, h.model.top())
	})

	t.Run(`Given a text file of a snapshot
	When the user opens it
	Then its content is previewed from the snapshot`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.pressAndRun(key(tea.KeyEnter))
		h.pressAndRun(runes("j"))

		h.pressAndRun(key(tea.KeyEnter))

		require.NotNil(t, h.preview())
		require.Equal(t, "line 1\nline 2", h.preview().viewport.GetContent())
		require.Equal(t, snapshotID+"/README.txt", h.model.top().location())
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

		require.Nil(t, h.preview())
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

		require.NotNil(t, h.prompt())
		require.True(t, filepath.IsAbs(h.prompt().input.Value()))
		require.Equal(t, snapshotID, filepath.Base(h.prompt().input.Value()))
	})

	t.Run(`Given the destination form with an empty destination
	When the user submits it
	Then an error is shown and nothing is restored`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.pressAndRun(runes("x"))
		h.prompt().input.SetValue("  ")

		h.pressAndRun(key(tea.KeyEnter))

		require.NotEmpty(t, h.prompt().err)
		require.Nil(t, h.confirmation())
	})

	t.Run(`Given the destination form with a non empty directory
	When the user submits it
	Then the confirmation warns that files will be overwritten`, func(t *testing.T) {
		t.Parallel()
		destination := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(destination, "file.txt"), []byte("data"), 0o600))
		h := newHarness(t, true).started()
		h.pressAndRun(runes("x"))
		h.prompt().input.SetValue(destination)

		h.pressAndRun(key(tea.KeyEnter))

		require.NotNil(t, h.confirmation())
		require.Contains(t, h.confirmation().body, destination)
		require.NotEmpty(t, h.confirmation().warning)
	})

	t.Run(`Given the restore confirmation
	When the user presses enter
	Then the restore does not start`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.pressAndRun(runes("x"))
		h.pressAndRun(key(tea.KeyEnter))

		h.pressAndRun(key(tea.KeyEnter))

		require.NotNil(t, h.confirmation())
		require.Empty(t, h.snapshots.RestoreCalls())
	})

	t.Run(`Given the restore confirmation
	When the user confirms with y
	Then the snapshot is restored to the chosen destination`, func(t *testing.T) {
		t.Parallel()
		destination := filepath.Join(t.TempDir(), "restore")
		h := newHarness(t, true).started()
		h.pressAndRun(runes("x"))
		h.prompt().input.SetValue(destination)
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

		require.Nil(t, h.model.dialog)
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
		created := fixtures.Snapshot().WithID("new-snapshot").Build()
		h.snapshots.CreateFunc = func(context.Context, string) (snapshot.Snapshot, error) {
			return created, nil
		}
		h.snapshots.ListFunc = func(context.Context) ([]snapshot.Snapshot, error) {
			return []snapshot.Snapshot{fixtures.Snapshot().WithID(snapshotID).Build(), created}, nil
		}
		h.pressAndRun(runes("c"))
		h.typeText(source)

		h.pressAndRun(key(tea.KeyEnter))

		require.Equal(t, source, h.snapshots.CreateCalls()[0].Source)
		require.Contains(t, h.model.message, "Snapshot new-snapshot created")
		require.Len(t, h.snapshots.ListCalls(), 2)
		selected, _ := h.model.snapshots.selected()
		require.Equal(t, "new-snapshot", selected.ID())
	})

	t.Run(`Given an operation in progress
	When the user asks to restore a snapshot
	Then the action is ignored`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.start(runes("v"))

		h.pressAndRun(runes("x"))

		require.True(t, h.model.loading)
		require.Nil(t, h.model.dialog)
	})

	t.Run(`Given a verification in progress
	When the user presses escape
	Then the operation is cancelled`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		operation := h.start(runes("v"))

		h.press(key(tea.KeyEsc))
		h.run(operation)

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

func TestConsoleDialogsAndCommands(t *testing.T) {
	t.Parallel()

	t.Run(`Given the create form
	When the user types a path with spaces
	Then the spaces are kept`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.pressAndRun(runes("c"))

		h.typeText("my")
		h.pressAndRun(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
		h.typeText("dir")

		require.Equal(t, "my dir", h.prompt().input.Value())
	})

	t.Run(`Given the create form with an empty origin
	When the user submits it
	Then an error is shown and nothing is created`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.pressAndRun(runes("c"))

		h.pressAndRun(key(tea.KeyEnter))

		require.Equal(t, errOriginRequired.Error(), h.prompt().err)
		require.Empty(t, h.snapshots.CreateCalls())
	})

	t.Run(`Given the create form
	When the user cancels it
	Then the form is closed with a message`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.pressAndRun(runes("c"))

		h.pressAndRun(key(tea.KeyEsc))

		require.Nil(t, h.model.dialog)
		require.Equal(t, "Snapshot creation cancelled", h.model.message)
	})

	t.Run(`Given the snapshots list
	When the user enters the quit command
	Then the console quits`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.pressAndRun(runes(":"))
		h.typeText("q")

		cmd := h.press(key(tea.KeyEnter))

		require.True(t, isQuit(cmd))
	})

	t.Run(`Given a nested directory of a snapshot
	When the user enters the snapshots command
	Then the snapshots list is shown`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.pressAndRun(key(tea.KeyEnter))
		h.pressAndRun(key(tea.KeyEnter))
		h.pressAndRun(runes(":"))
		h.typeText("snap")

		h.pressAndRun(key(tea.KeyEnter))

		require.Nil(t, h.model.command)
		require.Equal(t, h.model.snapshots, h.model.top())
	})

	t.Run(`Given the command line
	When the user enters an unknown command
	Then a message reports it`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.pressAndRun(runes(":"))
		h.typeText("nope")

		h.pressAndRun(key(tea.KeyEnter))

		require.Equal(t, "Unknown command: nope", h.model.message)
	})

	t.Run(`Given an empty command line
	When the user presses backspace
	Then the command line is closed`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.pressAndRun(runes(":"))

		h.pressAndRun(key(tea.KeyBackspace))

		require.Nil(t, h.model.command)
	})
}

func TestConsoleLayout(t *testing.T) {
	t.Parallel()

	t.Run(`Given an error and a small terminal
	When the console is rendered
	Then the output fits the terminal`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.model.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
		h.model.err = errors.New("first failure\nsecond failure")

		view := h.view()

		require.Equal(t, 20, lipgloss.Height(view))
		require.Equal(t, 80, lipgloss.Width(view))
		require.Contains(t, view, "second failure")
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

func TestConsoleRepositories(t *testing.T) {
	t.Parallel()

	t.Run(`Given several configured repositories
	When the console starts
	Then the first one is opened and shown in the header`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).withRepository("nas", "smb://user@nas/backups").started()

		require.Equal(t, "default", h.catalog.OpenCalls()[0].Name)
		require.Equal(t, "default", h.model.current.Name())
		require.Contains(t, h.view(), "default")
	})

	t.Run(`Given no configured repositories
	When the console starts
	Then the error is shown`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true)
		h.profiles = nil

		h.started()

		require.ErrorIs(t, h.model.err, app.ErrNoRepositories)
		require.Empty(t, h.catalog.OpenCalls())
	})

	t.Run(`Given the snapshots list
	When the user enters the repositories command
	Then the configured repositories are listed with the current one selected`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).withRepository("nas", "smb://user@nas/backups").started()
		h.pressAndRun(runes(":"))
		h.typeText("repo")

		h.pressAndRun(key(tea.KeyEnter))

		require.NotNil(t, h.repositoriesScreen())
		selected, _ := h.repositoriesScreen().selected()
		require.Equal(t, "default", selected.Name())
		require.Contains(t, h.view(), "smb")
	})

	t.Run(`Given the repositories list
	When the user opens another repository
	Then it becomes the current one and its snapshots are listed`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).withRepository("nas", "smb://user@nas/backups").started()
		h.pressAndRun(runes(":"))
		h.typeText("repo")
		h.pressAndRun(key(tea.KeyEnter))
		h.pressAndRun(runes("j"))

		h.pressAndRun(key(tea.KeyEnter))

		require.Equal(t, "nas", h.catalog.OpenCalls()[1].Name)
		require.Equal(t, "nas", h.model.current.Name())
		require.Equal(t, h.model.snapshots, h.model.top())
		require.Len(t, h.repositories.InitializedCalls(), 2)
		require.Len(t, h.snapshots.ListCalls(), 2)
	})

	t.Run(`Given the repositories list
	When the repository chosen cannot be opened
	Then the error is shown and the current repository is kept`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).withRepository("nas", "broken://nas").started()
		h.pressAndRun(runes(":"))
		h.typeText("repo")
		h.pressAndRun(key(tea.KeyEnter))
		h.pressAndRun(runes("j"))

		h.pressAndRun(key(tea.KeyEnter))

		require.ErrorIs(t, h.model.err, errUnreachable)
		require.Equal(t, "default", h.model.current.Name())
		require.NotNil(t, h.repositoriesScreen())
	})

	t.Run(`Given the repositories list
	When the user goes back
	Then the snapshots of the current repository are shown`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).withRepository("nas", "smb://user@nas/backups").started()
		h.pressAndRun(runes(":"))
		h.typeText("repo")
		h.pressAndRun(key(tea.KeyEnter))

		h.pressAndRun(key(tea.KeyEsc))

		require.Equal(t, h.model.snapshots, h.model.top())
		require.Len(t, h.catalog.OpenCalls(), 1)
	})

	t.Run(`Given several repositories and one that is not initialized
	When the user declines to initialize it
	Then the repositories list is shown instead of quitting`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, false).withRepository("nas", "smb://user@nas/backups").started()

		cmd := h.press(runes("n"))
		require.False(t, isQuit(cmd))
		h.run(cmd)

		require.NotNil(t, h.repositoriesScreen())
		require.Contains(t, h.model.message, "not initialized")
		require.Empty(t, h.repositories.InitializeCalls())
	})
}

func TestFitColumns(t *testing.T) {
	t.Parallel()

	widths := func(columns []table.Column) []int {
		result := make([]int, len(columns))
		for i, column := range columns {
			result[i] = column.Width
		}
		return result
	}

	t.Run(`Given the snapshot columns and a wide table
	When the columns are fitted
	Then the flexible column takes the remaining width`, func(t *testing.T) {
		t.Parallel()

		columns := fitColumns(snapshotColumns, 100)

		require.Equal(t, []int{19, 26, 7, 10, 28}, widths(columns))
	})

	t.Run(`Given the snapshot columns and a narrow table
	When the columns are fitted
	Then the columns that do not fit are hidden from the right`, func(t *testing.T) {
		t.Parallel()

		columns := fitColumns(snapshotColumns, 60)

		require.Equal(t, []int{19, 26, 7, 0, 0}, widths(columns))
	})
}

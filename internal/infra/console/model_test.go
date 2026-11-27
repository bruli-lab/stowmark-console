package console

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bruli-lab/stowmark-console/internal/app"
	"github.com/bruli-lab/stowmark-console/internal/domain/encryption"
	"github.com/bruli-lab/stowmark-console/internal/domain/integrity"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/domain/retention"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
	"github.com/bruli-lab/stowmark-console/internal/fixtures"
	"github.com/bruli-lab/stowmark-console/internal/mocks"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
)

const snapshotID = "20261126T160034Z-f482b24d"

type harness struct {
	model         *Model
	openDirectory string
	opened        []string
	openErr       error
	profiles      []repository.Profile
	catalog       *mocks.CatalogMock
	repositories  *mocks.RepositoryMock
	snapshots     *mocks.SnapshotRepositoryMock
	keys          *mocks.KeysMock
	lockReport    repository.LockReport
	retention     *mocks.RetentionRepositoryMock
	checker       *mocks.CheckerMock
	checkReport   integrity.Report
}

var errUnreachable = errors.New("unreachable")

func newHarness(t *testing.T, initialized bool) *harness {
	t.Helper()
	repositories := &mocks.RepositoryMock{
		StatusFunc: func(context.Context) (repository.Status, error) {
			if initialized {
				return repository.StatusReady, nil
			}
			return repository.StatusUninitialized, nil
		},
		InitializeFunc: func(context.Context, repository.InitOptions) error { return nil },
	}
	h := &harness{}
	repositories.UnlockFunc = func(context.Context, bool) (repository.LockReport, error) { return h.lockReport, nil }
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
		VerifyFunc: func(ctx context.Context, _ *snapshot.Verification) ([]snapshot.Failure, error) {
			return nil, ctx.Err()
		},
		RestoreFunc: func(context.Context, *snapshot.Restoration) ([]snapshot.Failure, error) { return nil, nil },
		ExtractFileFunc: func(_ context.Context, extraction *snapshot.FileExtraction) (string, error) {
			return filepath.Join(extraction.Directory(), filepath.Base(extraction.Path())), nil
		},
	}
	h.profiles = []repository.Profile{
		repository.NewProfile("default", filepath.Join(t.TempDir(), "repository"), repository.NewInitOptions("zstd", nil, nil)),
	}
	h.repositories = repositories
	h.snapshots = snapshots
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
	h.keys = &mocks.KeysMock{
		GenerateFunc: func(_ context.Context, generation *encryption.KeyGeneration) (encryption.KeyPair, error) {
			return encryption.NewKeyPair(
				filepath.Join(generation.Folder(), "1234-private.pem"),
				filepath.Join(generation.Folder(), "1234-public.pem"),
			)
		},
		RewrapFunc: func(context.Context, *encryption.Rewrap) error { return nil },
		RotateFunc: func(context.Context, *encryption.Rotation) error { return nil },
	}
	h.retention = &mocks.RetentionRepositoryMock{
		PlanForgetFunc: func(context.Context, retention.Policy) (retention.Plan, error) { return nil, nil },
		ForgetFunc: func(context.Context, *retention.Forgetting) (retention.ForgetResult, error) {
			return retention.ForgetResult{}, nil
		},
		PlanPruneFunc: func(context.Context) (retention.PrunePlan, error) { return retention.PrunePlan{}, nil },
		PruneFunc: func(context.Context, *retention.Pruning) (retention.PruneResult, error) {
			return retention.PruneResult{}, nil
		},
	}
	h.checker = &mocks.CheckerMock{
		CheckFunc: func(context.Context, integrity.Scope) (integrity.Report, error) { return h.checkReport, nil },
	}
	buses := app.NewBuses(&app.Ports{
		Catalog:      h.catalog,
		Repositories: repositories,
		Snapshots:    snapshots,
		Keys:         h.keys,
		Retention:    h.retention,
		Checker:      h.checker,
	})
	h.openDirectory = t.TempDir()
	model := New(t.Context(), buses, WithFileOpener(h.openDirectory, func(path string) error {
		h.opened = append(h.opened, path)
		return h.openErr
	}))
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

func (h *harness) clearInput() {
	h.pressAndRun(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
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
		h.repositories.StatusFunc = func(context.Context) (repository.Status, error) {
			return repository.StatusUninitialized, errUnreachable
		}

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
	Then it is previewed in hexadecimal`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.snapshots.FileContentFunc = func(context.Context, string, string, int64) ([]byte, error) {
			return []byte{0x89, 'P', 'N', 'G', 0x00, 0x01}, nil
		}
		h.pressAndRun(key(tea.KeyEnter))
		h.pressAndRun(runes("j"))

		h.pressAndRun(key(tea.KeyEnter))

		require.NoError(t, h.model.err)
		require.Equal(t, binaryPreview, h.preview().content.kind)
		require.Contains(t, h.view(), "89 50 4e 47 00 01")
		require.Contains(t, h.view(), "hexadecimal view")
	})

	t.Run(`Given an image file of a snapshot
	When the user opens it
	Then it is drawn with its format and size`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.snapshots.FileContentFunc = func(context.Context, string, string, int64) ([]byte, error) {
			return encodedImage(t, 40, 20), nil
		}
		h.pressAndRun(key(tea.KeyEnter))
		h.pressAndRun(runes("j"))

		h.pressAndRun(key(tea.KeyEnter))

		require.Equal(t, imagePreview, h.preview().content.kind)
		view := h.view()
		require.Contains(t, view, "PNG image, 40×20")
		require.Contains(t, view, "▀")
	})
}

func TestConsoleOpenFile(t *testing.T) {
	t.Parallel()

	t.Run(`Given a file of a snapshot
	When the user opens it with the system application
	Then it is extracted to the open directory and handed to the opener`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.pressAndRun(key(tea.KeyEnter))
		h.pressAndRun(runes("j"))

		h.pressAndRun(runes("o"))

		call := h.snapshots.ExtractFileCalls()[0]
		require.Equal(t, "/home/user/documents/README.txt", call.Extraction.Path())
		require.Equal(t, h.openDirectory, call.Extraction.Directory())
		require.Equal(t, []string{filepath.Join(h.openDirectory, "README.txt")}, h.opened)
		require.Contains(t, h.model.message, "Opened README.txt")
	})

	t.Run(`Given a previewed file
	When the user opens it with the system application
	Then the previewed file is opened`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.pressAndRun(key(tea.KeyEnter))
		h.pressAndRun(runes("j"))
		h.pressAndRun(key(tea.KeyEnter))

		h.pressAndRun(runes("o"))

		require.Len(t, h.opened, 1)
		require.NotNil(t, h.preview())
	})

	t.Run(`Given a directory of a snapshot
	When the user tries to open it with the system application
	Then nothing is extracted`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.pressAndRun(key(tea.KeyEnter))

		h.pressAndRun(runes("o"))

		require.Empty(t, h.snapshots.ExtractFileCalls())
	})

	t.Run(`Given an opener that fails
	When the user opens a file with the system application
	Then the error is shown`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.openErr = errUnreachable
		h.pressAndRun(key(tea.KeyEnter))
		h.pressAndRun(runes("j"))

		h.pressAndRun(runes("o"))

		require.ErrorIs(t, h.model.err, errUnreachable)
	})
}

func encodedImage(t *testing.T, width, height int) []byte {
	t.Helper()
	picture := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := range width {
		for y := range height {
			picture.Set(x, y, color.RGBA{R: uint8(x * 6), G: uint8(y * 12), B: 128, A: 255})
		}
	}

	var buffer bytes.Buffer
	require.NoError(t, png.Encode(&buffer, picture))

	return buffer.Bytes()
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
		require.Equal(t, destination, h.snapshots.RestoreCalls()[0].Restoration.Destination())
		require.Contains(t, h.model.message, "restored to "+destination)
	})

	t.Run(`Given a restore that fails for some files
	When it finishes
	Then the failures are shown as an error`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.snapshots.RestoreFunc = func(context.Context, *snapshot.Restoration) ([]snapshot.Failure, error) {
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
		h.snapshots.CreateFunc = func(context.Context, *snapshot.Creation) (snapshot.Snapshot, error) {
			return created, nil
		}
		h.snapshots.ListFunc = func(context.Context) ([]snapshot.Snapshot, error) {
			return []snapshot.Snapshot{fixtures.Snapshot().WithID(snapshotID).Build(), created}, nil
		}
		h.pressAndRun(runes("c"))
		h.clearInput()
		h.typeText(source)

		h.pressAndRun(key(tea.KeyEnter))

		require.Equal(t, source, h.snapshots.CreateCalls()[0].Creation.Source())
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

		require.Equal(t, snapshotID, h.snapshots.VerifyCalls()[0].Verification.SnapshotID())
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
		h.clearInput()

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
		h.clearInput()

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

	t.Run(`Given an unencrypted repository
	When the console starts
	Then the top panel lists the general commands but not the key rotation ones`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()

		view := h.view()

		require.Contains(t, view, "Commands")
		require.Contains(t, view, ":repo")
		require.Contains(t, view, ":keygen")
		require.NotContains(t, view, ":rewrap")
		require.NotContains(t, view, ":rekey")
	})

	t.Run(`Given the command line
	When the user types the beginning of a command
	Then the footer shows the matching commands`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.pressAndRun(runes(":"))

		h.typeText("re")

		footer := h.model.renderFooter()
		require.Contains(t, footer, ":repo")
		require.Contains(t, footer, ":rewrap")
		require.Contains(t, footer, ":rekey")
		require.NotContains(t, footer, ":keygen")
	})

	t.Run(`Given the command line with the beginning of a command
	When the user presses tab
	Then the command is completed`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.pressAndRun(runes(":"))
		h.typeText("keyg")

		h.pressAndRun(key(tea.KeyTab))

		require.Equal(t, "keygen", h.model.command.input.Value())
	})

	t.Run(`Given the command line
	When the user types something no command starts with
	Then the footer says that no command matches`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.pressAndRun(runes(":"))

		h.typeText("zz")

		require.Contains(t, h.model.renderFooter(), "no matching command")
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

func TestConsoleCreateForm(t *testing.T) {
	t.Parallel()

	newSource := func(t *testing.T) string {
		t.Helper()
		source := t.TempDir()
		for _, directory := range []string{"alpha", "beta", ".hidden"} {
			require.NoError(t, os.Mkdir(filepath.Join(source, directory), 0o700))
		}
		require.NoError(t, os.WriteFile(filepath.Join(source, "file.txt"), []byte("data"), 0o600))
		return source
	}

	t.Run(`Given a selected snapshot
	When the user opens the create form
	Then the origin is prefilled with the source of the snapshot`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()

		h.pressAndRun(runes("c"))

		require.Equal(t, "/home/user/documents", h.prompt().input.Value())
	})

	t.Run(`Given the create form with the prefix of a directory
	When the user presses tab
	Then the directory is completed and its status is shown`, func(t *testing.T) {
		t.Parallel()
		source := newSource(t)
		h := newHarness(t, true).started()
		h.pressAndRun(runes("c"))
		h.clearInput()
		h.typeText(source + "/al")

		h.pressAndRun(key(tea.KeyTab))

		require.Equal(t, source+"/alpha/", h.prompt().input.Value())
		require.Contains(t, h.view(), "Directory with 0 entries")
	})

	t.Run(`Given the create form with a directory that does not exist
	When the user submits it
	Then an error is shown and nothing is created`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.pressAndRun(runes("c"))
		h.clearInput()
		h.typeText(filepath.Join(t.TempDir(), "missing"))

		h.pressAndRun(key(tea.KeyEnter))

		require.True(t, h.prompt().invalid)
		require.Equal(t, errDirectoryNotFound.Error(), h.prompt().err)
		require.Empty(t, h.snapshots.CreateCalls())
	})

	t.Run(`Given the create form with a file instead of a directory
	When the value is checked
	Then it is reported as not a directory`, func(t *testing.T) {
		t.Parallel()
		source := newSource(t)
		h := newHarness(t, true).started()
		h.pressAndRun(runes("c"))
		h.clearInput()

		h.typeText(filepath.Join(source, "file.txt"))

		require.Equal(t, errNotADirectory.Error(), h.prompt().status)
	})

	t.Run(`Given a directory with subdirectories, hidden ones and files
	When the suggestions for it are computed
	Then the visible subdirectories and the known origins are suggested`, func(t *testing.T) {
		t.Parallel()
		source := newSource(t)

		suggestions := pathSuggestions(source+"/", []string{"/known"})

		require.ElementsMatch(t, []string{"/known", source + "/alpha/", source + "/beta/"}, suggestions)
	})

	t.Run(`Given a prefix that starts with a dot
	When the suggestions are computed
	Then hidden directories are suggested too`, func(t *testing.T) {
		t.Parallel()
		source := newSource(t)

		suggestions := pathSuggestions(source+"/.h", nil)

		require.Contains(t, suggestions, source+"/.hidden/")
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

	t.Run(`Given a tall and wide terminal
	When the console is rendered
	Then the top panel shows the logo at its full size and the output fits the terminal`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.model.Update(tea.WindowSizeMsg{Width: 160, Height: 50})

		view := h.view()

		require.Contains(t, view, stowmarkLogo()[:40])
		require.Equal(t, 50, lipgloss.Height(view))
		require.Equal(t, 160, lipgloss.Width(view))
	})

	t.Run(`Given a terminal shorter than the logo needs
	When the console is rendered
	Then the logo is hidden and the output fits the terminal`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.model.Update(tea.WindowSizeMsg{Width: 160, Height: 40})

		view := h.view()

		require.NotContains(t, view, "▀")
		require.Equal(t, 40, lipgloss.Height(view))
	})

	t.Run(`Given a tall terminal too narrow for the logo and every panel column
	When the console is rendered
	Then the logo is hidden and the output fits the terminal`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.model.Update(tea.WindowSizeMsg{Width: 80, Height: 50})

		view := h.view()

		require.NotContains(t, view, "▀")
		require.Equal(t, 80, lipgloss.Width(view))
		require.Equal(t, 50, lipgloss.Height(view))
	})
}

func TestLogoLines(t *testing.T) {
	t.Parallel()

	t.Run(`Given the embedded logo
	When it is drawn
	Then it uses two pixels per cell`, func(t *testing.T) {
		t.Parallel()

		lines := strings.Split(stowmarkLogo(), "\n")

		require.Len(t, lines, 12)
		require.Equal(t, 18, ansi.StringWidth(lines[0]))
	})

	t.Run(`Given an image with a transparent and an opaque pixel
	When it is drawn over a background
	Then the transparent pixel takes the background color`, func(t *testing.T) {
		t.Parallel()
		picture := image.NewNRGBA(image.Rect(0, 0, 1, 2))
		picture.Set(0, 0, color.NRGBA{})
		picture.Set(0, 1, color.NRGBA{R: 0, G: 230, B: 240, A: 255})

		lines := logoLines(picture, color.RGBA{R: 24, G: 24, B: 37, A: 255})

		require.Equal(t, []string{"\x1b[38;2;24;24;37m\x1b[48;2;0;230;240m▀\x1b[0m"}, lines)
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

	const nasLocation = "smb://user@nas/backups"

	t.Run(`Given a single configured repository
	When the console starts
	Then it is opened directly and shown in the header`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()

		require.Equal(t, "default", h.catalog.OpenCalls()[0].Name)
		require.Equal(t, h.model.snapshots, h.model.top())
		require.Contains(t, h.view(), "default")
	})

	t.Run(`Given several configured repositories
	When the console starts
	Then the repositories are listed without opening any`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).withRepository("nas", nasLocation).started()

		require.NotNil(t, h.repositoriesScreen())
		require.Empty(t, h.catalog.OpenCalls())
		require.Contains(t, h.view(), "Choose the repository to open")
		require.Contains(t, h.view(), nasLocation)
	})

	t.Run(`Given the repositories list at startup
	When the user chooses one
	Then it is opened and its snapshots are listed`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).withRepository("nas", nasLocation).started()
		h.pressAndRun(runes("j"))

		h.pressAndRun(key(tea.KeyEnter))

		require.Equal(t, "nas", h.catalog.OpenCalls()[0].Name)
		require.Equal(t, "nas", h.model.current.Name())
		require.Equal(t, h.model.snapshots, h.model.top())
		require.Len(t, h.snapshots.ListCalls(), 1)
	})

	t.Run(`Given the repositories list at startup
	When the user tries to go back or to the snapshots
	Then the list stays until a repository is chosen`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).withRepository("nas", nasLocation).started()

		h.pressAndRun(key(tea.KeyEsc))
		h.pressAndRun(runes(":"))
		h.typeText("snap")
		h.pressAndRun(key(tea.KeyEnter))

		require.NotNil(t, h.repositoriesScreen())
		require.Equal(t, "Choose a repository first", h.model.message)
		require.Empty(t, h.snapshots.ListCalls())
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

	t.Run(`Given an open repository
	When the user enters the repositories command
	Then the repositories are listed with the current one selected`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).withRepository("nas", nasLocation).started()
		h.pressAndRun(key(tea.KeyEnter))
		h.pressAndRun(runes(":"))
		h.typeText("repo")

		h.pressAndRun(key(tea.KeyEnter))

		require.NotNil(t, h.repositoriesScreen())
		selected, _ := h.repositoriesScreen().selected()
		require.Equal(t, "default", selected.Name())
		require.Contains(t, h.view(), currentMarker+"default")
	})

	t.Run(`Given the repositories list of an open repository
	When the user goes back
	Then the snapshots of the current repository are shown`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).withRepository("nas", nasLocation).started()
		h.pressAndRun(key(tea.KeyEnter))
		h.pressAndRun(runes(":"))
		h.typeText("repo")
		h.pressAndRun(key(tea.KeyEnter))

		h.pressAndRun(key(tea.KeyEsc))

		require.Equal(t, h.model.snapshots, h.model.top())
		require.Len(t, h.catalog.OpenCalls(), 1)
	})

	t.Run(`Given an open repository
	When the repository chosen cannot be opened
	Then the error is shown and the current repository is kept`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).withRepository("nas", "broken://nas").started()
		h.pressAndRun(key(tea.KeyEnter))
		h.pressAndRun(runes(":"))
		h.typeText("repo")
		h.pressAndRun(key(tea.KeyEnter))
		h.pressAndRun(runes("j"))

		h.pressAndRun(key(tea.KeyEnter))

		require.ErrorIs(t, h.model.err, errUnreachable)
		require.Equal(t, "default", h.model.current.Name())
		require.NotNil(t, h.repositoriesScreen())
	})

	t.Run(`Given several repositories and one that is not initialized
	When the user declines to initialize it
	Then the repositories list is shown instead of quitting`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, false).withRepository("nas", nasLocation).started()
		h.pressAndRun(key(tea.KeyEnter))

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

func TestKnownOrigins(t *testing.T) {
	t.Parallel()

	t.Run(`Given snapshots that share a source
	When the known origins are computed
	Then each source appears once as a directory`, func(t *testing.T) {
		t.Parallel()
		snapshots := []snapshot.Snapshot{
			fixtures.Snapshot().WithSource("/data").Build(),
			fixtures.Snapshot().WithSource("/data/").Build(),
			fixtures.Snapshot().WithSource("/home").Build(),
		}

		require.Equal(t, []string{"/data/", "/home/"}, knownOrigins(snapshots))
	})
}

func TestHexLines(t *testing.T) {
	t.Parallel()

	t.Run(`Given content longer than a line
	When it is shown in hexadecimal
	Then each line has the offset, the bytes and the printable characters`, func(t *testing.T) {
		t.Parallel()

		lines := hexLines([]byte("ABCDEFGHIJKLMNOPQ\x00"))

		require.Len(t, lines, 2)
		require.Contains(t, ansi.Strip(lines[0]), "00000000  41 42 43 44 45 46 47 48  49 4a 4b 4c 4d 4e 4f 50  |ABCDEFGHIJKLMNOP|")
		require.Contains(t, ansi.Strip(lines[1]), "00000010  51 00")
		require.True(t, strings.HasSuffix(ansi.Strip(lines[1]), "|Q.|"))
	})
}

func TestImageLines(t *testing.T) {
	t.Parallel()

	t.Run(`Given a wide image and a panel
	When it is drawn
	Then it keeps its aspect ratio inside the panel using two pixels per cell`, func(t *testing.T) {
		t.Parallel()
		picture, _, err := image.Decode(bytes.NewReader(encodedImage(t, 200, 100)))
		require.NoError(t, err)

		lines := imageLines(picture, 50, 40)

		require.Len(t, lines, 13)
		require.Equal(t, 50, ansi.StringWidth(lines[0]))
	})

	t.Run(`Given a large image
	When it is classified
	Then a thumbnail is kept with the original size`, func(t *testing.T) {
		t.Parallel()

		content := classifyContent(encodedImage(t, 1024, 256))

		require.Equal(t, imagePreview, content.kind)
		require.Equal(t, 1024, content.width)
		require.Equal(t, maxImagePixels, content.image.Bounds().Dx())
	})
}

package console

import (
	"context"

	"github.com/bruli-lab/stowmark-console/internal/app"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
	tea "github.com/charmbracelet/bubbletea"
)

const maxPreviewSize = 1 << 20

func (m *Model) startOperation(message string, run func(ctx context.Context) tea.Msg) tea.Cmd {
	ctx, cancel := context.WithCancel(m.ctx)
	m.cancelOperation = cancel
	m.loading = true
	m.err = nil
	m.message = message

	return func() tea.Msg {
		defer cancel()
		return run(ctx)
	}
}

func (m *Model) finishOperation() {
	m.loading = false
	if m.cancelOperation != nil {
		m.cancelOperation()
		m.cancelOperation = nil
	}
}

func (m *Model) checkRepository() tea.Cmd {
	handler := m.handlers.RepositoryStatus
	return m.startOperation("Opening repository...", func(ctx context.Context) tea.Msg {
		initialized, err := handler.Handle(ctx, app.RepositoryStatusQuery{})
		return repositoryStatusMsg{initialized: initialized, err: err}
	})
}

func (m *Model) initRepository() tea.Cmd {
	handler := m.handlers.InitRepository
	options := m.settings.Init
	return m.startOperation("Initializing repository...", func(ctx context.Context) tea.Msg {
		event, err := handler.Handle(ctx, app.InitRepositoryCommand{Options: options})
		return repositoryInitializedMsg{event: event, err: err}
	})
}

func (m *Model) loadSnapshots(selectID string) tea.Cmd {
	handler := m.handlers.ListSnapshots
	return m.startOperation("Loading snapshots...", func(ctx context.Context) tea.Msg {
		snapshots, err := handler.Handle(ctx, app.ListSnapshotsQuery{})
		return snapshotsLoadedMsg{snapshots: snapshots, selectID: selectID, err: err}
	})
}

func (m *Model) openSnapshot(snapshotID string) tea.Cmd {
	handler := m.handlers.GetManifest
	return m.startOperation("Loading snapshot "+snapshotID+"...", func(ctx context.Context) tea.Msg {
		manifest, err := handler.Handle(ctx, app.GetManifestQuery{SnapshotID: snapshotID})
		return manifestLoadedMsg{manifest: manifest, err: err}
	})
}

func (m *Model) loadFileContent(snapshotID, name string, file snapshot.File) tea.Cmd {
	handler := m.handlers.GetFileContent
	return m.startOperation("Reading "+name+"...", func(ctx context.Context) tea.Msg {
		content, err := handler.Handle(ctx, app.GetFileContentQuery{
			SnapshotID: snapshotID,
			Path:       file.Path(),
			MaxSize:    maxPreviewSize,
		})
		return fileContentLoadedMsg{
			name:      name,
			content:   content,
			truncated: file.Size() > maxPreviewSize,
			err:       err,
		}
	})
}

func (m *Model) createSnapshot(source string) tea.Cmd {
	handler := m.handlers.CreateSnapshot
	return m.startOperation("Creating snapshot of "+source+"...", func(ctx context.Context) tea.Msg {
		event, err := handler.Handle(ctx, app.CreateSnapshotCommand{Source: source})
		return snapshotCreatedMsg{event: event, err: err}
	})
}

func (m *Model) verifySnapshot(snapshotID string) tea.Cmd {
	handler := m.handlers.VerifySnapshot
	return m.startOperation("Verifying snapshot "+snapshotID+"...", func(ctx context.Context) tea.Msg {
		event, err := handler.Handle(ctx, app.VerifySnapshotCommand{SnapshotID: snapshotID})
		return snapshotVerifiedMsg{snapshotID: snapshotID, event: event, err: err}
	})
}

func (m *Model) restoreSnapshot(request restoreRequest) tea.Cmd {
	handler := m.handlers.RestoreSnapshot
	return m.startOperation("Restoring snapshot "+request.snapshotID+"...", func(ctx context.Context) tea.Msg {
		event, err := handler.Handle(ctx, app.RestoreSnapshotCommand{
			SnapshotID:  request.snapshotID,
			Destination: request.destination,
		})
		return snapshotRestoredMsg{snapshotID: request.snapshotID, event: event, err: err}
	})
}

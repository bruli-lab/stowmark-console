package console

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/bruli-lab/stowmark-console/internal/app"
)

const maxPreviewSize = 1 << 20

func (m *Model) startOperation(message string, run func(ctx context.Context) tea.Msg) tea.Cmd {
	ctx, cancel := context.WithCancel(m.ctx)
	m.cancelOperation = cancel
	m.loading = true
	m.err = nil
	m.progress = message

	return func() tea.Msg {
		defer cancel()
		return run(ctx)
	}
}

func (m *Model) finishOperation() {
	m.loading = false
	m.progress = ""
	if m.cancelOperation != nil {
		m.cancelOperation()
		m.cancelOperation = nil
	}
}

func (m *Model) openInitialRepository() tea.Cmd {
	list := m.handlers.ListRepositories
	open := m.handlers.OpenRepository
	return m.startOperation("Opening repository...", func(ctx context.Context) tea.Msg {
		profiles, err := list.Handle(ctx, app.ListRepositoriesQuery{})
		if err != nil {
			return repositoryOpenedMsg{err: err}
		}
		name := profiles[0].Name()
		event, err := open.Handle(ctx, app.OpenRepositoryCommand{Name: name})
		return repositoryOpenedMsg{profiles: profiles, name: name, event: event, err: err}
	})
}

func (m *Model) openRepository(name string) tea.Cmd {
	handler := m.handlers.OpenRepository
	return m.startOperation("Opening repository "+name+"...", func(ctx context.Context) tea.Msg {
		event, err := handler.Handle(ctx, app.OpenRepositoryCommand{Name: name})
		return repositoryOpenedMsg{name: name, event: event, err: err}
	})
}

func (m *Model) checkRepository() tea.Cmd {
	handler := m.handlers.RepositoryStatus
	return m.startOperation("Checking repository...", func(ctx context.Context) tea.Msg {
		initialized, err := handler.Handle(ctx, app.RepositoryStatusQuery{})
		return repositoryStatusMsg{initialized: initialized, err: err}
	})
}

func (m *Model) initRepository() tea.Cmd {
	handler := m.handlers.InitRepository
	options := m.current.Init()
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

func (m *Model) loadFileContent(request filePreviewRequested) tea.Cmd {
	handler := m.handlers.GetFileContent
	return m.startOperation("Reading "+request.name+"...", func(ctx context.Context) tea.Msg {
		content, err := handler.Handle(ctx, app.GetFileContentQuery{
			SnapshotID: request.snapshotID,
			Path:       request.file.Path(),
			MaxSize:    maxPreviewSize,
		})
		return fileContentLoadedMsg{
			request:   request,
			content:   content,
			truncated: request.file.Size() > maxPreviewSize,
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

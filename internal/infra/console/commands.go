package console

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/bruli-lab/stowmark-console/internal/app"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

const maxPreviewSize = 1 << 20

func (m *Model) startOperation(message string, run func(ctx context.Context) tea.Msg) tea.Cmd {
	ctx, cancel := context.WithCancel(m.ctx)
	m.cancelOperation = cancel
	m.loading = true
	m.err = nil
	m.activity = message
	m.cancelling = false
	m.transfer = nil

	return func() tea.Msg {
		defer cancel()
		return run(ctx)
	}
}

func (m *Model) finishOperation() {
	m.loading = false
	m.activity = ""
	m.cancelling = false
	m.transfer = nil
	if m.cancelOperation != nil {
		m.cancelOperation()
		m.cancelOperation = nil
	}
}

func (m *Model) loadRepositories() tea.Cmd {
	bus := m.buses.Queries
	return m.startOperation("Loading repositories...", func(ctx context.Context) tea.Msg {
		profiles, err := ask[[]repository.Profile](ctx, bus, app.ListRepositoriesQuery{})
		return repositoriesLoadedMsg{profiles: profiles, err: err}
	})
}

func (m *Model) openRepository(name string) tea.Cmd {
	bus := m.buses.Commands
	return m.startOperation("Opening repository "+name+"...", func(ctx context.Context) tea.Msg {
		event, err := dispatch[repository.Opened](ctx, bus, app.OpenRepositoryCommand{RepositoryName: name})
		return repositoryOpenedMsg{name: name, event: event, err: err}
	})
}

func (m *Model) checkRepository() tea.Cmd {
	bus := m.buses.Queries
	return m.startOperation("Checking repository...", func(ctx context.Context) tea.Msg {
		status, err := ask[repository.Status](ctx, bus, app.RepositoryStatusQuery{})
		return repositoryStatusMsg{status: status, err: err}
	})
}

func (m *Model) initRepository(options repository.InitOptions) tea.Cmd {
	bus := m.buses.Commands
	return m.startOperation("Initializing repository...", func(ctx context.Context) tea.Msg {
		event, err := dispatch[repository.Initialized](ctx, bus, app.InitRepositoryCommand{Options: options})
		return repositoryInitializedMsg{event: event, err: err}
	})
}

func (m *Model) loadSnapshots(selectID string) tea.Cmd {
	bus := m.buses.Queries
	return m.startOperation("Loading snapshots...", func(ctx context.Context) tea.Msg {
		snapshots, err := ask[[]snapshot.Snapshot](ctx, bus, app.ListSnapshotsQuery{})
		return snapshotsLoadedMsg{snapshots: snapshots, selectID: selectID, err: err}
	})
}

func (m *Model) openSnapshot(snapshotID string) tea.Cmd {
	bus := m.buses.Queries
	return m.startOperation("Loading snapshot "+snapshotID+"...", func(ctx context.Context) tea.Msg {
		manifest, err := ask[snapshot.Manifest](ctx, bus, app.GetManifestQuery{SnapshotID: snapshotID})
		return manifestLoadedMsg{manifest: manifest, err: err}
	})
}

func (m *Model) loadFileContent(request filePreviewRequested) tea.Cmd {
	bus := m.buses.Queries
	return m.startOperation("Reading "+request.name+"...", func(ctx context.Context) tea.Msg {
		limit := previewLimit(request.name)
		content, err := ask[[]byte](ctx, bus, app.GetFileContentQuery{
			SnapshotID: request.snapshotID,
			Path:       request.file.Path(),
			MaxSize:    limit,
		})
		if err != nil {
			return fileContentLoadedMsg{request: request, err: err}
		}
		return fileContentLoadedMsg{
			request:   request,
			content:   classifyContent(content),
			truncated: request.file.Size() > limit,
		}
	})
}

func (m *Model) openFile(request filePreviewRequested) tea.Cmd {
	if m.opener == nil {
		m.message = "Opening files is not available"
		return nil
	}

	bus := m.buses.Commands
	directory := m.openDirectory
	opener := m.opener
	return m.startOperation("Extracting "+request.name+"...", func(ctx context.Context) tea.Msg {
		event, err := dispatch[snapshot.FileExtracted](ctx, bus, app.ExtractFileCommand{
			SnapshotID: request.snapshotID,
			Path:       request.file.Path(),
			Directory:  directory,
		})
		if err == nil {
			err = opener(event.Location)
		}
		return fileOpenedMsg{name: request.name, err: err}
	})
}

func (m *Model) createSnapshot(source string, exclusions snapshot.Exclusions) tea.Cmd {
	bus := m.buses.Commands
	return m.startOperation("Creating snapshot of "+source+"...", func(ctx context.Context) tea.Msg {
		command := app.CreateSnapshotCommand{Source: source, Exclusions: exclusions}
		event, err := dispatch[snapshot.Created](ctx, bus, command)
		return snapshotCreatedMsg{event: event, err: err}
	})
}

func (m *Model) verifySnapshot(snapshotID string) tea.Cmd {
	bus := m.buses.Commands
	return m.startOperation("Verifying snapshot "+snapshotID+"...", func(ctx context.Context) tea.Msg {
		event, err := dispatch[snapshot.Verified](ctx, bus, app.VerifySnapshotCommand{SnapshotID: snapshotID})
		return snapshotVerifiedMsg{snapshotID: snapshotID, event: event, err: err}
	})
}

func (m *Model) restoreSnapshot(request restoreRequest) tea.Cmd {
	bus := m.buses.Commands
	return m.startOperation("Restoring snapshot "+request.snapshotID+"...", func(ctx context.Context) tea.Msg {
		event, err := dispatch[snapshot.Restored](ctx, bus, app.RestoreSnapshotCommand{
			SnapshotID:  request.snapshotID,
			Destination: request.destination,
		})
		return snapshotRestoredMsg{snapshotID: request.snapshotID, event: event, err: err}
	})
}

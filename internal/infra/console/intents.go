package console

import (
	tea "charm.land/bubbletea/v2"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

type repositoryOpenRequested struct {
	name string
}

type repositoriesScreenRequested struct{}

type repositoryInitDeclined struct{}

type snapshotOpenRequested struct {
	snapshotID string
}

type snapshotsRefreshRequested struct {
	selectID string
}

type filePreviewRequested struct {
	snapshotID string
	location   string
	name       string
	file       snapshot.File
}

type fileOpenRequested struct {
	request filePreviewRequested
}

type screenCloseRequested struct{}

type snapshotsScreenRequested struct{}

type snapshotCreateRequested struct{}

type snapshotSourceChosen struct {
	source string
}

type snapshotVerifyRequested struct {
	snapshotID string
}

type snapshotRestoreRequested struct {
	snapshotID string
}

type restoreDestinationChosen struct {
	request restoreRequest
}

type restoreConfirmed struct {
	request restoreRequest
}

type repositoryInitConfirmed struct{}

type statusMessage struct {
	text string
}

type restoreRequest struct {
	snapshotID  string
	destination string
}

func emit(msg tea.Msg) tea.Cmd {
	return func() tea.Msg { return msg }
}

func (m *Model) handleIntent(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case screenCloseRequested:
		m.pop()
	case snapshotsScreenRequested:
		if !m.connected() {
			m.message = "Choose a repository first"
			return nil, true
		}
		m.screens = nil
	case repositoriesScreenRequested:
		m.showRepositories()
	case repositoryInitDeclined:
		if len(m.repositories) < 2 {
			return tea.Quit, true
		}
		m.message = "Repository " + m.current.Name() + " was not initialized"
		m.showRepositories()
	case statusMessage:
		m.err = nil
		m.message = msg.text
	case snapshotCreateRequested:
		m.openCreateDialog()
	case snapshotRestoreRequested:
		m.openRestoreDialog(msg.snapshotID)
	case restoreDestinationChosen:
		m.openRestoreConfirmation(msg.request)
	default:
		if cmd, handled := m.handleKeyIntent(msg); handled {
			return cmd, true
		}
		if cmd, handled := m.handleLockIntent(msg); handled {
			return cmd, true
		}
		return m.handleOperationIntent(msg)
	}

	return nil, true
}

func (m *Model) handleOperationIntent(msg tea.Msg) (tea.Cmd, bool) {
	var start func() tea.Cmd

	switch msg := msg.(type) {
	case repositoryInitConfirmed:
		start = m.initRepository
	case repositoryOpenRequested:
		start = func() tea.Cmd { return m.openRepository(msg.name) }
	case snapshotsRefreshRequested:
		start = func() tea.Cmd { return m.loadSnapshots(msg.selectID) }
	case snapshotOpenRequested:
		start = func() tea.Cmd { return m.openSnapshot(msg.snapshotID) }
	case filePreviewRequested:
		start = func() tea.Cmd { return m.loadFileContent(msg) }
	case fileOpenRequested:
		start = func() tea.Cmd { return m.openFile(msg.request) }
	case snapshotSourceChosen:
		start = func() tea.Cmd { return m.createSnapshot(msg.source) }
	case snapshotVerifyRequested:
		start = func() tea.Cmd { return m.verifySnapshot(msg.snapshotID) }
	case restoreConfirmed:
		start = func() tea.Cmd { return m.restoreSnapshot(msg.request) }
	case keysFolderChosen:
		start = func() tea.Cmd { return m.generateKeys(msg.folder) }
	case rewrapConfirmed:
		start = func() tea.Cmd { return m.rewrapKey(msg.request) }
	case rotationConfirmed:
		start = func() tea.Cmd { return m.rotateKey(msg.request) }
	case unlockConfirmed:
		start = func() tea.Cmd { return m.unlockRepository(msg.all) }
	default:
		return nil, false
	}

	if m.loading {
		return nil, true
	}

	return start(), true
}

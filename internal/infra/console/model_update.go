package console

import (
	"context"
	"errors"
	"fmt"

	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		return m.handleKey(msg)

	case repositoryStatusMsg:
		m.finishOperation()
		if m.fail("open repository", msg.err) {
			return m, nil
		}
		if !msg.initialized {
			m.message = ""
			m.confirmingInit = true
			m.initWarning = initRepositoryWarning(m.settings.RepositoryPath)
			return m, nil
		}
		cmd := m.loadSnapshots("")
		return m, cmd

	case repositoryInitializedMsg:
		m.finishOperation()
		if m.fail("initialize repository", msg.err) {
			return m, nil
		}
		cmd := m.reloadWithMessage("Repository initialized successfully", "")
		return m, cmd

	case snapshotsLoadedMsg:
		m.finishOperation()
		m.consumeMessage()
		if m.fail("load snapshots", msg.err) {
			return m, nil
		}
		m.snapshots = msg.snapshots
		m.selectSnapshot(msg.selectID)

	case manifestLoadedMsg:
		m.finishOperation()
		m.consumeMessage()
		if m.fail("load snapshot", msg.err) {
			return m, nil
		}
		m.manifest = msg.manifest
		m.view = filesView
		m.showDirectory("", "")

	case fileContentLoadedMsg:
		m.finishOperation()
		m.message = ""
		if m.fail("read "+msg.name, msg.err) {
			return m, nil
		}
		if !isText(msg.content) {
			m.err = fmt.Errorf("%q is not a text file", msg.name)
			return m, nil
		}
		m.openPreview(msg.name, msg.content, msg.truncated)

	case snapshotCreatedMsg:
		m.finishOperation()
		if m.fail("create snapshot", msg.err) {
			return m, nil
		}
		created := msg.event.Snapshot
		m.view = snapshotsView
		message := fmt.Sprintf(
			"Snapshot %s created: %d files, %s",
			created.ID(),
			created.FileCount(),
			formatSize(created.TotalSize()),
		)
		cmd := m.reloadWithMessage(message, created.ID())
		return m, cmd

	case snapshotVerifiedMsg:
		m.finishOperation()
		if m.fail("verify snapshot "+msg.snapshotID, msg.err) {
			return m, nil
		}
		if !msg.event.Succeeded() {
			m.err = fmt.Errorf("snapshot %s is corrupted: %w", msg.snapshotID, failuresError(msg.event.Failures))
			return m, nil
		}
		m.message = fmt.Sprintf("Snapshot %s verified successfully", msg.snapshotID)

	case snapshotRestoredMsg:
		m.finishOperation()
		if m.fail("restore snapshot "+msg.snapshotID, msg.err) {
			return m, nil
		}
		if !msg.event.Succeeded() {
			m.err = fmt.Errorf("snapshot %s restored with errors: %w", msg.snapshotID, failuresError(msg.event.Failures))
			return m, nil
		}
		m.message = fmt.Sprintf("Snapshot %s restored to %s", msg.snapshotID, msg.event.Destination)
	}

	return m, nil
}

func (m *Model) fail(action string, err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.Canceled) {
		m.err = nil
		m.message = "Cancelled: " + action
		return true
	}

	m.err = fmt.Errorf("failed to %s: %w", action, err)
	m.message = ""

	return true
}

func (m *Model) reloadWithMessage(message, selectID string) tea.Cmd {
	cmd := m.loadSnapshots(selectID)
	m.message = message
	m.preserveMessage = true

	return cmd
}

func (m *Model) consumeMessage() {
	if !m.preserveMessage {
		m.message = ""
	}
	m.preserveMessage = false
}

func failuresError(failures []snapshot.Failure) error {
	errs := make([]error, len(failures))
	for i, failure := range failures {
		errs[i] = fmt.Errorf("%s: %s", failure.Path, failure.Reason)
	}

	return errors.Join(errs...)
}

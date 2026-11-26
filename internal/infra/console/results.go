package console

import (
	"context"
	"errors"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

func (m *Model) handleResult(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case repositoryStatusMsg:
		m.finishOperation()
		if m.fail("open repository", msg.err) {
			return nil
		}
		if !msg.initialized {
			m.openInitConfirmation()
			return nil
		}
		return m.loadSnapshots("")

	case repositoryInitializedMsg:
		m.finishOperation()
		if m.fail("initialize repository", msg.err) {
			return nil
		}
		m.message = "Repository initialized successfully"
		return m.loadSnapshots("")

	case snapshotsLoadedMsg:
		m.finishOperation()
		if m.fail("load snapshots", msg.err) {
			return nil
		}
		m.snapshots.setSnapshots(msg.snapshots, msg.selectID)

	case manifestLoadedMsg:
		m.finishOperation()
		if m.fail("load snapshot", msg.err) {
			return nil
		}
		m.showManifest(msg.manifest)

	case fileContentLoadedMsg:
		m.handleFileContent(&msg)

	case snapshotCreatedMsg:
		m.finishOperation()
		if m.fail("create snapshot", msg.err) {
			return nil
		}
		created := msg.event.Snapshot
		m.screens = nil
		m.message = fmt.Sprintf(
			"Snapshot %s created: %d files, %s",
			created.ID(),
			created.FileCount(),
			formatSize(created.TotalSize()),
		)
		return m.loadSnapshots(created.ID())

	case snapshotVerifiedMsg:
		m.finishOperation()
		if m.fail("verify snapshot "+msg.snapshotID, msg.err) {
			return nil
		}
		if !msg.event.Succeeded() {
			m.err = fmt.Errorf("snapshot %s is corrupted: %w", msg.snapshotID, failuresError(msg.event.Failures))
			return nil
		}
		m.message = fmt.Sprintf("Snapshot %s verified successfully", msg.snapshotID)

	case snapshotRestoredMsg:
		m.finishOperation()
		if m.fail("restore snapshot "+msg.snapshotID, msg.err) {
			return nil
		}
		if !msg.event.Succeeded() {
			m.err = fmt.Errorf("snapshot %s restored with errors: %w", msg.snapshotID, failuresError(msg.event.Failures))
			return nil
		}
		m.message = fmt.Sprintf("Snapshot %s restored to %s", msg.snapshotID, msg.event.Destination)
	}

	return nil
}

func (m *Model) handleFileContent(msg *fileContentLoadedMsg) {
	m.finishOperation()
	if m.fail("read "+msg.request.name, msg.err) {
		return
	}
	if !isText(msg.content) {
		m.err = fmt.Errorf("%q is not a text file", msg.request.name)
		return
	}

	m.push(newPreviewScreen(msg.request.location, msg.request.name, msg.content, msg.truncated))
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

func failuresError(failures []snapshot.Failure) error {
	errs := make([]error, len(failures))
	for i, failure := range failures {
		errs[i] = fmt.Errorf("%s: %s", failure.Path, failure.Reason)
	}

	return errors.Join(errs...)
}

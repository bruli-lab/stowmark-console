package console

import (
	"context"
	"errors"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/bruli-lab/stowmark-console/internal/domain/encryption"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

func (m *Model) handleResult(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case repositoriesLoadedMsg:
		m.finishOperation()
		if m.fail("load repositories", msg.err) {
			return nil
		}
		m.repositories = msg.profiles
		if len(m.repositories) == 1 {
			return m.openRepository(m.repositories[0].Name())
		}
		m.showRepositories()

	case repositoryOpenedMsg:
		m.finishOperation()
		if m.fail("open repository "+msg.name, msg.err) {
			return nil
		}
		m.current = msg.event.Profile
		m.screens = nil
		m.snapshots.setSnapshots(nil, "")
		return m.checkRepository()

	case repositoryStatusMsg:
		m.finishOperation()
		if m.fail("open repository", msg.err) {
			return nil
		}
		return m.applyStatus(msg.status)

	case encryptedRepositoryInitializedMsg:
		return m.handleEncryptedInitialized(&msg)

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

	case fileOpenedMsg:
		m.finishOperation()
		if m.fail("open "+msg.name, msg.err) {
			return nil
		}
		m.message = "Opened " + msg.name + " with the system application"

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
		if len(msg.event.Skipped) > 0 {
			m.message += "; " + skippedSummary(msg.event.Skipped)
			m.dialog = skippedDialog(&msg.event)
		}
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

	case repositoryUnlockedMsg:
		m.handleUnlocked(&msg)

	case repositoryCheckedMsg:
		m.handleChecked(&msg)

	case forgetPlannedMsg, snapshotsForgottenMsg, prunePlannedMsg, repositoryPrunedMsg:
		return m.handleRetentionResult(msg)

	case keysGeneratedMsg, keyRewrappedMsg, keyRotatedMsg:
		return m.handleKeyResult(msg)

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

func (m *Model) applyStatus(status repository.Status) tea.Cmd {
	switch status {
	case repository.StatusUninitialized:
		m.openInitConfirmation()
	case repository.StatusKeyRequired:
		m.err = explainKeyError(m.current, encryption.ErrPrivateKeyRequired)
		if len(m.repositories) > 1 {
			m.showRepositories()
		}
	default:
		return m.loadSnapshots("")
	}

	return nil
}

func (m *Model) handleFileContent(msg *fileContentLoadedMsg) {
	m.finishOperation()
	if m.fail("read "+msg.request.name, msg.err) {
		return
	}
	m.push(newPreviewScreen(msg.request, &msg.content, msg.truncated))
}

func (m *Model) fail(action string, err error) bool {
	return m.failWithKey(action, err, m.current.PrivateKey())
}

func (m *Model) failWithKey(action string, err error, privateKey *string) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.Canceled) {
		m.err = nil
		m.message = "Cancelled: " + action
		return true
	}

	m.err = fmt.Errorf("failed to %s: %w", action,
		explainLockError(explainKeyError(m.current.WithPrivateKey(privateKey), err)))
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

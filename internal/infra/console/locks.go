package console

import (
	"context"
	"errors"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/bruli-lab/stowmark-console/internal/app"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
)

type unlockRequested struct {
	all bool
}

type unlockConfirmed struct {
	all bool
}

type repositoryUnlockedMsg struct {
	event repository.Unlocked
	err   error
}

type lockError struct {
	message string
	cause   error
}

func (e lockError) Error() string {
	return e.message
}

func (e lockError) Unwrap() error {
	return e.cause
}

func explainLockError(err error) error {
	if !errors.Is(err, repository.ErrLocked) {
		return err
	}

	return lockError{
		message: err.Error() + ". If no other Stowmark process uses the repository, " +
			":unlock removes the stale locks and :unlock! every lock",
		cause: err,
	}
}

func (m *Model) handleLockIntent(msg tea.Msg) (tea.Cmd, bool) {
	request, ok := msg.(unlockRequested)
	if !ok {
		return nil, false
	}

	if !m.requireRepository() {
		return nil, true
	}

	if !request.all {
		return emit(unlockConfirmed{}), true
	}

	m.dialog = &confirmDialog{
		title: "Remove every lock of " + m.current.Name() + "?",
		body: "This removes the locks of every Stowmark process, including the ones\n" +
			"of commands that are still running.",
		warning: "Only continue if no other Stowmark process (a backup, a prune, a rekey...)\n" +
			"is using this repository: removing the lock of a running command can\n" +
			"corrupt the repository.",
		hint:      "[y] Remove every lock  [n/Esc] Cancel",
		onConfirm: emit(unlockConfirmed{all: true}),
		onCancel:  emit(statusMessage{text: "Unlock cancelled"}),
	}

	return nil, true
}

func (m *Model) unlockRepository(all bool) tea.Cmd {
	bus := m.buses.Commands
	message := "Removing stale locks..."
	if all {
		message = "Removing every lock..."
	}

	return m.startOperation(message, func(ctx context.Context) tea.Msg {
		event, err := dispatch[repository.Unlocked](ctx, bus, app.UnlockRepositoryCommand{All: all})
		return repositoryUnlockedMsg{event: event, err: err}
	})
}

func (m *Model) handleUnlocked(msg *repositoryUnlockedMsg) {
	m.finishOperation()
	if m.fail("unlock repository "+m.current.Name(), msg.err) {
		return
	}

	report := msg.event.Report
	if msg.event.All {
		m.message = fmt.Sprintf("Removed %s from repository %s", plural(report.Removed, "lock"), m.current.Name())
		return
	}

	m.message = fmt.Sprintf("Removed %s from repository %s", plural(report.Removed, "stale lock"), m.current.Name())
	if report.Active > 0 {
		m.message += fmt.Sprintf("; %s still held by running processes (:unlock! removes them)",
			plural(report.Active, "lock"))
	}
}

func plural(count int, noun string) string {
	if count == 1 {
		return "1 " + noun
	}

	return fmt.Sprintf("%d %ss", count, noun)
}

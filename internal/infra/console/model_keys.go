package console

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.confirmingInit:
		return m.handleInitConfirmationKey(msg)
	case m.snapshotForm != nil:
		return m.handleSnapshotFormKey(msg)
	case m.restoreForm != nil:
		return m.handleRestoreFormKey(msg)
	case m.confirmingRestore():
		return m.handleRestoreConfirmationKey(msg)
	case m.commandMode:
		return m.handleCommandKey(msg)
	case msg.String() == "ctrl+c":
		return m, tea.Quit
	case m.loading:
		return m.handleLoadingKey(msg)
	case m.preview != nil:
		return m.handlePreviewKey(msg)
	case m.view == filesView:
		return m.handleFilesKey(msg)
	default:
		return m.handleSnapshotsKey(msg)
	}
}

func (m *Model) handleLoadingKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyEsc && m.cancelOperation != nil {
		m.cancelOperation()
		m.message = "Cancelling..."
	}

	return m, nil
}

func (m *Model) handleInitConfirmationKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.confirmingInit = false
		m.initWarning = ""
		cmd := m.initRepository()
		return m, cmd

	case "n", "N", "esc", "q", "ctrl+c":
		return m, tea.Quit
	}

	return m, nil
}

func (m *Model) handleSnapshotFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.snapshotForm.handleKey(msg) {
	case promptCancelled:
		m.snapshotForm = nil
		m.message = "Snapshot creation cancelled"

	case promptSubmitted:
		source, err := resolveLocalPath(m.snapshotForm.trimmedValue())
		if err != nil {
			m.snapshotForm.fail("Origin is required")
			return m, nil
		}

		m.snapshotForm = nil

		cmd := m.createSnapshot(source)
		return m, cmd

	case promptEditing:
	}

	return m, nil
}

func (m *Model) handleRestoreFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.restoreForm.handleKey(msg) {
	case promptCancelled:
		m.restoreForm = nil
		m.restore = nil
		m.message = "Restore cancelled"

	case promptSubmitted:
		destination, err := resolveLocalPath(m.restoreForm.trimmedValue())
		if err != nil {
			m.restoreForm.fail(err.Error())
			return m, nil
		}

		m.restore.destination = destination
		m.restore.warning = restoreDestinationWarning(destination)
		m.restoreForm = nil

	case promptEditing:
	}

	return m, nil
}

func (m *Model) handleRestoreConfirmationKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		request := *m.restore
		m.restore = nil

		cmd := m.restoreSnapshot(request)
		return m, cmd

	case "n", "N", "esc", "q", "ctrl+c":
		m.restore = nil
		m.err = nil
		m.message = "Restore cancelled"
	}

	return m, nil
}

func (m *Model) handleCommandKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		m.commandMode = false
		m.commandInput = ""

	case tea.KeyEnter:
		command := m.commandInput
		m.commandMode = false
		m.commandInput = ""

		if command == "q" {
			return m, tea.Quit
		}

	case tea.KeyBackspace:
		runes := []rune(m.commandInput)
		if len(runes) == 0 {
			m.commandMode = false
			return m, nil
		}
		m.commandInput = string(runes[:len(runes)-1])

	case tea.KeyRunes:
		m.commandInput += string(msg.Runes)

	default:
	}

	return m, nil
}

func (m *Model) handlePreviewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace", "q":
		m.preview = nil
		m.err = nil

	case ":":
		m.commandMode = true
		m.commandInput = ""

	case "up", "k":
		m.preview.offset = max(0, m.preview.offset-1)

	case "down", "j":
		m.preview.offset = min(m.maxPreviewOffset(), m.preview.offset+1)

	case "pgup":
		m.preview.offset = max(0, m.preview.offset-m.previewVisibleLines())

	case "pgdown", " ":
		m.preview.offset = min(m.maxPreviewOffset(), m.preview.offset+m.previewVisibleLines())

	case "home", "g":
		m.preview.offset = 0

	case "end", "G":
		m.preview.offset = m.maxPreviewOffset()
	}

	return m, nil
}

func (m *Model) handleSnapshotsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if cmd, handled := m.handleSnapshotActionKey(msg); handled {
		return m, cmd
	}

	switch msg.String() {
	case "enter", "right", "l":
		selected, ok := m.selectedSnapshot()
		if !ok {
			return m, nil
		}
		cmd := m.openSnapshot(selected.ID())
		return m, cmd

	case "r":
		selected, _ := m.selectedSnapshot()
		cmd := m.loadSnapshots(selected.ID())
		return m, cmd

	default:
		m.snapshotCursor = moveCursor(msg.String(), m.snapshotCursor, len(m.snapshots), m.contentHeight())
	}

	return m, nil
}

func (m *Model) handleFilesKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if cmd, handled := m.handleSnapshotActionKey(msg); handled {
		return m, cmd
	}

	switch msg.String() {
	case "enter", "right", "l":
		entry, ok := m.selectedEntry()
		if !ok {
			return m, nil
		}
		if entry.IsDir() {
			m.enterSelectedEntry()
			return m, nil
		}
		file, found := m.manifest.FileAt(m.directory, entry.Name())
		if !found {
			return m, nil
		}
		cmd := m.loadFileContent(m.manifest.SnapshotID(), entry.Name(), file)
		return m, cmd

	case "esc", "backspace", "left", "h":
		m.err = nil
		m.message = ""
		m.leaveDirectory()

	case "r":
		cmd := m.openSnapshot(m.manifest.SnapshotID())
		return m, cmd

	default:
		m.fileCursor = moveCursor(msg.String(), m.fileCursor, len(m.entries), m.contentHeight())
	}

	return m, nil
}

func (m *Model) handleSnapshotActionKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch msg.String() {
	case ":":
		m.commandMode = true
		m.commandInput = ""
		return nil, true

	case "c":
		m.snapshotForm = newTextPrompt("Create snapshot", "Origin", "")
		m.message = ""
		m.err = nil
		return nil, true

	case "v":
		snapshotID, ok := m.currentSnapshotID()
		if !ok {
			return nil, true
		}
		return m.verifySnapshot(snapshotID), true

	case "x":
		snapshotID, ok := m.currentSnapshotID()
		if !ok {
			return nil, true
		}
		m.restore = &restoreRequest{snapshotID: snapshotID}
		m.restoreForm = newTextPrompt(
			"Restore snapshot "+snapshotID,
			"Destination",
			defaultRestoreDestination(snapshotID),
		)
		m.err = nil
		m.message = ""
		return nil, true
	}

	return nil, false
}

func moveCursor(key string, cursor, total, pageSize int) int {
	if total == 0 {
		return 0
	}

	switch key {
	case "up", "k":
		cursor--
	case "down", "j":
		cursor++
	case "pgup":
		cursor -= pageSize
	case "pgdown":
		cursor += pageSize
	case "home", "g":
		cursor = 0
	case "end", "G":
		cursor = total - 1
	}

	return min(max(cursor, 0), total-1)
}

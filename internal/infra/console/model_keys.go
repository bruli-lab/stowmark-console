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
		return m, nil
	case m.previewing:
		return m.handlePreviewKey(msg)
	default:
		return m.handleBrowserKey(msg)
	}
}

func (m *Model) handleInitConfirmationKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.confirmingInit = false
		m.initWarning = ""
		m.loading = true
		m.message = "Initializing repository..."

		return m, initRepositoryCmd(m.ctx, m.repoPath)

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
		origin, err := resolveLocalPath(m.snapshotForm.trimmedValue())
		if err != nil {
			m.snapshotForm.fail("Origin is required")
			return m, nil
		}

		m.snapshotForm = nil
		m.loading = true
		m.err = nil
		m.message = "Creating snapshot..."

		return m, createSnapshotCmd(m.ctx, m.repoPath, origin, m.privateKey)

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
		m.loading = true
		m.err = nil
		m.message = "Restoring snapshot..."

		return m, restoreSnapshotCmd(
			m.ctx,
			m.repoPath,
			request.snapshotID,
			request.destination,
			m.privateKey,
		)

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
	case "esc", "backspace":
		m.previewing = false
		m.previewPath = ""
		m.previewLines = nil
		m.previewOffset = 0
		m.err = nil

	case ":":
		m.commandMode = true
		m.commandInput = ""

	case "up", "k":
		if m.previewOffset > 0 {
			m.previewOffset--
		}

	case "down", "j":
		if m.previewOffset < m.maxPreviewOffset() {
			m.previewOffset++
		}

	case "home", "g":
		m.previewOffset = 0

	case "end", "G":
		m.previewOffset = m.maxPreviewOffset()
	}

	return m, nil
}

func (m *Model) maxPreviewOffset() int {
	return max(0, len(m.previewLines)-m.previewVisibleLines())
}

func (m *Model) handleBrowserKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case ":":
		m.commandMode = true
		m.commandInput = ""

	case "c":
		if !m.isSnapshotsDirectory() {
			return m, nil
		}

		m.snapshotForm = newTextPrompt("Create snapshot", "Origin", "")
		m.message = ""
		m.err = nil

	case "v":
		snapshotID, ok := m.selectedSnapshotID()
		if !ok {
			return m, nil
		}

		m.loading = true
		m.err = nil
		m.message = "Verifying snapshot..."

		return m, verifySnapshotCmd(m.ctx, m.repoPath, snapshotID, m.privateKey)

	case "x":
		snapshotID, ok := m.selectedSnapshotID()
		if !ok {
			return m, nil
		}

		m.restore = &restoreRequest{snapshotID: snapshotID}
		m.restoreForm = newTextPrompt(
			"Restore snapshot "+snapshotID,
			"Destination",
			defaultRestoreDestination(snapshotID),
		)
		m.err = nil
		m.message = ""

	case "esc", "backspace", "left", "h":
		return m.startLoading(m.openParentDirectory())

	case "enter":
		entry, ok := m.selectedEntry()
		if !ok {
			return m, nil
		}

		if entry.IsDir() {
			return m.startLoading(m.openSelectedDirectory())
		}

		return m.startLoading(m.previewSelectedFile())

	case "r":
		return m.startLoading(loadRepositoryContent(m.ctx, m.folderRepo, m.currentPath))

	case "up", "k":
		m.message = ""
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		m.message = ""
		if m.cursor < len(m.entries)-1 {
			m.cursor++
		}

	case "home", "g":
		m.message = ""
		m.cursor = 0

	case "end", "G":
		m.message = ""
		if len(m.entries) > 0 {
			m.cursor = len(m.entries) - 1
		}
	}

	return m, nil
}

func (m *Model) startLoading(cmd tea.Cmd) (tea.Model, tea.Cmd) {
	if cmd == nil {
		return m, nil
	}

	m.loading = true
	m.err = nil
	m.message = ""

	return m, cmd
}

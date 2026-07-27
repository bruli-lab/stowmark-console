package console

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case filePreviewLoadedMsg:
		m.loading = false
		m.err = msg.err

		if msg.err != nil {
			return m, nil
		}

		m.previewPath = msg.path
		m.previewLines = strings.Split(msg.content, "\n")
		m.previewOffset = 0
		m.previewing = true

	case snapshotVerifiedMsg:
		m.loading = false

		if msg.err != nil {
			m.err = fmt.Errorf(
				"failed to verify snapshot %s: %w",
				msg.snapshotID,
				msg.err,
			)
			m.message = ""
			return m, nil
		}

		m.err = nil
		m.message = fmt.Sprintf(
			"Snapshot %s verified successfully",
			msg.snapshotID,
		)

		return m, nil

	case snapshotRestoredMsg:
		m.loading = false

		if msg.err != nil {
			m.err = fmt.Errorf(
				"failed to restore snapshot %s: %w",
				msg.snapshotID,
				msg.err,
			)
			m.message = ""
			return m, nil
		}

		m.err = nil
		m.message = fmt.Sprintf(
			"Snapshot %s restored successfully",
			msg.snapshotID,
		)

		return m, nil

	case initRepositoryMsg:
		m.loading = false
		m.err = msg.err

		if msg.err != nil {
			return m, nil
		}

		m.message = "Repository initialized successfully"
		m.loading = true

		return m, loadRepositoryContent(
			m.ctx,
			m.folderRepo,
			m.repoPath,
		)
	case snapshotCreatedMsg:
		m.loading = false

		if msg.err != nil {
			m.err = fmt.Errorf("failed to create snapshot: %w", msg.err)
			m.message = ""
			return m, nil
		}

		m.err = nil
		m.message = fmt.Sprintf(
			"Snapshot %s created: %d files, %s",
			msg.snapshotID,
			msg.fileCount,
			formatSize(msg.totalSize),
		)
		m.preserveMessage = true
		m.loading = true

		return m, loadRepositoryContent(
			m.ctx,
			m.folderRepo,
			filepath.Join(filepath.Clean(m.repoPath), "snapshots"),
		)

	case repositoryLoadedMsg:
		m.loading = false
		m.err = msg.err
		if !m.preserveMessage {
			m.message = ""
		}
		m.preserveMessage = false
		m.commandMode = false
		m.commandInput = ""

		if msg.err != nil {
			return m, nil
		}

		m.currentPath = msg.path
		m.entries = msg.entries
		m.cursor = 0

	case editorFinishedMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("opening file with vim: %w", msg.err)
			return m, nil
		}

		m.loading = true
		m.err = nil

		return m, loadRepositoryContent(
			m.ctx,
			m.folderRepo,
			m.currentPath,
		)

	case tea.KeyMsg:
		if m.creatingSnapshot {
			switch msg.String() {
			case "ctrl+c", "esc":
				m.creatingSnapshot = false
				m.snapshotOrigin = ""
				m.snapshotFormError = ""
				m.message = "Snapshot creation cancelled"

				return m, nil

			case "enter":
				origin := strings.TrimSpace(m.snapshotOrigin)
				if origin == "" {
					m.snapshotFormError = "Origin is required"
					return m, nil
				}

				m.creatingSnapshot = false
				m.snapshotOrigin = ""
				m.snapshotFormError = ""
				m.loading = true
				m.err = nil
				m.message = "Creating snapshot..."

				return m, createSnapshotCmd(
					m.ctx,
					m.repoPath,
					origin,
				)

			case "backspace":
				runes := []rune(m.snapshotOrigin)
				if len(runes) > 0 {
					m.snapshotOrigin = string(runes[:len(runes)-1])
				}
				m.snapshotFormError = ""

			default:
				if msg.Type == tea.KeyRunes {
					m.snapshotOrigin += string(msg.Runes)
					m.snapshotFormError = ""
				}
			}

			return m, nil
		}

		if m.confirmingRestore {
			switch msg.String() {
			case "y", "Y", "enter":
				snapshotID := m.pendingSnapshotID

				m.confirmingRestore = false
				m.pendingSnapshotID = ""
				m.loading = true
				m.err = nil
				m.message = "Restoring snapshot..."

				return m, restoreSnapshotCmd(
					m.ctx,
					m.repoPath,
					snapshotID,
				)

			case "n", "N", "esc", "q":
				m.confirmingRestore = false
				m.pendingSnapshotID = ""
				m.message = "Restore cancelled"
				m.err = nil

				return m, nil
			}

			return m, nil
		}
		if m.commandMode {
			switch msg.String() {
			case "esc":
				m.commandMode = false
				m.commandInput = ""

			case "enter":
				command := m.commandInput
				m.commandMode = false
				m.commandInput = ""

				if command == "q" {
					return m, tea.Quit
				}

			case "backspace":
				runes := []rune(m.commandInput)

				if len(runes) > 0 {
					m.commandInput = string(runes[:len(runes)-1])
				} else {
					m.commandMode = false
				}

			default:
				if msg.Type == tea.KeyRunes {
					m.commandInput += string(msg.Runes)
				}
			}

			return m, nil
		}

		if m.previewing {
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
				maxOffset := max(
					0,
					len(m.previewLines)-m.previewVisibleLines(),
				)

				if m.previewOffset < maxOffset {
					m.previewOffset++
				}

			case "home", "g":
				m.previewOffset = 0

			case "end", "G":
				m.previewOffset = max(
					0,
					len(m.previewLines)-m.previewVisibleLines(),
				)

			case "ctrl+c":
				return m, tea.Quit
			}

			return m, nil
		}

		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case ":":
			m.commandMode = true
			m.commandInput = ""

		case "c":
			if !m.isSnapshotsDirectory() {
				return m, nil
			}

			m.creatingSnapshot = true
			m.snapshotOrigin = ""
			m.snapshotFormError = ""
			m.message = ""
			m.err = nil

			return m, nil

		case "v":
			snapshotPath, ok := m.selectedSnapshotPath()
			if !ok {
				return m, nil
			}
			snapshotFile := filepath.Base(snapshotPath)
			snapshotID := strings.TrimSuffix(snapshotFile, filepath.Ext(snapshotFile))

			m.loading = true
			m.message = "Verifying snapshot..."
			return m, verifySnapshotCmd(m.ctx, m.repoPath, snapshotID)

		case "x":
			snapshotPath, ok := m.selectedSnapshotPath()
			if !ok {
				return m, nil
			}

			snapshotFile := filepath.Base(snapshotPath)
			snapshotID := strings.TrimSuffix(
				snapshotFile,
				filepath.Ext(snapshotFile),
			)

			m.requestRestore(snapshotID)

			return m, nil

		case "esc", "backspace", "left", "h":
			rootPath := filepath.Clean(m.conf.Repository())
			currentPath := filepath.Clean(m.currentPath)

			if currentPath != rootPath {
				m.loading = true
				m.err = nil
				m.message = ""

				cmd := m.openParentDirectory()
				return m, cmd
			}

		case "enter":
			if len(m.entries) == 0 ||
				m.cursor < 0 ||
				m.cursor >= len(m.entries) {
				return m, nil
			}

			m.loading = true
			m.err = nil
			m.message = ""

			if m.entries[m.cursor].IsDir() {
				cmd := m.openSelectedDirectory()
				return m, cmd
			}

			cmd := m.previewSelectedFile()
			return m, cmd

		case "e":
			m.message = ""
			cmd := m.openSelectedFile()
			return m, cmd

		case "r":
			m.loading = true
			m.err = nil
			m.message = ""

			return m, loadRepositoryContent(
				m.ctx,
				m.folderRepo,
				m.currentPath,
			)

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
	}

	return m, nil
}

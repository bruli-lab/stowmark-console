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

	case tea.KeyMsg:
		return m.handleKey(msg)

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
			m.err = fmt.Errorf("failed to verify snapshot %s: %w", msg.snapshotID, msg.err)
			m.message = ""
			return m, nil
		}

		m.err = nil
		m.message = fmt.Sprintf("Snapshot %s verified successfully", msg.snapshotID)

	case snapshotRestoredMsg:
		m.loading = false

		if msg.err != nil {
			m.err = fmt.Errorf("failed to restore snapshot %s: %w", msg.snapshotID, msg.err)
			m.message = ""
			return m, nil
		}

		m.err = nil
		m.message = fmt.Sprintf("Snapshot %s restored to %s", msg.snapshotID, msg.destination)

	case initRepositoryMsg:
		m.loading = false
		m.err = msg.err

		if msg.err != nil {
			m.message = ""
			return m, nil
		}

		m.message = "Repository initialized successfully"
		m.preserveMessage = true
		m.loading = true

		return m, loadRepositoryContent(m.ctx, m.folderRepo, m.currentPath)

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
			filepath.Join(m.rootPath(), "snapshots"),
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
	}

	return m, nil
}

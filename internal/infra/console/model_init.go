package console

import (
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) Init() tea.Cmd {
	configPath := filepath.Join(m.repoPath, "config.json")

	_, err := os.Stat(configPath)
	switch {
	case err == nil:
		return loadRepositoryContent(
			m.ctx,
			m.folderRepo,
			m.currentPath,
		)

	case os.IsNotExist(err):
		m.loading = true
		m.message = "Initializing repository..."

		return initRepositoryCmd(
			m.ctx,
			m.repoPath,
		)

	default:
		return func() tea.Msg {
			return repositoryLoadedMsg{
				path: m.repoPath,
				err:  err,
			}
		}
	}
}

package console

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) Init() tea.Cmd {
	_, err := os.Stat(filepath.Join(m.repoPath, "config.json"))

	switch {
	case err == nil:
		return loadRepositoryContent(m.ctx, m.folderRepo, m.currentPath)

	case errors.Is(err, fs.ErrNotExist):
		m.loading = false
		m.confirmingInit = true
		m.initWarning = initRepositoryWarning(m.repoPath)

		return nil

	default:
		return func() tea.Msg {
			return repositoryLoadedMsg{path: m.repoPath, err: err}
		}
	}
}

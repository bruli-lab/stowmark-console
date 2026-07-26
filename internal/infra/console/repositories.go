package console

import (
	"context"

	"github.com/bruli-lab/stowmark-console.git/internal/domain/repository"
	tea "github.com/charmbracelet/bubbletea"
)

func loadRepositoryContent(ctx context.Context, repo repository.FolderRepository, path string) tea.Cmd {
	return func() tea.Msg {
		svc := repository.NewRead(repo)
		entries, err := svc.ReadEntries(ctx, path)
		return repositoryLoadedMsg{
			entries: entries,
			err:     err,
			path:    path,
		}
	}
}

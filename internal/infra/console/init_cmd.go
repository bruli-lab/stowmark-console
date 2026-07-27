package console

import (
	"context"

	"github.com/bruli-lab/stowmark/pkg/stowmark"
	tea "github.com/charmbracelet/bubbletea"
)

func initRepositoryCmd(
	ctx context.Context,
	repositoryPath string,
) tea.Cmd {
	return func() tea.Msg {
		handler, err := stowmark.NewHandler(repositoryPath)
		if err != nil {
			return initRepositoryMsg{err: err}
		}

		compression := stowmark.Compression{
			Type:  "none",
			Level: nil,
		}
		if err := handler.Init(ctx, repositoryPath, &compression); err != nil {
			return initRepositoryMsg{err: err}
		}

		return initRepositoryMsg{}
	}
}

package console

import (
	"context"

	"github.com/bruli-lab/stowmark/pkg/stowmark"
	tea "github.com/charmbracelet/bubbletea"
)

func initRepositoryCmd(ctx context.Context, repositoryPath string, settings InitSettings) tea.Cmd {
	return func() tea.Msg {
		handler, err := stowmark.NewHandler(ctx, repositoryPath)
		if err != nil {
			return initRepositoryMsg{err: err}
		}
		defer func() { _ = handler.Close() }()

		if err := handler.InitWithOptions(ctx, settings.options()); err != nil {
			return initRepositoryMsg{err: err}
		}

		return initRepositoryMsg{}
	}
}

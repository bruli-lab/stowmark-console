package console

import (
	"context"

	"github.com/bruli-lab/stowmark/pkg/stowmark"
	tea "github.com/charmbracelet/bubbletea"
)

func initRepositoryCmd(
	ctx context.Context,
	repositoryPath string,
	formatVersion int,
	publicKey *string,
	force bool,
) tea.Cmd {
	return func() tea.Msg {
		handler, err := stowmark.NewHandler(ctx, repositoryPath)
		if err != nil {
			return initRepositoryMsg{err: err}
		}

		compression := stowmark.Compression{
			Type:  "none",
			Level: nil,
		}
		if err := handler.Init(ctx, &compression, formatVersion, publicKey, force); err != nil {
			return initRepositoryMsg{err: err}
		}

		return initRepositoryMsg{}
	}
}

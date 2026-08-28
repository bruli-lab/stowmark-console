package console

import (
	"context"

	"github.com/bruli-lab/stowmark/pkg/stowmark"
	tea "github.com/charmbracelet/bubbletea"
)

func createSnapshotCmd(
	ctx context.Context,
	repositoryPath string,
	origin string,
	privateKey *string,
) tea.Cmd {
	return func() tea.Msg {
		msg := snapshotCreatedMsg{}

		handler, err := stowmark.NewHandler(ctx, repositoryPath)
		if err != nil {
			msg.err = err
			return msg
		}

		result, err := handler.CreateSnapshot(ctx, origin, privateKey)
		if err != nil {
			msg.err = err
			return msg
		}

		msg.snapshotID = result.ID
		msg.fileCount = result.FileCount
		msg.totalSize = result.TotalSize

		return msg
	}
}

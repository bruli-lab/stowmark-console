package console

import (
	"context"

	"github.com/bruli-lab/stowmark/pkg/stowmark"
	tea "github.com/charmbracelet/bubbletea"
)

func restoreSnapshotCmd(
	ctx context.Context,
	repositoryPath, snapshotID, destination string,
	privateKey *string,
) tea.Cmd {
	return func() tea.Msg {
		msg := snapshotRestoredMsg{snapshotID: snapshotID, destination: destination}

		handler, err := stowmark.NewHandler(ctx, repositoryPath)
		if err != nil {
			msg.err = err
			return msg
		}
		defer func() { _ = handler.Close() }()

		result, err := handler.RestoreSnapshot(ctx, snapshotID, &destination, privateKey)
		if err != nil {
			msg.err = err
			return msg
		}

		msg.err = failuresError(result)

		return msg
	}
}

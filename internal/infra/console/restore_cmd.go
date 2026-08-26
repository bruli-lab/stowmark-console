package console

import (
	"context"
	"errors"
	"fmt"

	"github.com/bruli-lab/stowmark/pkg/stowmark"
	tea "github.com/charmbracelet/bubbletea"
)

func restoreSnapshotCmd(
	ctx context.Context,
	repositoryPath, snapshotID string,
	destinationPath, privateKey *string,
) tea.Cmd {
	return func() tea.Msg {
		msg := snapshotRestoredMsg{}
		msg.snapshotID = snapshotID
		hand, err := stowmark.NewHandler(ctx, repositoryPath)
		if err != nil {
			msg.err = err
			return msg
		}
		result, err := hand.RestoreSnapshot(ctx, snapshotID, destinationPath, privateKey)
		if err != nil {
			msg.err = err
			return msg
		}
		if !result.IsSuccess {
			var errs error
			for _, failed := range result.Failed {
				errs = errors.Join(errs, fmt.Errorf("%s: %s", failed.Path, failed.Reason))
			}
			msg.err = errs
			return msg
		}
		return msg
	}
}

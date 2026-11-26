package console

import (
	"context"
	"errors"
	"fmt"

	"github.com/bruli-lab/stowmark/pkg/stowmark"
	tea "github.com/charmbracelet/bubbletea"
)

func verifySnapshotCmd(ctx context.Context, repositoryPath, snapshotID string, privateKey *string) tea.Cmd {
	return func() tea.Msg {
		msg := snapshotVerifiedMsg{snapshotID: snapshotID}

		handler, err := stowmark.NewHandler(ctx, repositoryPath)
		if err != nil {
			msg.err = err
			return msg
		}
		defer func() { _ = handler.Close() }()

		result, err := handler.VerifySnapshot(ctx, snapshotID, privateKey)
		if err != nil {
			msg.err = err
			return msg
		}

		msg.err = failuresError(result)

		return msg
	}
}

func failuresError(result *stowmark.Result) error {
	if result.IsSuccess {
		return nil
	}

	var errs error
	for _, failed := range result.Failed {
		errs = errors.Join(errs, fmt.Errorf("%s: %s", failed.Path, failed.Reason))
	}

	return errs
}

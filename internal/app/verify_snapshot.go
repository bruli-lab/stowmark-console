package app

import (
	"context"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

const VerifySnapshotCommandName = "verify-snapshot"

type VerifySnapshotCommand struct {
	SnapshotID string
}

func (VerifySnapshotCommand) Name() string {
	return VerifySnapshotCommandName
}

type VerifySnapshot struct {
	snapshots snapshot.Repository
}

func NewVerifySnapshot(snapshots snapshot.Repository) VerifySnapshot {
	return VerifySnapshot{snapshots: snapshots}
}

func (h VerifySnapshot) Handle(ctx context.Context, command cqs.Command) ([]event.Event, error) {
	cmd, ok := command.(VerifySnapshotCommand)
	if !ok {
		return nil, cqs.NewInvalidCommandError(VerifySnapshotCommandName, command.Name())
	}

	if cmd.SnapshotID == "" {
		return nil, snapshot.ErrSnapshotIDRequired
	}

	failures, err := h.snapshots.Verify(ctx, cmd.SnapshotID)
	if err != nil {
		return nil, err
	}

	return []event.Event{snapshot.NewVerified(cmd.SnapshotID, failures)}, nil
}

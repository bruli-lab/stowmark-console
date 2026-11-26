package app

import (
	"context"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

const RestoreSnapshotCommandName = "restore-snapshot"

type RestoreSnapshotCommand struct {
	SnapshotID  string
	Destination string
}

func (RestoreSnapshotCommand) Name() string {
	return RestoreSnapshotCommandName
}

type RestoreSnapshot struct {
	snapshots snapshot.Repository
}

func NewRestoreSnapshot(snapshots snapshot.Repository) RestoreSnapshot {
	return RestoreSnapshot{snapshots: snapshots}
}

func (h RestoreSnapshot) Handle(ctx context.Context, command cqs.Command) ([]event.Event, error) {
	cmd, ok := command.(RestoreSnapshotCommand)
	if !ok {
		return nil, cqs.NewInvalidCommandError(RestoreSnapshotCommandName, command.Name())
	}

	if cmd.SnapshotID == "" {
		return nil, snapshot.ErrSnapshotIDRequired
	}

	if err := snapshot.ValidateDestination(cmd.Destination); err != nil {
		return nil, err
	}

	failures, err := h.snapshots.Restore(ctx, cmd.SnapshotID, cmd.Destination)
	if err != nil {
		return nil, err
	}

	return []event.Event{snapshot.NewRestored(cmd.SnapshotID, cmd.Destination, failures)}, nil
}

package app

import (
	"context"

	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

type RestoreSnapshotCommand struct {
	SnapshotID  string
	Destination string
}

type RestoreSnapshot struct {
	snapshots snapshot.Repository
}

func NewRestoreSnapshot(snapshots snapshot.Repository) *RestoreSnapshot {
	return &RestoreSnapshot{snapshots: snapshots}
}

func (h *RestoreSnapshot) Handle(ctx context.Context, cmd RestoreSnapshotCommand) (snapshot.Restored, error) {
	if cmd.SnapshotID == "" {
		return snapshot.Restored{}, snapshot.ErrSnapshotIDRequired
	}

	if err := snapshot.ValidateDestination(cmd.Destination); err != nil {
		return snapshot.Restored{}, err
	}

	failures, err := h.snapshots.Restore(ctx, cmd.SnapshotID, cmd.Destination)
	if err != nil {
		return snapshot.Restored{}, err
	}

	return snapshot.Restored{
		SnapshotID:  cmd.SnapshotID,
		Destination: cmd.Destination,
		Failures:    failures,
	}, nil
}

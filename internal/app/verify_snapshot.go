package app

import (
	"context"

	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

type VerifySnapshotCommand struct {
	SnapshotID string
}

type VerifySnapshot struct {
	snapshots snapshot.Repository
}

func NewVerifySnapshot(snapshots snapshot.Repository) *VerifySnapshot {
	return &VerifySnapshot{snapshots: snapshots}
}

func (h *VerifySnapshot) Handle(ctx context.Context, cmd VerifySnapshotCommand) (snapshot.Verified, error) {
	if cmd.SnapshotID == "" {
		return snapshot.Verified{}, snapshot.ErrSnapshotIDRequired
	}

	failures, err := h.snapshots.Verify(ctx, cmd.SnapshotID)
	if err != nil {
		return snapshot.Verified{}, err
	}

	return snapshot.Verified{SnapshotID: cmd.SnapshotID, Failures: failures}, nil
}

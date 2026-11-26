package app

import (
	"context"

	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

type CreateSnapshotCommand struct {
	Source string
}

type CreateSnapshot struct {
	snapshots snapshot.Repository
}

func NewCreateSnapshot(snapshots snapshot.Repository) *CreateSnapshot {
	return &CreateSnapshot{snapshots: snapshots}
}

func (h *CreateSnapshot) Handle(ctx context.Context, cmd CreateSnapshotCommand) (snapshot.Created, error) {
	if cmd.Source == "" {
		return snapshot.Created{}, snapshot.ErrSourceRequired
	}

	created, err := h.snapshots.Create(ctx, cmd.Source)
	if err != nil {
		return snapshot.Created{}, err
	}

	return snapshot.Created{Snapshot: created}, nil
}

package app

import (
	"context"

	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

type ListSnapshotsQuery struct{}

type ListSnapshots struct {
	snapshots snapshot.Repository
}

func NewListSnapshots(snapshots snapshot.Repository) *ListSnapshots {
	return &ListSnapshots{snapshots: snapshots}
}

func (h *ListSnapshots) Handle(ctx context.Context, _ ListSnapshotsQuery) ([]snapshot.Snapshot, error) {
	snapshots, err := h.snapshots.List(ctx)
	if err != nil {
		return nil, err
	}

	return snapshot.SortByNewest(snapshots), nil
}

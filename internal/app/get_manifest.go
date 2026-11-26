package app

import (
	"context"

	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

type GetManifestQuery struct {
	SnapshotID string
}

type GetManifest struct {
	snapshots snapshot.Repository
}

func NewGetManifest(snapshots snapshot.Repository) *GetManifest {
	return &GetManifest{snapshots: snapshots}
}

func (h *GetManifest) Handle(ctx context.Context, query GetManifestQuery) (snapshot.Manifest, error) {
	if query.SnapshotID == "" {
		return snapshot.Manifest{}, snapshot.ErrSnapshotIDRequired
	}

	return h.snapshots.Manifest(ctx, query.SnapshotID)
}

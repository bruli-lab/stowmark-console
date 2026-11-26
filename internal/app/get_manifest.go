package app

import (
	"context"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

const GetManifestQueryName = "get-manifest"

type GetManifestQuery struct {
	SnapshotID string
}

func (GetManifestQuery) Name() string {
	return GetManifestQueryName
}

type GetManifest struct {
	snapshots snapshot.Repository
}

func NewGetManifest(snapshots snapshot.Repository) GetManifest {
	return GetManifest{snapshots: snapshots}
}

func (h GetManifest) Handle(ctx context.Context, query cqs.Query) (any, error) {
	q, ok := query.(GetManifestQuery)
	if !ok {
		return nil, cqs.NewInvalidQueryError(GetManifestQueryName, query.Name())
	}

	if q.SnapshotID == "" {
		return nil, snapshot.ErrSnapshotIDRequired
	}

	return h.snapshots.Manifest(ctx, q.SnapshotID)
}

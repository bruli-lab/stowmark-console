package app

import (
	"context"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

const ListSnapshotsQueryName = "list-snapshots"

type ListSnapshotsQuery struct{}

func (ListSnapshotsQuery) Name() string {
	return ListSnapshotsQueryName
}

type ListSnapshots struct {
	snapshots snapshot.Repository
}

func NewListSnapshots(snapshots snapshot.Repository) ListSnapshots {
	return ListSnapshots{snapshots: snapshots}
}

func (h ListSnapshots) Handle(ctx context.Context, query cqs.Query) (any, error) {
	if _, ok := query.(ListSnapshotsQuery); !ok {
		return nil, cqs.NewInvalidQueryError(ListSnapshotsQueryName, query.Name())
	}

	snapshots, err := h.snapshots.List(ctx)
	if err != nil {
		return nil, err
	}

	return snapshot.SortByNewest(snapshots), nil
}

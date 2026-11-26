package app

import (
	"context"
	"errors"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

const GetFileContentQueryName = "get-file-content"

var ErrInvalidMaxSize = errors.New("max size must be positive")

type GetFileContentQuery struct {
	SnapshotID string
	Path       string
	MaxSize    int64
}

func (GetFileContentQuery) Name() string {
	return GetFileContentQueryName
}

type GetFileContent struct {
	snapshots snapshot.Repository
}

func NewGetFileContent(snapshots snapshot.Repository) GetFileContent {
	return GetFileContent{snapshots: snapshots}
}

func (h GetFileContent) Handle(ctx context.Context, query cqs.Query) (any, error) {
	q, ok := query.(GetFileContentQuery)
	if !ok {
		return nil, cqs.NewInvalidQueryError(GetFileContentQueryName, query.Name())
	}

	switch {
	case q.SnapshotID == "":
		return nil, snapshot.ErrSnapshotIDRequired
	case q.MaxSize <= 0:
		return nil, ErrInvalidMaxSize
	}

	return h.snapshots.FileContent(ctx, q.SnapshotID, q.Path, q.MaxSize)
}

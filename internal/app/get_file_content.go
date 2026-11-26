package app

import (
	"context"
	"errors"

	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

var ErrInvalidMaxSize = errors.New("max size must be positive")

type GetFileContentQuery struct {
	SnapshotID string
	Path       string
	MaxSize    int64
}

type GetFileContent struct {
	snapshots snapshot.Repository
}

func NewGetFileContent(snapshots snapshot.Repository) *GetFileContent {
	return &GetFileContent{snapshots: snapshots}
}

func (h *GetFileContent) Handle(ctx context.Context, query GetFileContentQuery) ([]byte, error) {
	if query.SnapshotID == "" {
		return nil, snapshot.ErrSnapshotIDRequired
	}

	if query.MaxSize <= 0 {
		return nil, ErrInvalidMaxSize
	}

	return h.snapshots.FileContent(ctx, query.SnapshotID, query.Path, query.MaxSize)
}

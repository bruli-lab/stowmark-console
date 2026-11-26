package app

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

var (
	ErrFilePathRequired            = errors.New("file path is required")
	ErrExtractDirectoryNotAbsolute = errors.New("extraction directory must be absolute")
)

type ExtractFileCommand struct {
	SnapshotID string
	Path       string
	Directory  string
}

type ExtractFile struct {
	snapshots snapshot.Repository
}

func NewExtractFile(snapshots snapshot.Repository) *ExtractFile {
	return &ExtractFile{snapshots: snapshots}
}

func (h *ExtractFile) Handle(ctx context.Context, cmd ExtractFileCommand) (snapshot.FileExtracted, error) {
	switch {
	case cmd.SnapshotID == "":
		return snapshot.FileExtracted{}, snapshot.ErrSnapshotIDRequired
	case cmd.Path == "":
		return snapshot.FileExtracted{}, ErrFilePathRequired
	case !filepath.IsAbs(cmd.Directory):
		return snapshot.FileExtracted{}, ErrExtractDirectoryNotAbsolute
	}

	location, err := h.snapshots.ExtractFile(ctx, cmd.SnapshotID, cmd.Path, cmd.Directory)
	if err != nil {
		return snapshot.FileExtracted{}, err
	}

	return snapshot.FileExtracted{SnapshotID: cmd.SnapshotID, Path: cmd.Path, Location: location}, nil
}

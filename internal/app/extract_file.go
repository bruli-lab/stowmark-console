package app

import (
	"context"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

const ExtractFileCommandName = "extract-file"

type ExtractFileCommand struct {
	SnapshotID string
	Path       string
	Directory  string
}

func (ExtractFileCommand) Name() string {
	return ExtractFileCommandName
}

type ExtractFile struct {
	snapshots snapshot.Repository
}

func NewExtractFile(snapshots snapshot.Repository) ExtractFile {
	return ExtractFile{snapshots: snapshots}
}

func (h ExtractFile) Handle(ctx context.Context, command cqs.Command) ([]event.Event, error) {
	cmd, ok := command.(ExtractFileCommand)
	if !ok {
		return nil, cqs.NewInvalidCommandError(ExtractFileCommandName, command.Name())
	}

	extraction, err := snapshot.NewFileExtraction(cmd.SnapshotID, cmd.Path, cmd.Directory)
	if err != nil {
		return nil, err
	}

	location, err := h.snapshots.ExtractFile(ctx, extraction)
	if err != nil {
		return nil, err
	}

	if err := extraction.Extracted(location); err != nil {
		return nil, err
	}

	return extraction.Events(), nil
}

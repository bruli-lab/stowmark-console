package app

import (
	"context"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

const CreateSnapshotCommandName = "create-snapshot"

type CreateSnapshotCommand struct {
	Source     string
	Exclusions snapshot.Exclusions
}

func (CreateSnapshotCommand) Name() string {
	return CreateSnapshotCommandName
}

type CreateSnapshot struct {
	snapshots snapshot.Repository
}

func NewCreateSnapshot(snapshots snapshot.Repository) CreateSnapshot {
	return CreateSnapshot{snapshots: snapshots}
}

func (h CreateSnapshot) Handle(ctx context.Context, command cqs.Command) ([]event.Event, error) {
	cmd, ok := command.(CreateSnapshotCommand)
	if !ok {
		return nil, cqs.NewInvalidCommandError(CreateSnapshotCommandName, command.Name())
	}

	creation, err := snapshot.NewCreation(cmd.Source, cmd.Exclusions)
	if err != nil {
		return nil, err
	}

	created, skipped, err := h.snapshots.Create(ctx, creation)
	if err != nil {
		return nil, err
	}

	if err := creation.Created(created, skipped); err != nil {
		return nil, err
	}

	return creation.Events(), nil
}

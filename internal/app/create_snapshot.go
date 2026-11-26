package app

import (
	"context"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

const CreateSnapshotCommandName = "create-snapshot"

type CreateSnapshotCommand struct {
	Source string
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

	if cmd.Source == "" {
		return nil, snapshot.ErrSourceRequired
	}

	created, err := h.snapshots.Create(ctx, cmd.Source)
	if err != nil {
		return nil, err
	}

	return []event.Event{snapshot.NewCreated(created)}, nil
}

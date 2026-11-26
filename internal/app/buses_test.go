package app_test

import (
	"context"
	"testing"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/stowmark-console/internal/app"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
	"github.com/bruli-lab/stowmark-console/internal/mocks"
	"github.com/stretchr/testify/require"
)

func failingBuses() app.Buses {
	catalog := &mocks.CatalogMock{
		ProfilesFunc: func() []repository.Profile { return nil },
		OpenFunc:     func(context.Context, string) (repository.Profile, error) { return repository.Profile{}, errBackend },
	}
	repositories := &mocks.RepositoryMock{
		InitializedFunc: func(context.Context) (bool, error) { return false, errBackend },
		InitializeFunc:  func(context.Context, repository.InitOptions) error { return errBackend },
	}
	snapshots := &mocks.SnapshotRepositoryMock{
		ListFunc:     func(context.Context) ([]snapshot.Snapshot, error) { return nil, errBackend },
		ManifestFunc: func(context.Context, string) (snapshot.Manifest, error) { return snapshot.Manifest{}, errBackend },
		FileContentFunc: func(context.Context, string, string, int64) ([]byte, error) {
			return nil, errBackend
		},
		CreateFunc:  func(context.Context, string) (snapshot.Snapshot, error) { return snapshot.Snapshot{}, errBackend },
		VerifyFunc:  func(context.Context, string) ([]snapshot.Failure, error) { return nil, errBackend },
		RestoreFunc: func(context.Context, string, string) ([]snapshot.Failure, error) { return nil, errBackend },
		ExtractFileFunc: func(context.Context, *snapshot.FileExtraction) (string, error) {
			return "", errBackend
		},
	}

	return app.NewBuses(catalog, repositories, snapshots)
}

func TestNewBuses(t *testing.T) {
	t.Parallel()

	buses := failingBuses()

	commands := []cqs.Command{
		app.OpenRepositoryCommand{RepositoryName: "nas"},
		app.InitRepositoryCommand{},
		app.CreateSnapshotCommand{Source: "/data"},
		app.VerifySnapshotCommand{SnapshotID: "snap"},
		app.RestoreSnapshotCommand{SnapshotID: "snap", Destination: "/restore"},
		app.ExtractFileCommand{SnapshotID: "snap", Path: "/data/a.txt", Directory: "/tmp/open"},
	}
	for _, command := range commands {
		t.Run(`Given the buses
		When the `+command.Name()+` command is dispatched
		Then it reaches its handler`, func(t *testing.T) {
			t.Parallel()

			_, err := buses.Commands.Handle(t.Context(), command)

			require.ErrorIs(t, err, errBackend)
		})
	}

	queries := []cqs.Query{
		app.RepositoryStatusQuery{},
		app.ListSnapshotsQuery{},
		app.GetManifestQuery{SnapshotID: "snap"},
		app.GetFileContentQuery{SnapshotID: "snap", Path: "/data/a.txt", MaxSize: 1},
	}
	for _, query := range queries {
		t.Run(`Given the buses
		When the `+query.Name()+` query is asked
		Then it reaches its handler`, func(t *testing.T) {
			t.Parallel()

			_, err := buses.Queries.Handle(t.Context(), query)

			require.ErrorIs(t, err, errBackend)
		})
	}

	t.Run(`Given the buses
	When the list repositories query is asked
	Then it reaches its handler`, func(t *testing.T) {
		t.Parallel()

		_, err := buses.Queries.Handle(t.Context(), app.ListRepositoriesQuery{})

		require.ErrorIs(t, err, app.ErrNoRepositories)
	})
}

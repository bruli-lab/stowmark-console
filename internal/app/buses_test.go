package app_test

import (
	"context"
	"testing"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/stowmark-console/internal/app"
	"github.com/bruli-lab/stowmark-console/internal/domain/encryption"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/domain/retention"
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
		StatusFunc:     func(context.Context) (repository.Status, error) { return repository.StatusUninitialized, errBackend },
		InitializeFunc: func(context.Context, repository.InitOptions) error { return errBackend },
		UnlockFunc:     func(context.Context, bool) (repository.LockReport, error) { return repository.LockReport{}, errBackend },
	}
	snapshots := &mocks.SnapshotRepositoryMock{
		ListFunc:     func(context.Context) ([]snapshot.Snapshot, error) { return nil, errBackend },
		ManifestFunc: func(context.Context, string) (snapshot.Manifest, error) { return snapshot.Manifest{}, errBackend },
		FileContentFunc: func(context.Context, string, string, int64) ([]byte, error) {
			return nil, errBackend
		},
		CreateFunc: func(context.Context, *snapshot.Creation) (snapshot.Snapshot, error) {
			return snapshot.Snapshot{}, errBackend
		},
		VerifyFunc:  func(context.Context, *snapshot.Verification) ([]snapshot.Failure, error) { return nil, errBackend },
		RestoreFunc: func(context.Context, *snapshot.Restoration) ([]snapshot.Failure, error) { return nil, errBackend },
		ExtractFileFunc: func(context.Context, *snapshot.FileExtraction) (string, error) {
			return "", errBackend
		},
	}

	keys := &mocks.KeysMock{
		GenerateFunc: func(context.Context, *encryption.KeyGeneration) (encryption.KeyPair, error) {
			return encryption.KeyPair{}, errBackend
		},
		RewrapFunc: func(context.Context, *encryption.Rewrap) error { return errBackend },
		RotateFunc: func(context.Context, *encryption.Rotation) error { return errBackend },
	}

	retentionRepository := &mocks.RetentionRepositoryMock{
		PlanForgetFunc: func(context.Context, retention.Policy) (retention.Plan, error) { return nil, errBackend },
		ForgetFunc: func(context.Context, *retention.Forgetting) (retention.ForgetResult, error) {
			return retention.ForgetResult{}, errBackend
		},
		PlanPruneFunc: func(context.Context) (retention.PrunePlan, error) { return retention.PrunePlan{}, errBackend },
		PruneFunc: func(context.Context, *retention.Pruning) (retention.PruneResult, error) {
			return retention.PruneResult{}, errBackend
		},
	}

	return app.NewBuses(&app.Ports{
		Catalog:      catalog,
		Repositories: repositories,
		Snapshots:    snapshots,
		Keys:         keys,
		Retention:    retentionRepository,
	})
}

func TestNewBuses(t *testing.T) {
	t.Parallel()

	buses := failingBuses()

	commands := []cqs.Command{
		app.OpenRepositoryCommand{RepositoryName: "nas"},
		app.InitRepositoryCommand{},
		app.UnlockRepositoryCommand{All: true},
		app.ForgetSnapshotsCommand{Rules: retention.Rules{Last: 1}},
		app.PruneRepositoryCommand{},
		app.CreateSnapshotCommand{Source: "/data"},
		app.VerifySnapshotCommand{SnapshotID: "snap"},
		app.RestoreSnapshotCommand{SnapshotID: "snap", Destination: "/restore"},
		app.ExtractFileCommand{SnapshotID: "snap", Path: "/data/a.txt", Directory: "/tmp/open"},
		app.GenerateKeysCommand{Folder: "/keys"},
		app.RewrapKeyCommand{OldPrivateKey: "/keys/old-private.pem", NewPublicKey: "/keys/new-public.pem"},
		app.RotateKeyCommand{PrivateKey: "/keys/private.pem", PublicKey: "/keys/public.pem"},
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
		app.PlanForgetQuery{Rules: retention.Rules{Last: 1}},
		app.PlanPruneQuery{},
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

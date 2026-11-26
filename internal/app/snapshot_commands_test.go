package app_test

import (
	"context"
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/app"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
	"github.com/bruli-lab/stowmark-console/internal/fixtures"
	"github.com/bruli-lab/stowmark-console/internal/mocks"
	"github.com/stretchr/testify/require"
)

func TestCreateSnapshot(t *testing.T) {
	t.Parallel()

	t.Run(`Given an empty source
	When a snapshot is created
	Then a source required error is returned`, func(t *testing.T) {
		t.Parallel()
		repo := &mocks.SnapshotRepositoryMock{}

		_, err := app.NewCreateSnapshot(repo).Handle(t.Context(), app.CreateSnapshotCommand{})

		require.ErrorIs(t, err, snapshot.ErrSourceRequired)
		require.Empty(t, repo.CreateCalls())
	})

	t.Run(`Given a source directory
	When a snapshot is created
	Then a created event with the new snapshot is returned`, func(t *testing.T) {
		t.Parallel()
		created := fixtures.Snapshot().WithSource("/data").Build()
		repo := &mocks.SnapshotRepositoryMock{
			CreateFunc: func(context.Context, string) (snapshot.Snapshot, error) { return created, nil },
		}

		event, err := app.NewCreateSnapshot(repo).Handle(t.Context(), app.CreateSnapshotCommand{Source: "/data"})

		require.NoError(t, err)
		require.Equal(t, created, event.Snapshot)
		require.Equal(t, "/data", repo.CreateCalls()[0].Source)
	})

	t.Run(`Given a repository that fails
	When a snapshot is created
	Then the error is returned`, func(t *testing.T) {
		t.Parallel()
		repo := &mocks.SnapshotRepositoryMock{
			CreateFunc: func(context.Context, string) (snapshot.Snapshot, error) { return snapshot.Snapshot{}, errBackend },
		}

		_, err := app.NewCreateSnapshot(repo).Handle(t.Context(), app.CreateSnapshotCommand{Source: "/data"})

		require.ErrorIs(t, err, errBackend)
	})
}

func TestVerifySnapshot(t *testing.T) {
	t.Parallel()

	t.Run(`Given an empty snapshot id
	When it is verified
	Then a snapshot id required error is returned`, func(t *testing.T) {
		t.Parallel()
		repo := &mocks.SnapshotRepositoryMock{}

		_, err := app.NewVerifySnapshot(repo).Handle(t.Context(), app.VerifySnapshotCommand{})

		require.ErrorIs(t, err, snapshot.ErrSnapshotIDRequired)
	})

	t.Run(`Given a snapshot with corrupted files
	When it is verified
	Then the verified event contains the failures`, func(t *testing.T) {
		t.Parallel()
		failures := []snapshot.Failure{{Path: "/data/a.txt", Reason: "hash mismatch"}}
		repo := &mocks.SnapshotRepositoryMock{
			VerifyFunc: func(context.Context, string) ([]snapshot.Failure, error) { return failures, nil },
		}

		event, err := app.NewVerifySnapshot(repo).Handle(t.Context(), app.VerifySnapshotCommand{SnapshotID: "snap"})

		require.NoError(t, err)
		require.Equal(t, "snap", event.SnapshotID)
		require.Equal(t, failures, event.Failures)
		require.False(t, event.Succeeded())
	})
}

func TestRestoreSnapshot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		command     app.RestoreSnapshotCommand
		expectedErr error
	}{
		{name: "an empty snapshot id", command: app.RestoreSnapshotCommand{Destination: "/restore"}, expectedErr: snapshot.ErrSnapshotIDRequired},
		{name: "no destination", command: app.RestoreSnapshotCommand{SnapshotID: "snap"}, expectedErr: snapshot.ErrDestinationRequired},
		{name: "a relative destination", command: app.RestoreSnapshotCommand{SnapshotID: "snap", Destination: "restore"}, expectedErr: snapshot.ErrDestinationNotAbsolute},
	}

	for _, tt := range tests {
		t.Run(`Given a restore command with `+tt.name+`
		When it is handled
		Then it is rejected without touching the repository`, func(t *testing.T) {
			t.Parallel()
			repo := &mocks.SnapshotRepositoryMock{}

			_, err := app.NewRestoreSnapshot(repo).Handle(t.Context(), tt.command)

			require.ErrorIs(t, err, tt.expectedErr)
			require.Empty(t, repo.RestoreCalls())
		})
	}

	t.Run(`Given a valid restore command
	When it is handled
	Then the snapshot is restored to the destination and a restored event is returned`, func(t *testing.T) {
		t.Parallel()
		repo := &mocks.SnapshotRepositoryMock{
			RestoreFunc: func(context.Context, string, string) ([]snapshot.Failure, error) { return nil, nil },
		}

		event, err := app.NewRestoreSnapshot(repo).Handle(t.Context(), app.RestoreSnapshotCommand{SnapshotID: "snap", Destination: "/restore"})

		require.NoError(t, err)
		require.True(t, event.Succeeded())
		require.Equal(t, "/restore", event.Destination)
		require.Equal(t, "/restore", repo.RestoreCalls()[0].Destination)
	})
}

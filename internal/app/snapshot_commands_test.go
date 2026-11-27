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

	t.Run(`Given a command that is not a create snapshot command
	When it is handled
	Then an invalid command error is returned`, func(t *testing.T) {
		t.Parallel()

		requireInvalidCommand(t, app.NewCreateSnapshot(&mocks.SnapshotRepositoryMock{}))
	})

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
		skipped := []snapshot.SkippedFile{{Path: "app.sock", Kind: "socket"}}
		exclusions := snapshot.NewExclusions([]string{"*.tmp"}, true)
		repo := &mocks.SnapshotRepositoryMock{
			CreateFunc: func(context.Context, *snapshot.Creation) (snapshot.Snapshot, []snapshot.SkippedFile, error) {
				return created, skipped, nil
			},
		}

		events, err := app.NewCreateSnapshot(repo).Handle(t.Context(), app.CreateSnapshotCommand{Source: "/data", Exclusions: exclusions})

		require.NoError(t, err)
		event := singleEvent[snapshot.Created](t, events, snapshot.CreatedEventName)
		require.Equal(t, created, event.Snapshot)
		require.Equal(t, skipped, event.Skipped)
		require.Equal(t, exclusions, repo.CreateCalls()[0].Creation.Exclusions())
		require.Equal(t, created.ID(), event.AggregateRootID())
		require.Equal(t, "/data", repo.CreateCalls()[0].Creation.Source())
	})

	t.Run(`Given a repository that fails
	When a snapshot is created
	Then the error is returned`, func(t *testing.T) {
		t.Parallel()
		repo := &mocks.SnapshotRepositoryMock{
			CreateFunc: func(context.Context, *snapshot.Creation) (snapshot.Snapshot, []snapshot.SkippedFile, error) {
				return snapshot.Snapshot{}, nil, errBackend
			},
		}

		_, err := app.NewCreateSnapshot(repo).Handle(t.Context(), app.CreateSnapshotCommand{Source: "/data"})

		require.ErrorIs(t, err, errBackend)
	})
}

func TestVerifySnapshot(t *testing.T) {
	t.Parallel()

	t.Run(`Given a command that is not a verify snapshot command
	When it is handled
	Then an invalid command error is returned`, func(t *testing.T) {
		t.Parallel()

		requireInvalidCommand(t, app.NewVerifySnapshot(&mocks.SnapshotRepositoryMock{}))
	})

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
			VerifyFunc: func(context.Context, *snapshot.Verification) ([]snapshot.Failure, error) { return failures, nil },
		}

		events, err := app.NewVerifySnapshot(repo).Handle(t.Context(), app.VerifySnapshotCommand{SnapshotID: "snap"})

		require.NoError(t, err)
		event := singleEvent[snapshot.Verified](t, events, snapshot.VerifiedEventName)
		require.Equal(t, "snap", event.SnapshotID)
		require.Equal(t, failures, event.Failures)
		require.False(t, event.Succeeded())
	})
}

func TestRestoreSnapshot(t *testing.T) {
	t.Parallel()

	t.Run(`Given a command that is not a restore snapshot command
	When it is handled
	Then an invalid command error is returned`, func(t *testing.T) {
		t.Parallel()

		requireInvalidCommand(t, app.NewRestoreSnapshot(&mocks.SnapshotRepositoryMock{}))
	})

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
			RestoreFunc: func(context.Context, *snapshot.Restoration) ([]snapshot.Failure, error) { return nil, nil },
		}

		events, err := app.NewRestoreSnapshot(repo).Handle(t.Context(), app.RestoreSnapshotCommand{SnapshotID: "snap", Destination: "/restore"})

		require.NoError(t, err)
		event := singleEvent[snapshot.Restored](t, events, snapshot.RestoredEventName)
		require.True(t, event.Succeeded())
		require.Equal(t, "/restore", event.Destination)
		require.Equal(t, "/restore", repo.RestoreCalls()[0].Restoration.Destination())
	})
}

func TestExtractFile(t *testing.T) {
	t.Parallel()

	t.Run(`Given a command that is not an extract file command
	When it is handled
	Then an invalid command error is returned`, func(t *testing.T) {
		t.Parallel()

		requireInvalidCommand(t, app.NewExtractFile(&mocks.SnapshotRepositoryMock{}))
	})

	t.Run(`Given a command with invalid values
	When the file is extracted
	Then the file extraction validation error is returned without touching the repository`, func(t *testing.T) {
		t.Parallel()
		snapshots := &mocks.SnapshotRepositoryMock{}

		_, err := app.NewExtractFile(snapshots).Handle(t.Context(), app.ExtractFileCommand{SnapshotID: "id", Path: "/data/a.txt", Directory: "open"})

		require.ErrorIs(t, err, snapshot.ErrExtractionDirectoryNotAbsolute)
		require.Empty(t, snapshots.ExtractFileCalls())
	})

	t.Run(`Given a repository that fails to extract
	When the file is extracted
	Then the error is returned without events`, func(t *testing.T) {
		t.Parallel()
		snapshots := &mocks.SnapshotRepositoryMock{
			ExtractFileFunc: func(context.Context, *snapshot.FileExtraction) (string, error) { return "", errBackend },
		}

		events, err := app.NewExtractFile(snapshots).Handle(t.Context(), app.ExtractFileCommand{SnapshotID: "id", Path: "/data/a.txt", Directory: "/tmp/open"})

		require.ErrorIs(t, err, errBackend)
		require.Empty(t, events)
	})

	t.Run(`Given a valid command
	When the file is extracted
	Then the file extraction is passed to the repository and its file extracted event is returned`, func(t *testing.T) {
		t.Parallel()
		snapshots := &mocks.SnapshotRepositoryMock{
			ExtractFileFunc: func(context.Context, *snapshot.FileExtraction) (string, error) {
				return "/tmp/open/extract-1/a.txt", nil
			},
		}

		events, err := app.NewExtractFile(snapshots).Handle(t.Context(), app.ExtractFileCommand{
			SnapshotID: "id",
			Path:       "/data/a.txt",
			Directory:  "/tmp/open",
		})

		require.NoError(t, err)
		event := singleEvent[snapshot.FileExtracted](t, events, snapshot.FileExtractedEventName)
		require.Equal(t, "id", event.SnapshotID)
		require.Equal(t, "/data/a.txt", event.Path)
		require.Equal(t, "/tmp/open/extract-1/a.txt", event.Location)
		extraction := snapshots.ExtractFileCalls()[0].Extraction
		require.Equal(t, "/tmp/open", extraction.Directory())
		require.Equal(t, "/data/a.txt", extraction.Path())
	})
}

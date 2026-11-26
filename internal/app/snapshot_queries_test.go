package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/bruli-lab/stowmark-console/internal/app"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
	"github.com/bruli-lab/stowmark-console/internal/fixtures"
	"github.com/bruli-lab/stowmark-console/internal/mocks"
	"github.com/stretchr/testify/require"
)

func TestListSnapshots(t *testing.T) {
	t.Parallel()

	t.Run(`Given a query that is not a list snapshots query
	When it is handled
	Then an invalid query error is returned`, func(t *testing.T) {
		t.Parallel()

		requireInvalidQuery(t, app.NewListSnapshots(&mocks.SnapshotRepositoryMock{}))
	})

	t.Run(`Given a repository with snapshots
	When the snapshots are listed
	Then they are returned newest first`, func(t *testing.T) {
		t.Parallel()
		base := time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)
		older := fixtures.Snapshot().WithID("older").WithCreatedAt(base).Build()
		newer := fixtures.Snapshot().WithID("newer").WithCreatedAt(base.Add(time.Hour)).Build()
		repo := &mocks.SnapshotRepositoryMock{
			ListFunc: func(context.Context) ([]snapshot.Snapshot, error) {
				return []snapshot.Snapshot{older, newer}, nil
			},
		}

		snapshots, err := app.NewListSnapshots(repo).Handle(t.Context(), app.ListSnapshotsQuery{})

		require.NoError(t, err)
		require.Equal(t, []snapshot.Snapshot{newer, older}, snapshots)
	})

	t.Run(`Given a repository that fails
	When the snapshots are listed
	Then the error is returned`, func(t *testing.T) {
		t.Parallel()
		repo := &mocks.SnapshotRepositoryMock{
			ListFunc: func(context.Context) ([]snapshot.Snapshot, error) { return nil, errBackend },
		}

		_, err := app.NewListSnapshots(repo).Handle(t.Context(), app.ListSnapshotsQuery{})

		require.ErrorIs(t, err, errBackend)
	})
}

func TestGetManifest(t *testing.T) {
	t.Parallel()

	t.Run(`Given a query that is not a get manifest query
	When it is handled
	Then an invalid query error is returned`, func(t *testing.T) {
		t.Parallel()

		requireInvalidQuery(t, app.NewGetManifest(&mocks.SnapshotRepositoryMock{}))
	})

	t.Run(`Given an empty snapshot id
	When the manifest is requested
	Then a snapshot id required error is returned without calling the repository`, func(t *testing.T) {
		t.Parallel()
		repo := &mocks.SnapshotRepositoryMock{}

		_, err := app.NewGetManifest(repo).Handle(t.Context(), app.GetManifestQuery{})

		require.ErrorIs(t, err, snapshot.ErrSnapshotIDRequired)
		require.Empty(t, repo.ManifestCalls())
	})

	t.Run(`Given an existing snapshot
	When its manifest is requested
	Then the manifest is returned`, func(t *testing.T) {
		t.Parallel()
		expected := fixtures.Manifest().WithSnapshotID("snap").Build()
		repo := &mocks.SnapshotRepositoryMock{
			ManifestFunc: func(context.Context, string) (snapshot.Manifest, error) { return expected, nil },
		}

		manifest, err := app.NewGetManifest(repo).Handle(t.Context(), app.GetManifestQuery{SnapshotID: "snap"})

		require.NoError(t, err)
		require.Equal(t, expected, manifest)
		require.Equal(t, "snap", repo.ManifestCalls()[0].SnapshotID)
	})
}

func TestGetFileContent(t *testing.T) {
	t.Parallel()

	t.Run(`Given a query that is not a get file content query
	When it is handled
	Then an invalid query error is returned`, func(t *testing.T) {
		t.Parallel()

		requireInvalidQuery(t, app.NewGetFileContent(&mocks.SnapshotRepositoryMock{}))
	})

	tests := []struct {
		name        string
		query       app.GetFileContentQuery
		expectedErr error
	}{
		{name: "an empty snapshot id, then it is rejected", query: app.GetFileContentQuery{Path: "/a", MaxSize: 10}, expectedErr: snapshot.ErrSnapshotIDRequired},
		{name: "a non positive max size, then it is rejected", query: app.GetFileContentQuery{SnapshotID: "snap", Path: "/a"}, expectedErr: app.ErrInvalidMaxSize},
	}

	for _, tt := range tests {
		t.Run(`Given a file content query with `+tt.name, func(t *testing.T) {
			t.Parallel()
			repo := &mocks.SnapshotRepositoryMock{}

			_, err := app.NewGetFileContent(repo).Handle(t.Context(), tt.query)

			require.ErrorIs(t, err, tt.expectedErr)
			require.Empty(t, repo.FileContentCalls())
		})
	}

	t.Run(`Given a valid file content query
	When it is handled
	Then the content is read with the requested limit`, func(t *testing.T) {
		t.Parallel()
		repo := &mocks.SnapshotRepositoryMock{
			FileContentFunc: func(context.Context, string, string, int64) ([]byte, error) { return []byte("hello"), nil },
		}

		content, err := app.NewGetFileContent(repo).Handle(t.Context(), app.GetFileContentQuery{SnapshotID: "snap", Path: "/a.txt", MaxSize: 64})

		require.NoError(t, err)
		require.Equal(t, []byte("hello"), content)
		call := repo.FileContentCalls()[0]
		require.Equal(t, "snap", call.SnapshotID)
		require.Equal(t, "/a.txt", call.Path)
		require.Equal(t, int64(64), call.MaxSize)
	})
}

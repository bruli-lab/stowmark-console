package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/fixtures"
	"github.com/stretchr/testify/require"
)

func TestReadEntries(t *testing.T) {
	t.Parallel()

	t.Run(`Given a folder repository with entries
	When the entries of a path are read
	Then the entries of that path are returned`, func(t *testing.T) {
		t.Parallel()
		expected := []repository.Entry{
			fixtures.Entry().WithName("snapshots").AsDirectory().Build(),
			fixtures.Entry().WithName("config.json").Build(),
		}
		repo := &FolderRepositoryMock{
			ReadEntriesFunc: func(_ context.Context, _ string) ([]repository.Entry, error) {
				return expected, nil
			},
		}

		entries, err := repository.NewRead(repo).ReadEntries(t.Context(), "/backups")

		require.NoError(t, err)
		require.Equal(t, expected, entries)
		require.Len(t, repo.ReadEntriesCalls(), 1)
		require.Equal(t, "/backups", repo.ReadEntriesCalls()[0].Path)
	})

	t.Run(`Given a folder repository that fails
	When the entries are read
	Then the error is returned`, func(t *testing.T) {
		t.Parallel()
		failure := errors.New("unreachable")
		repo := &FolderRepositoryMock{
			ReadEntriesFunc: func(_ context.Context, _ string) ([]repository.Entry, error) {
				return nil, failure
			},
		}

		_, err := repository.NewRead(repo).ReadEntries(t.Context(), "/backups")

		require.ErrorIs(t, err, failure)
	})
}

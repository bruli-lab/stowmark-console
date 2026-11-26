package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/infra/store"
	"github.com/stretchr/testify/require"
)

func writeSource(t *testing.T) string {
	t.Helper()
	source := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(source, "notes"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(source, "README.txt"), []byte("hello stowmark"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(source, "notes", "todo.md"), []byte("- verify\n- restore\n"), 0o600))
	return source
}

func TestStoreWithLocalRepository(t *testing.T) {
	t.Setenv("STOWMARK_LOG_LEVEL", "error")

	repositoryPath := filepath.Join(t.TempDir(), "repository")
	source := writeSource(t)

	session, err := store.Open(t.Context(), repositoryPath, nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, session.Close()) })

	repositories := store.NewRepositories(session)
	snapshots := store.NewSnapshots(session)

	var snapshotID string

	t.Run(`Given a path without repository
	When its status is checked
	Then it is not initialized`, func(t *testing.T) {
		initialized, err := repositories.Initialized(t.Context())

		require.NoError(t, err)
		require.False(t, initialized)
	})

	t.Run(`Given a path without repository
	When it is initialized with zstd compression
	Then it becomes initialized`, func(t *testing.T) {
		level := 3
		err := repositories.Initialize(t.Context(), repository.NewInitOptions("zstd", &level, nil))
		require.NoError(t, err)

		initialized, err := repositories.Initialized(t.Context())

		require.NoError(t, err)
		require.True(t, initialized)
	})

	t.Run(`Given an initialized repository
	When a snapshot of a directory is created
	Then it is listed with its source, files and size`, func(t *testing.T) {
		created, err := snapshots.Create(t.Context(), source)
		require.NoError(t, err)
		snapshotID = created.ID()

		listed, err := snapshots.List(t.Context())

		require.NoError(t, err)
		require.Len(t, listed, 1)
		require.Equal(t, snapshotID, listed[0].ID())
		require.Equal(t, listed[0].FileCount(), created.FileCount())
		require.Equal(t, listed[0].TotalSize(), created.TotalSize())
		require.Equal(t, source, listed[0].Source())
		require.Equal(t, 2, listed[0].FileCount())
		require.False(t, created.CreatedAt().IsZero())
	})

	t.Run(`Given a source that is already stored
	When a second snapshot of it is created
	Then the created snapshot reports every file of the source, not only the new ones`, func(t *testing.T) {
		created, err := snapshots.Create(t.Context(), source)

		require.NoError(t, err)
		require.Equal(t, 2, created.FileCount())
		require.Equal(t, int64(len("hello stowmark")+len("- verify\n- restore\n")), created.TotalSize())
	})

	t.Run(`Given a snapshot
	When its manifest is read
	Then its files can be browsed by directory`, func(t *testing.T) {
		manifest, err := snapshots.Manifest(t.Context(), snapshotID)
		require.NoError(t, err)

		entries := manifest.Entries("")

		require.Len(t, entries, 2)
		require.Equal(t, "notes", entries[0].Name())
		require.True(t, entries[0].IsDir())
		require.Equal(t, "README.txt", entries[1].Name())
	})

	t.Run(`Given a snapshot
	When the content of one of its files is read with a limit
	Then only up to the limit is returned`, func(t *testing.T) {
		content, err := snapshots.FileContent(t.Context(), snapshotID, filepath.Join(source, "README.txt"), 5)

		require.NoError(t, err)
		require.Equal(t, "hello", string(content))
	})

	t.Run(`Given a snapshot
	When it is verified
	Then there are no failures`, func(t *testing.T) {
		failures, err := snapshots.Verify(t.Context(), snapshotID)

		require.NoError(t, err)
		require.Empty(t, failures)
	})

	t.Run(`Given a snapshot
	When it is restored to another directory
	Then the files are restored there and the source is untouched`, func(t *testing.T) {
		destination := filepath.Join(t.TempDir(), "restore")

		failures, err := snapshots.Restore(t.Context(), snapshotID, destination)

		require.NoError(t, err)
		require.Empty(t, failures)
		restored, err := os.ReadFile(filepath.Join(destination, "notes", "todo.md"))
		require.NoError(t, err)
		require.Equal(t, "- verify\n- restore\n", string(restored))
		original, err := os.ReadFile(filepath.Join(source, "README.txt"))
		require.NoError(t, err)
		require.Equal(t, "hello stowmark", string(original))
	})
}

package store_test

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
	"github.com/bruli-lab/stowmark-console/internal/infra/store"
	"github.com/stretchr/testify/require"
)

func filePaths(manifest snapshot.Manifest) []string {
	files := manifest.Files()
	paths := make([]string, len(files))
	for i := range files {
		paths[i] = files[i].Path()
	}
	return paths
}

func TestStoreCreateOptions(t *testing.T) {
	t.Setenv("STOWMARK_LOG_LEVEL", "error")
	session, err := store.Open(t.Context(), filepath.Join(t.TempDir(), "repository"), nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, session.Close()) })
	require.NoError(t, store.NewRepositories(session).Initialize(t.Context(), repository.NewInitOptions("none", nil, nil)))
	snapshots := store.NewSnapshots(session)

	source := writeSource(t)
	require.NoError(t, os.WriteFile(filepath.Join(source, "build.tmp"), []byte("tmp"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(source, "notes", "draft.log"), []byte("log"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(source, snapshot.IgnoreFileName), []byte("*.log\n"), 0o600))
	require.NoError(t, syscall.Mkfifo(filepath.Join(source, "events.pipe"), 0o600))

	create := func(t *testing.T, exclusions snapshot.Exclusions) ([]string, []snapshot.SkippedFile) {
		t.Helper()
		created, err := snapshot.NewCreation(source, exclusions)
		require.NoError(t, err)
		result, skipped, err := snapshots.Create(t.Context(), created)
		require.NoError(t, err)
		manifest, err := snapshots.Manifest(t.Context(), result.ID())
		require.NoError(t, err)
		return filePaths(manifest), skipped
	}

	t.Run(`Given an origin with a .stowmarkignore file and a named pipe
	When a snapshot is created without options
	Then the ignore file applies and the pipe is reported as skipped`, func(t *testing.T) {
		paths, skipped := create(t, snapshot.Exclusions{})

		require.Contains(t, paths, filepath.Join(source, "build.tmp"))
		require.NotContains(t, paths, filepath.Join(source, "notes", "draft.log"))
		require.Equal(t, []snapshot.SkippedFile{{Path: "events.pipe", Kind: "named pipe"}}, skipped)
	})

	t.Run(`Given exclude patterns and the ignore files skipped
	When a snapshot is created
	Then the patterns apply and the ignore file does not`, func(t *testing.T) {
		paths, _ := create(t, snapshot.NewExclusions([]string{"*.tmp"}, true))

		require.NotContains(t, paths, filepath.Join(source, "build.tmp"))
		require.Contains(t, paths, filepath.Join(source, "notes", "draft.log"))
		require.Contains(t, paths, filepath.Join(source, "README.txt"))
	})

	t.Run(`Given an invalid exclude pattern
	When a snapshot is created
	Then it fails with the invalid pattern error`, func(t *testing.T) {
		created, err := snapshot.NewCreation(source, snapshot.NewExclusions([]string{"a//b"}, false))
		require.NoError(t, err)

		_, _, err = snapshots.Create(t.Context(), created)

		require.ErrorIs(t, err, snapshot.ErrInvalidExcludePattern)
		require.ErrorContains(t, err, "a//b")
	})
}

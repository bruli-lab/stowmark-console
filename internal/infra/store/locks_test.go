package store_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/infra/store"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func writeLock(t *testing.T, repositoryPath, kind, hostname string, createdAt time.Time) {
	t.Helper()
	id := uuid.NewString()
	data, err := json.Marshal(map[string]any{
		"id":         id,
		"kind":       kind,
		"hostname":   hostname,
		"pid":        4242,
		"created_at": createdAt.UTC(),
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(repositoryPath, "locks", id+".json"), data, 0o600))
}

func TestStoreLocks(t *testing.T) {
	t.Setenv("STOWMARK_LOG_LEVEL", "error")
	source := writeSource(t)
	repositoryPath := filepath.Join(t.TempDir(), "repository")

	session, err := store.Open(t.Context(), repositoryPath, nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, session.Close()) })
	repositories := store.NewRepositories(session)
	snapshots := store.NewSnapshots(session)
	require.NoError(t, repositories.Initialize(t.Context(), repository.NewInitOptions("none", nil, nil)))
	require.NoError(t, os.MkdirAll(filepath.Join(repositoryPath, "locks"), 0o700))
	writeLock(t, repositoryPath, "exclusive", "backup-host", time.Now())
	writeLock(t, repositoryPath, "exclusive", "crashed-host", time.Now().Add(-2*time.Hour))

	t.Run(`Given a repository with an active and a stale lock
	When the stale locks are removed
	Then only the stale one is removed and the active one is reported`, func(t *testing.T) {
		report, err := repositories.Unlock(t.Context(), false)

		require.NoError(t, err)
		require.Equal(t, repository.LockReport{Removed: 1, Active: 1}, report)
	})

	t.Run(`Given a repository with an active exclusive lock
	When a snapshot is created
	Then it fails with a locked error that names the holder`, func(t *testing.T) {
		_, err := snapshots.Create(t.Context(), creation(t, source))

		require.ErrorIs(t, err, repository.ErrLocked)
		require.ErrorContains(t, err, "backup-host (pid 4242)")
	})

	t.Run(`Given a repository with an active lock
	When every lock is removed
	Then it is removed and snapshots can be created again`, func(t *testing.T) {
		report, err := repositories.Unlock(t.Context(), true)
		require.NoError(t, err)
		require.Equal(t, repository.LockReport{Removed: 1, Active: 0}, report)

		_, err = snapshots.Create(t.Context(), creation(t, source))

		require.NoError(t, err)
	})
}

func TestStoreUnlockWithoutRepository(t *testing.T) {
	t.Parallel()

	_, err := store.NewRepositories(store.NewSession()).Unlock(t.Context(), false)

	require.ErrorIs(t, err, store.ErrNotConnected)
}

package store_test

import (
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/domain/progress"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/infra/store"
	"github.com/stretchr/testify/require"
)

type progressRecorder struct {
	mu      sync.Mutex
	updates []progress.Update
}

func (r *progressRecorder) Report(update progress.Update) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.updates = append(r.updates, update)
}

func (r *progressRecorder) take() []progress.Update {
	r.mu.Lock()
	defer r.mu.Unlock()
	updates := r.updates
	r.updates = nil
	return updates
}

func TestStoreProgress(t *testing.T) {
	t.Setenv("STOWMARK_LOG_LEVEL", "error")
	source := writeSource(t)
	recorder := &progressRecorder{}

	session, err := store.Open(t.Context(), filepath.Join(t.TempDir(), "repository"), nil, store.WithProgress(recorder))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, session.Close()) })
	require.NoError(t, store.NewRepositories(session).Initialize(t.Context(), repository.NewInitOptions("none", nil, nil)))
	snapshots := store.NewSnapshots(session)

	var snapshotID string
	entries := len([]string{"README.txt", "notes", "notes/todo.md"})
	files := len([]string{"README.txt", "notes/todo.md"})

	t.Run(`Given a session with a progress listener
	When a snapshot is created
	Then the listener follows the backup from its totals to the finish`, func(t *testing.T) {
		created, _, err := snapshots.Create(t.Context(), creation(t, source))
		require.NoError(t, err)
		snapshotID = created.ID()

		requireStage(t, recorder.take(), progress.StageBackingUp, entries, created.TotalSize())
	})

	t.Run(`Given a session with a progress listener
	When a snapshot is verified
	Then the listener follows the verification`, func(t *testing.T) {
		_, err := snapshots.Verify(t.Context(), verification(t, snapshotID))
		require.NoError(t, err)

		requireStage(t, recorder.take(), progress.StageVerifying, entries, -1)
	})

	t.Run(`Given a session with a progress listener
	When a snapshot is restored
	Then the listener follows the restore`, func(t *testing.T) {
		_, err := snapshots.Restore(t.Context(), restoration(t, snapshotID, t.TempDir()))
		require.NoError(t, err)

		requireStage(t, recorder.take(), progress.StageRestoring, files, -1)
	})
}

func requireStage(t *testing.T, updates []progress.Update, stage progress.Stage, items int, bytes int64) {
	t.Helper()

	start := slices.IndexFunc(updates, func(update progress.Update) bool { return update.Stage == stage })
	require.GreaterOrEqual(t, start, 0, "stage %q not reported in %v", stage, updates)
	require.Equal(t, items, updates[start].TotalItems)
	require.Zero(t, updates[start].Items)

	last := updates[len(updates)-1]
	require.Equal(t, stage, last.Stage)
	require.True(t, last.Finished)
	require.Equal(t, items, last.Items)
	if bytes >= 0 {
		require.Equal(t, bytes, last.TotalBytes)
		require.Equal(t, bytes, last.Bytes)
	}
}

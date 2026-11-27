package store_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bruli-lab/stowmark-console/internal/domain/encryption"
	"github.com/bruli-lab/stowmark-console/internal/domain/integrity"
	"github.com/bruli-lab/stowmark-console/internal/domain/progress"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/infra/store"
	"github.com/stretchr/testify/require"
)

func objectFiles(t *testing.T, repositoryPath string) []string {
	t.Helper()
	var files []string
	require.NoError(t, filepath.WalkDir(filepath.Join(repositoryPath, "objects"), func(path string, entry fs.DirEntry, err error) error {
		if err == nil && !entry.IsDir() {
			files = append(files, path)
		}
		return err
	}))
	require.GreaterOrEqual(t, len(files), 2)
	return files
}

func TestStoreChecker(t *testing.T) {
	t.Setenv("STOWMARK_LOG_LEVEL", "error")
	source := writeSource(t)
	repositoryPath := filepath.Join(t.TempDir(), "repository")
	recorder := &progressRecorder{}

	session, err := store.Open(t.Context(), repositoryPath, nil, store.WithProgress(recorder))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, session.Close()) })
	require.NoError(t, store.NewRepositories(session).Initialize(t.Context(), repository.NewInitOptions("none", nil, nil)))
	created, _, err := store.NewSnapshots(session).Create(t.Context(), creation(t, source))
	require.NoError(t, err)
	checker := store.NewChecker(session)

	t.Run(`Given a healthy repository
	When its structure is checked
	Then it reports its snapshots and objects without errors`, func(t *testing.T) {
		report, err := checker.Check(t.Context(), integrity.StructureOnly())

		require.NoError(t, err)
		require.Equal(t, 1, report.Snapshots)
		require.Positive(t, report.StoredObjects)
		require.Equal(t, report.StoredObjects, report.ReferencedObjects)
		require.Zero(t, report.ReadObjects)
		require.False(t, report.HasErrors())
	})

	t.Run(`Given a healthy repository
	When all its data is checked
	Then every referenced object is read and the progress reports the reading`, func(t *testing.T) {
		recorder.take()

		report, err := checker.Check(t.Context(), integrity.AllData())

		require.NoError(t, err)
		require.Equal(t, report.ReferencedObjects, report.ReadObjects)
		require.Empty(t, report.CorruptObjects)
		updates := recorder.take()
		require.NotEmpty(t, updates)
		require.Equal(t, progress.StageReadingData, updates[len(updates)-1].Stage)
	})

	t.Run(`Given a repository with a stale lock
	When its structure is checked
	Then the check removes it while taking its own lock`, func(t *testing.T) {
		writeLock(t, repositoryPath, "exclusive", "crashed-host", time.Now().Add(-2*time.Hour))

		report, err := checker.Check(t.Context(), integrity.StructureOnly())

		require.NoError(t, err)
		require.Zero(t, report.StaleLocks)
		locks, err := os.ReadDir(filepath.Join(repositoryPath, "locks"))
		require.NoError(t, err)
		require.Empty(t, locks)
	})

	objects := objectFiles(t, repositoryPath)

	t.Run(`Given a repository with a corrupt object
	When a subset with every object is checked
	Then the corrupt object is reported as an error`, func(t *testing.T) {
		require.NoError(t, os.WriteFile(objects[0], []byte("garbage"), 0o600))
		scope, err := integrity.DataSubset("100%")
		require.NoError(t, err)

		report, err := checker.Check(t.Context(), scope)

		require.NoError(t, err)
		require.Len(t, report.CorruptObjects, 1)
		require.True(t, report.HasErrors())
	})

	t.Run(`Given a repository with a missing object
	When its structure is checked
	Then the missing object names the snapshot that uses it`, func(t *testing.T) {
		require.NoError(t, os.Remove(objects[1]))

		report, err := checker.Check(t.Context(), integrity.StructureOnly())

		require.NoError(t, err)
		require.Len(t, report.MissingObjects, 1)
		require.Equal(t, []string{created.ID()}, report.DamagedSnapshots())
	})
}

func TestStoreCheckerEncrypted(t *testing.T) {
	t.Setenv("STOWMARK_LOG_LEVEL", "error")
	setup := encryptedSetup(t, 2)
	repositoryPath := filepath.Join(t.TempDir(), "repository")

	writer, err := store.Open(t.Context(), repositoryPath, setup.privateKey)
	require.NoError(t, err)
	require.NoError(t, store.NewRepositories(writer).Initialize(t.Context(), setup.initOptions()))
	_, _, err = store.NewSnapshots(writer).Create(t.Context(), creation(t, writeSource(t)))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	reader, err := store.Open(t.Context(), repositoryPath, nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, reader.Close()) })

	t.Run(`Given an encrypted repository opened without its private key
	When its data is checked
	Then the private key is required`, func(t *testing.T) {
		_, err := store.NewChecker(reader).Check(t.Context(), integrity.AllData())

		require.ErrorIs(t, err, encryption.ErrPrivateKeyRequired)
	})

	t.Run(`Given an encrypted repository opened without its private key
	When only its structure is checked
	Then it works`, func(t *testing.T) {
		report, err := store.NewChecker(reader).Check(t.Context(), integrity.StructureOnly())

		require.NoError(t, err)
		require.Equal(t, 1, report.Snapshots)
	})
}

func TestStoreCheckerWithoutRepository(t *testing.T) {
	t.Parallel()

	_, err := store.NewChecker(store.NewSession()).Check(t.Context(), integrity.StructureOnly())

	require.ErrorIs(t, err, store.ErrNotConnected)
}

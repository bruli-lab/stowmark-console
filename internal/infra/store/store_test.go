package store_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/domain/encryption"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
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

type repositorySetup struct {
	privateKey    *string
	publicKey     *string
	formatVersion int
}

func (s repositorySetup) initOptions() repository.InitOptions {
	level := 3
	return repository.NewInitOptions("zstd", &level, s.publicKey).WithFormatVersion(s.formatVersion)
}

func encryptedSetup(t *testing.T, formatVersion int) repositorySetup {
	t.Helper()
	keys := generateKeys(t)
	privateKey, publicKey := keys.PrivateKey(), keys.PublicKey()

	return repositorySetup{privateKey: &privateKey, publicKey: &publicKey, formatVersion: formatVersion}
}

func generateKeys(t *testing.T) encryption.KeyPair {
	t.Helper()

	session, err := store.Open(t.Context(), filepath.Join(t.TempDir(), "keys-repository"), nil)
	require.NoError(t, err)
	defer func() { require.NoError(t, session.Close()) }()

	generation, err := encryption.NewKeyGeneration(t.TempDir())
	require.NoError(t, err)

	keys, err := store.NewKeys(session).Generate(t.Context(), generation)
	require.NoError(t, err)

	return keys
}

func TestStoreWithLocalRepository(t *testing.T) {
	t.Setenv("STOWMARK_LOG_LEVEL", "error")

	t.Run(`Given an unencrypted repository`, func(t *testing.T) {
		exerciseRepository(t, filepath.Join(t.TempDir(), "repository"), repositorySetup{})
	})

	t.Run(`Given a repository encrypted with format version 2`, func(t *testing.T) {
		exerciseRepository(t, filepath.Join(t.TempDir(), "repository"), encryptedSetup(t, 2))
	})

	t.Run(`Given a repository encrypted with format version 3`, func(t *testing.T) {
		exerciseRepository(t, filepath.Join(t.TempDir(), "repository"), encryptedSetup(t, 3))
	})
}

func exerciseRepository(t *testing.T, repositoryURL string, setup repositorySetup) {
	t.Helper()
	source := writeSource(t)

	session, err := store.Open(t.Context(), repositoryURL, setup.privateKey)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, session.Close()) })

	repositories := store.NewRepositories(session)
	snapshots := store.NewSnapshots(session)

	var snapshotID string

	t.Run(`Given a path without repository
	When its status is checked
	Then it is not initialized`, func(t *testing.T) {
		status, err := repositories.Status(t.Context())

		require.NoError(t, err)
		require.Equal(t, repository.StatusUninitialized, status)
	})

	t.Run(`Given a path without repository
	When it is initialized with zstd compression
	Then it becomes ready`, func(t *testing.T) {
		require.NoError(t, repositories.Initialize(t.Context(), setup.initOptions()))

		status, err := repositories.Status(t.Context())

		require.NoError(t, err)
		require.Equal(t, repository.StatusReady, status)
	})

	t.Run(`Given an initialized repository
	When a snapshot of a directory is created
	Then it is listed with its source, files and size`, func(t *testing.T) {
		created, err := snapshots.Create(t.Context(), creation(t, source))
		require.NoError(t, err)
		snapshotID = created.ID()

		listed, err := snapshots.List(t.Context())

		require.NoError(t, err)
		require.Len(t, listed, 1)
		require.Equal(t, snapshotID, listed[0].ID())
		require.Equal(t, listed[0].TotalSize(), created.TotalSize())
		require.Equal(t, source, listed[0].Source())
		require.Equal(t, 2, created.FileCount())
		require.False(t, created.CreatedAt().IsZero())
	})

	t.Run(`Given a source that is already stored
	When a second snapshot of it is created
	Then the created snapshot reports every file of the source, not only the new ones`, func(t *testing.T) {
		created, err := snapshots.Create(t.Context(), creation(t, source))

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
	When one of its files is extracted to a directory
	Then the whole file is written inside that directory`, func(t *testing.T) {
		directory := t.TempDir()

		extraction, err := snapshot.NewFileExtraction(snapshotID, filepath.Join(source, "README.txt"), directory)
		require.NoError(t, err)

		location, err := snapshots.ExtractFile(t.Context(), extraction)

		require.NoError(t, err)
		require.True(t, strings.HasPrefix(location, directory))
		require.Equal(t, "README.txt", filepath.Base(location))
		extracted, err := os.ReadFile(location)
		require.NoError(t, err)
		original, err := os.ReadFile(filepath.Join(source, "README.txt"))
		require.NoError(t, err)
		require.Equal(t, original, extracted)
	})

	t.Run(`Given a snapshot
	When it is verified
	Then there are no failures`, func(t *testing.T) {
		failures, err := snapshots.Verify(t.Context(), verification(t, snapshotID))

		require.NoError(t, err)
		require.Empty(t, failures)
	})

	t.Run(`Given a snapshot
	When it is restored to another directory
	Then the files are restored there and the source is untouched`, func(t *testing.T) {
		destination := filepath.Join(t.TempDir(), "restore")

		failures, err := snapshots.Restore(t.Context(), restoration(t, snapshotID, destination))

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

func creation(t *testing.T, source string) *snapshot.Creation {
	t.Helper()

	created, err := snapshot.NewCreation(source)
	require.NoError(t, err)

	return created
}

func verification(t *testing.T, snapshotID string) *snapshot.Verification {
	t.Helper()

	verified, err := snapshot.NewVerification(snapshotID)
	require.NoError(t, err)

	return verified
}

func restoration(t *testing.T, snapshotID, destination string) *snapshot.Restoration {
	t.Helper()

	restored, err := snapshot.NewRestoration(snapshotID, destination)
	require.NoError(t, err)

	return restored
}

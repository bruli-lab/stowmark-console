package store_test

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/domain/integrity"
	"github.com/bruli-lab/stowmark-console/internal/infra/store"
	"github.com/stretchr/testify/require"
)

func TestStoreWithEmptyFiles(t *testing.T) {
	t.Setenv("STOWMARK_LOG_LEVEL", "error")

	setups := map[string]func(t *testing.T) repositorySetup{
		"unencrypted format version 1": func(*testing.T) repositorySetup { return repositorySetup{formatVersion: 1} },
		"unencrypted format version 2": func(*testing.T) repositorySetup { return repositorySetup{formatVersion: 2} },
		"unencrypted format version 3": func(*testing.T) repositorySetup { return repositorySetup{formatVersion: 3} },
		"encrypted format version 3":   func(t *testing.T) repositorySetup { return encryptedSetup(t, 3) },
	}

	for name, setup := range setups {
		t.Run(`Given an `+name+` repository and a source with an empty file
		When a snapshot is created, verified, checked and restored
		Then it succeeds and the empty file is restored empty`, func(t *testing.T) {
			repositorySetup := setup(t)
			source := writeSource(t)
			require.NoError(t, os.WriteFile(filepath.Join(source, "notes", "empty.txt"), nil, 0o600))

			session, err := store.Open(t.Context(), filepath.Join(t.TempDir(), "repository"), repositorySetup.privateKey)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, session.Close()) })
			require.NoError(t, store.NewRepositories(session).Initialize(t.Context(), repositorySetup.initOptions()))
			snapshots := store.NewSnapshots(session)

			created, _, err := snapshots.Create(t.Context(), creation(t, source))
			require.NoError(t, err, "format version "+strconv.Itoa(repositorySetup.formatVersion))

			failures, err := snapshots.Verify(t.Context(), verification(t, created.ID()))
			require.NoError(t, err)
			require.Empty(t, failures)

			report, err := store.NewChecker(session).Check(t.Context(), integrity.AllData())
			require.NoError(t, err)
			require.False(t, report.HasErrors())

			destination := filepath.Join(t.TempDir(), "restore")
			failures, err = snapshots.Restore(t.Context(), restoration(t, created.ID(), destination))
			require.NoError(t, err)
			require.Empty(t, failures)
			info, err := os.Stat(filepath.Join(destination, "notes", "empty.txt"))
			require.NoError(t, err)
			require.Zero(t, info.Size())
		})
	}
}

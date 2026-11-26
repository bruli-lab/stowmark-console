package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/infra/store"
	"github.com/stretchr/testify/require"
)

const catalogVariable = "STOWMARK_CONSOLE_CATALOG_TEST"

func newEntry(name, location string, environment map[string]string) store.Entry {
	return store.Entry{
		Profile:     repository.NewProfile(name, location, repository.NewInitOptions("", nil, nil)),
		Environment: environment,
	}
}

func TestCatalog(t *testing.T) {
	first := filepath.Join(t.TempDir(), "first")
	second := filepath.Join(t.TempDir(), "second")
	session := store.NewSession()
	t.Cleanup(func() { require.NoError(t, session.Close()) })
	catalog := store.NewCatalog(session, []store.Entry{
		newEntry("first", first, map[string]string{catalogVariable: "first"}),
		newEntry("second", second, nil),
		newEntry("broken", "webdav://127.0.0.1:1/missing-credentials", map[string]string{catalogVariable: "broken"}),
	})
	repositories := store.NewRepositories(session)

	t.Run(`Given a catalog that has not opened any repository
	When the repository status is asked
	Then it reports that no repository is open`, func(t *testing.T) {
		_, err := repositories.Initialized(t.Context())

		require.ErrorIs(t, err, store.ErrNotConnected)
	})

	t.Run(`Given a catalog with several entries
	When the profiles are listed
	Then they keep the configured order`, func(t *testing.T) {
		profiles := catalog.Profiles()

		require.Len(t, profiles, 3)
		require.Equal(t, "first", profiles[0].Name())
		require.Equal(t, "second", profiles[1].Name())
	})

	t.Run(`Given an entry with environment variables
	When it is opened
	Then the variables are applied and the session uses its repository`, func(t *testing.T) {
		profile, err := catalog.Open(t.Context(), "first")
		require.NoError(t, err)
		require.Equal(t, first, profile.Location())
		require.Equal(t, "first", os.Getenv(catalogVariable))

		require.NoError(t, repositories.Initialize(t.Context(), profile.Init()))
		initialized, err := repositories.Initialized(t.Context())
		require.NoError(t, err)
		require.True(t, initialized)
	})

	t.Run(`Given an entry that cannot be opened
	When it is opened
	Then the error is returned and the previous repository and variables are kept`, func(t *testing.T) {
		_, err := catalog.Open(t.Context(), "broken")

		require.Error(t, err)
		require.Equal(t, "first", os.Getenv(catalogVariable))
		initialized, err := repositories.Initialized(t.Context())
		require.NoError(t, err)
		require.True(t, initialized)
	})

	t.Run(`Given an entry without environment variables
	When it is opened after another one
	Then the previous variables are restored and its repository is used`, func(t *testing.T) {
		_, err := catalog.Open(t.Context(), "second")
		require.NoError(t, err)

		_, defined := os.LookupEnv(catalogVariable)
		require.False(t, defined)
		initialized, err := repositories.Initialized(t.Context())
		require.NoError(t, err)
		require.False(t, initialized)
	})

	t.Run(`Given an unknown name
	When it is opened
	Then an unknown repository error is returned`, func(t *testing.T) {
		_, err := catalog.Open(t.Context(), "missing")

		require.ErrorIs(t, err, repository.ErrUnknownProfile)
	})
}

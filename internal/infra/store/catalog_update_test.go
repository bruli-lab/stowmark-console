package store_test

import (
	"path/filepath"
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/infra/store"
	"github.com/stretchr/testify/require"
)

func TestCatalogUpdate(t *testing.T) {
	t.Setenv("STOWMARK_LOG_LEVEL", "error")
	location := filepath.Join(t.TempDir(), "repository")
	session := store.NewSession()
	t.Cleanup(func() { require.NoError(t, session.Close()) })
	catalog := store.NewCatalog(session, []store.Entry{newEntry("vault", location, nil)})
	repositories := store.NewRepositories(session)

	t.Run(`Given a profile that is not in the catalog
	When it is updated
	Then it is rejected`, func(t *testing.T) {
		_, err := catalog.Update(t.Context(), repository.NewProfile("other", location, repository.NewInitOptions("", nil, nil)))

		require.ErrorIs(t, err, repository.ErrUnknownProfile)
	})

	t.Run(`Given an uninitialized repository and a new key pair
	When its profile is updated with the keys and the repository is initialized
	Then it is an encrypted format version 3 repository that the session can use`, func(t *testing.T) {
		profile, err := catalog.Open(t.Context(), "vault")
		require.NoError(t, err)
		keys := generateKeys(t)

		updated, err := catalog.Update(t.Context(), profile.
			WithInit(profile.Init().WithFormatVersion(repository.LatestFormatVersion)).
			WithKeys(keys.PublicKey(), keys.PrivateKey()))
		require.NoError(t, err)
		require.NoError(t, repositories.Initialize(t.Context(), updated.Init()))

		require.True(t, updated.Init().EncryptsMetadata())
		require.Equal(t, keys.PrivateKey(), *updated.PrivateKey())
		require.NotEmpty(t, updated.Fingerprint())
		require.Equal(t, updated.Init(), catalog.Profiles()[0].Init())
		status, err := repositories.Status(t.Context())
		require.NoError(t, err)
		require.Equal(t, repository.StatusReady, status)
		_, _, err = store.NewSnapshots(session).Create(t.Context(), creation(t, writeSource(t)))
		require.NoError(t, err)
		fingerprint, _ := configFingerprint(t, location)
		require.Equal(t, fingerprint, updated.Fingerprint())
	})
}

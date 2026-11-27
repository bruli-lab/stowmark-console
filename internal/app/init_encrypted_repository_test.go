package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/app"
	"github.com/bruli-lab/stowmark-console/internal/domain/encryption"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/mocks"
	"github.com/stretchr/testify/require"
)

type encryptedInitPorts struct {
	catalog      *mocks.CatalogMock
	repositories *mocks.RepositoryMock
	keys         *mocks.KeysMock
}

func newEncryptedInitPorts() encryptedInitPorts {
	return encryptedInitPorts{
		catalog: &mocks.CatalogMock{
			UpdateFunc: func(_ context.Context, profile repository.Profile) (repository.Profile, error) {
				return profile.WithFingerprint("SHA256:new"), nil
			},
		},
		repositories: &mocks.RepositoryMock{
			InitializeFunc: func(context.Context, repository.InitOptions) error { return nil },
		},
		keys: &mocks.KeysMock{
			GenerateFunc: func(context.Context, *encryption.KeyGeneration) (encryption.KeyPair, error) {
				return encryption.NewKeyPair(privateKeyPath, publicKeyPath)
			},
		},
	}
}

func (p encryptedInitPorts) handler() app.InitEncryptedRepository {
	return app.NewInitEncryptedRepository(p.catalog, p.repositories, p.keys)
}

func TestInitEncryptedRepository(t *testing.T) {
	t.Parallel()

	level := 3
	profile := repository.NewProfile("vault", "/backups", repository.NewInitOptions("zstd", &level, nil))

	t.Run(`Given a command that is not an encrypted init command
	When it is handled
	Then an invalid command error is returned`, func(t *testing.T) {
		t.Parallel()

		requireInvalidCommand(t, newEncryptedInitPorts().handler())
	})

	t.Run(`Given a relative keys folder
	When the repository is initialized encrypted
	Then nothing is generated nor initialized`, func(t *testing.T) {
		t.Parallel()
		ports := newEncryptedInitPorts()

		_, err := ports.handler().Handle(t.Context(), app.InitEncryptedRepositoryCommand{Profile: profile, KeysFolder: "keys"})

		require.ErrorIs(t, err, encryption.ErrKeyFolderNotAbsolute)
		require.Empty(t, ports.keys.GenerateCalls())
		require.Empty(t, ports.repositories.InitializeCalls())
	})

	t.Run(`Given a profile without name
	When the repository is initialized encrypted
	Then it is rejected`, func(t *testing.T) {
		t.Parallel()
		ports := newEncryptedInitPorts()

		_, err := ports.handler().Handle(t.Context(), app.InitEncryptedRepositoryCommand{KeysFolder: "/keys"})

		require.ErrorIs(t, err, app.ErrRepositoryNameRequired)
		require.Empty(t, ports.keys.GenerateCalls())
	})

	t.Run(`Given an unencrypted profile and a keys folder
	When the repository is initialized encrypted
	Then a key pair is generated, the profile is reopened with it and the repository is initialized with format version 3`, func(t *testing.T) {
		t.Parallel()
		ports := newEncryptedInitPorts()

		events, err := ports.handler().Handle(t.Context(), app.InitEncryptedRepositoryCommand{Profile: profile, KeysFolder: "/keys"})

		require.NoError(t, err)
		require.Equal(t, "/keys", ports.keys.GenerateCalls()[0].Generation.Folder())
		updated := ports.catalog.UpdateCalls()[0].Profile
		require.Equal(t, privateKeyPath, *updated.PrivateKey())
		require.Equal(t, publicKeyPath, *updated.Init().PublicKey())
		options := ports.repositories.InitializeCalls()[0].Options
		require.Equal(t, repository.LatestFormatVersion, options.FormatVersion())
		require.Equal(t, "zstd", options.Compression())
		require.Equal(t, publicKeyPath, *options.PublicKey())
		require.Len(t, events, 3)
		generated, ok := events[0].(encryption.KeysGenerated)
		require.True(t, ok)
		require.Equal(t, privateKeyPath, generated.Keys.PrivateKey())
		opened, ok := events[1].(repository.Opened)
		require.True(t, ok)
		require.Equal(t, "SHA256:new", opened.Profile.Fingerprint())
		require.Equal(t, repository.InitializedEventName, events[2].EventName())
	})

	t.Run(`Given a repository that cannot be reopened with the new keys
	When the repository is initialized encrypted
	Then the error is returned and the repository is not initialized`, func(t *testing.T) {
		t.Parallel()
		ports := newEncryptedInitPorts()
		ports.catalog.UpdateFunc = func(context.Context, repository.Profile) (repository.Profile, error) {
			return repository.Profile{}, errors.New("unreachable")
		}

		_, err := ports.handler().Handle(t.Context(), app.InitEncryptedRepositoryCommand{Profile: profile, KeysFolder: "/keys"})

		require.ErrorContains(t, err, "unreachable")
		require.Empty(t, ports.repositories.InitializeCalls())
	})
}

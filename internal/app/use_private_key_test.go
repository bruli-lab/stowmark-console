package app_test

import (
	"context"
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/app"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/mocks"
	"github.com/stretchr/testify/require"
)

func TestUsePrivateKey(t *testing.T) {
	t.Parallel()

	profile := repository.NewProfile("default", "/backups", repository.NewInitOptions("zstd", nil, nil))

	t.Run(`Given a command that is not a use private key command
	When it is handled
	Then an invalid command error is returned`, func(t *testing.T) {
		t.Parallel()

		requireInvalidCommand(t, app.NewUsePrivateKey(&mocks.CatalogMock{}))
	})

	t.Run(`Given a repository and a private key file
	When the key is used
	Then the profile is reopened with the key and an opened event is returned`, func(t *testing.T) {
		t.Parallel()
		catalog := &mocks.CatalogMock{
			UpdateFunc: func(_ context.Context, profile repository.Profile) (repository.Profile, error) {
				return profile.WithFingerprint("SHA256:abc"), nil
			},
		}

		events, err := app.NewUsePrivateKey(catalog).Handle(t.Context(), app.UsePrivateKeyCommand{
			Profile:    profile,
			PrivateKey: "/keys/private.pem",
		})

		require.NoError(t, err)
		opened := singleEvent[repository.Opened](t, events, repository.OpenedEventName)
		require.Equal(t, "/keys/private.pem", *opened.Profile.PrivateKey())
		require.Equal(t, "SHA256:abc", opened.Profile.Fingerprint())
		require.Equal(t, "/keys/private.pem", *catalog.UpdateCalls()[0].Profile.PrivateKey())
	})

	tests := []struct {
		name    string
		command app.UsePrivateKeyCommand
		err     error
	}{
		{name: "no repository", command: app.UsePrivateKeyCommand{PrivateKey: "/keys/private.pem"}, err: app.ErrRepositoryNameRequired},
		{name: "no private key file", command: app.UsePrivateKeyCommand{Profile: profile}, err: app.ErrPrivateKeyPathRequired},
	}

	for _, tt := range tests {
		t.Run(`Given `+tt.name+`
		When the key is used
		Then it is rejected without touching the catalog`, func(t *testing.T) {
			t.Parallel()
			catalog := &mocks.CatalogMock{}

			_, err := app.NewUsePrivateKey(catalog).Handle(t.Context(), tt.command)

			require.ErrorIs(t, err, tt.err)
			require.Empty(t, catalog.UpdateCalls())
		})
	}

	t.Run(`Given a catalog that cannot reopen the repository
	When the key is used
	Then the error is returned`, func(t *testing.T) {
		t.Parallel()
		catalog := &mocks.CatalogMock{
			UpdateFunc: func(context.Context, repository.Profile) (repository.Profile, error) {
				return repository.Profile{}, errBackend
			},
		}

		_, err := app.NewUsePrivateKey(catalog).Handle(t.Context(), app.UsePrivateKeyCommand{
			Profile:    profile,
			PrivateKey: "/keys/private.pem",
		})

		require.ErrorIs(t, err, errBackend)
	})
}

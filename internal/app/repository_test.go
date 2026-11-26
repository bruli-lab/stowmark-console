package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/app"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/mocks"
	"github.com/stretchr/testify/require"
)

var errBackend = errors.New("backend unavailable")

func TestRepositoryStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		initialized bool
		err         error
	}{
		{name: "an initialized repository, then it reports it as initialized", initialized: true},
		{name: "a repository not initialized yet, then it reports it as not initialized"},
		{name: "a repository that cannot be reached, then it returns the error", err: errBackend},
	}

	for _, tt := range tests {
		t.Run(`Given `+tt.name, func(t *testing.T) {
			t.Parallel()
			repo := &mocks.RepositoryMock{
				InitializedFunc: func(context.Context) (bool, error) { return tt.initialized, tt.err },
			}

			initialized, err := app.NewRepositoryStatus(repo).Handle(t.Context(), app.RepositoryStatusQuery{})

			require.ErrorIs(t, err, tt.err)
			require.Equal(t, tt.initialized, initialized)
		})
	}
}

func TestInitRepository(t *testing.T) {
	t.Parallel()

	t.Run(`Given init options
	When the repository is initialized
	Then the options are passed to the repository and an initialized event is returned`, func(t *testing.T) {
		t.Parallel()
		level := 3
		options := repository.NewInitOptions("zstd", &level, nil)
		repo := &mocks.RepositoryMock{
			InitializeFunc: func(context.Context, repository.InitOptions) error { return nil },
		}

		event, err := app.NewInitRepository(repo).Handle(t.Context(), app.InitRepositoryCommand{Options: options})

		require.NoError(t, err)
		require.Equal(t, options, event.Options)
		require.Len(t, repo.InitializeCalls(), 1)
		require.Equal(t, options, repo.InitializeCalls()[0].Options)
	})

	t.Run(`Given a repository that fails to initialize
	When the repository is initialized
	Then the error is returned`, func(t *testing.T) {
		t.Parallel()
		repo := &mocks.RepositoryMock{
			InitializeFunc: func(context.Context, repository.InitOptions) error { return errBackend },
		}

		_, err := app.NewInitRepository(repo).Handle(t.Context(), app.InitRepositoryCommand{})

		require.ErrorIs(t, err, errBackend)
	})
}

func TestListRepositories(t *testing.T) {
	t.Parallel()

	t.Run(`Given a catalog with profiles
	When the repositories are listed
	Then the profiles are returned in order`, func(t *testing.T) {
		t.Parallel()
		profiles := []repository.Profile{
			repository.NewProfile("local", "/backups", repository.NewInitOptions("", nil, nil)),
			repository.NewProfile("nas", "smb://user@nas/backups", repository.NewInitOptions("", nil, nil)),
		}
		catalog := &mocks.CatalogMock{ProfilesFunc: func() []repository.Profile { return profiles }}

		listed, err := app.NewListRepositories(catalog).Handle(t.Context(), app.ListRepositoriesQuery{})

		require.NoError(t, err)
		require.Equal(t, profiles, listed)
	})

	t.Run(`Given an empty catalog
	When the repositories are listed
	Then it returns an error`, func(t *testing.T) {
		t.Parallel()
		catalog := &mocks.CatalogMock{ProfilesFunc: func() []repository.Profile { return nil }}

		_, err := app.NewListRepositories(catalog).Handle(t.Context(), app.ListRepositoriesQuery{})

		require.ErrorIs(t, err, app.ErrNoRepositories)
	})
}

func TestOpenRepository(t *testing.T) {
	t.Parallel()

	t.Run(`Given a repository name
	When the repository is opened
	Then an opened event with its profile is returned`, func(t *testing.T) {
		t.Parallel()
		profile := repository.NewProfile("nas", "smb://user@nas/backups", repository.NewInitOptions("", nil, nil))
		catalog := &mocks.CatalogMock{
			OpenFunc: func(context.Context, string) (repository.Profile, error) { return profile, nil },
		}

		event, err := app.NewOpenRepository(catalog).Handle(t.Context(), app.OpenRepositoryCommand{Name: "nas"})

		require.NoError(t, err)
		require.Equal(t, profile, event.Profile)
		require.Equal(t, "nas", catalog.OpenCalls()[0].Name)
	})

	t.Run(`Given an empty repository name
	When the repository is opened
	Then it returns an error without touching the catalog`, func(t *testing.T) {
		t.Parallel()
		catalog := &mocks.CatalogMock{}

		_, err := app.NewOpenRepository(catalog).Handle(t.Context(), app.OpenRepositoryCommand{})

		require.ErrorIs(t, err, app.ErrRepositoryNameRequired)
		require.Empty(t, catalog.OpenCalls())
	})

	t.Run(`Given a repository that cannot be opened
	When the repository is opened
	Then the error is returned`, func(t *testing.T) {
		t.Parallel()
		catalog := &mocks.CatalogMock{
			OpenFunc: func(context.Context, string) (repository.Profile, error) { return repository.Profile{}, errBackend },
		}

		_, err := app.NewOpenRepository(catalog).Handle(t.Context(), app.OpenRepositoryCommand{Name: "nas"})

		require.ErrorIs(t, err, errBackend)
	})
}

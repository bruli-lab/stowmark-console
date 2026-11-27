package app_test

import (
	"context"
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/app"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/mocks"
	"github.com/stretchr/testify/require"
)

func TestRepositoryStatus(t *testing.T) {
	t.Parallel()

	t.Run(`Given a query that is not a repository status query
	When it is handled
	Then an invalid query error is returned`, func(t *testing.T) {
		t.Parallel()

		requireInvalidQuery(t, app.NewRepositoryStatus(&mocks.RepositoryMock{}))
	})

	tests := []struct {
		name   string
		status repository.Status
		err    error
	}{
		{name: "an initialized repository, then it reports it as ready", status: repository.StatusReady},
		{name: "a repository not initialized yet, then it reports it as not initialized", status: repository.StatusUninitialized},
		{name: "an encrypted repository without its key, then it reports that the key is required", status: repository.StatusKeyRequired},
		{name: "a repository that cannot be reached, then it returns the error", err: errBackend},
	}

	for _, tt := range tests {
		t.Run(`Given `+tt.name, func(t *testing.T) {
			t.Parallel()
			repo := &mocks.RepositoryMock{
				StatusFunc: func(context.Context) (repository.Status, error) { return tt.status, tt.err },
			}

			status, err := app.NewRepositoryStatus(repo).Handle(t.Context(), app.RepositoryStatusQuery{})

			require.ErrorIs(t, err, tt.err)
			require.Equal(t, tt.status, status)
		})
	}
}

func TestInitRepository(t *testing.T) {
	t.Parallel()

	t.Run(`Given a command that is not an init repository command
	When it is handled
	Then an invalid command error is returned`, func(t *testing.T) {
		t.Parallel()

		requireInvalidCommand(t, app.NewInitRepository(&mocks.RepositoryMock{}))
	})

	t.Run(`Given init options
	When the repository is initialized
	Then the options are passed to the repository and an initialized event is returned`, func(t *testing.T) {
		t.Parallel()
		level := 3
		options := repository.NewInitOptions("zstd", &level, nil)
		repo := &mocks.RepositoryMock{
			InitializeFunc: func(context.Context, repository.InitOptions) error { return nil },
		}

		events, err := app.NewInitRepository(repo).Handle(t.Context(), app.InitRepositoryCommand{Options: options})

		require.NoError(t, err)
		initialized := singleEvent[repository.Initialized](t, events, repository.InitializedEventName)
		require.Equal(t, options, initialized.Options)
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

func TestUnlockRepository(t *testing.T) {
	t.Parallel()

	t.Run(`Given a command that is not an unlock repository command
	When it is handled
	Then an invalid command error is returned`, func(t *testing.T) {
		t.Parallel()

		requireInvalidCommand(t, app.NewUnlockRepository(&mocks.RepositoryMock{}))
	})

	for _, all := range []bool{false, true} {
		t.Run(`Given a repository with locks
		When it is unlocked
		Then the choice of removing every lock reaches the repository and an unlocked event reports the locks`, func(t *testing.T) {
			t.Parallel()
			report := repository.LockReport{Removed: 2, Active: 1}
			repo := &mocks.RepositoryMock{
				UnlockFunc: func(context.Context, bool) (repository.LockReport, error) { return report, nil },
			}

			events, err := app.NewUnlockRepository(repo).Handle(t.Context(), app.UnlockRepositoryCommand{All: all})

			require.NoError(t, err)
			unlocked := singleEvent[repository.Unlocked](t, events, repository.UnlockedEventName)
			require.Equal(t, report, unlocked.Report)
			require.Equal(t, all, unlocked.All)
			require.Equal(t, all, repo.UnlockCalls()[0].All)
		})
	}

	t.Run(`Given a repository that fails to unlock
	When it is unlocked
	Then the error is returned`, func(t *testing.T) {
		t.Parallel()
		repo := &mocks.RepositoryMock{
			UnlockFunc: func(context.Context, bool) (repository.LockReport, error) {
				return repository.LockReport{}, errBackend
			},
		}

		_, err := app.NewUnlockRepository(repo).Handle(t.Context(), app.UnlockRepositoryCommand{})

		require.ErrorIs(t, err, errBackend)
	})
}

func TestListRepositories(t *testing.T) {
	t.Parallel()

	t.Run(`Given a query that is not a list repositories query
	When it is handled
	Then an invalid query error is returned`, func(t *testing.T) {
		t.Parallel()

		requireInvalidQuery(t, app.NewListRepositories(&mocks.CatalogMock{}))
	})

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

	t.Run(`Given a command that is not an open repository command
	When it is handled
	Then an invalid command error is returned`, func(t *testing.T) {
		t.Parallel()

		requireInvalidCommand(t, app.NewOpenRepository(&mocks.CatalogMock{}))
	})

	t.Run(`Given a repository name
	When the repository is opened
	Then an opened event with its profile is returned`, func(t *testing.T) {
		t.Parallel()
		profile := repository.NewProfile("nas", "smb://user@nas/backups", repository.NewInitOptions("", nil, nil))
		catalog := &mocks.CatalogMock{
			OpenFunc: func(context.Context, string) (repository.Profile, error) { return profile, nil },
		}

		events, err := app.NewOpenRepository(catalog).Handle(t.Context(), app.OpenRepositoryCommand{RepositoryName: "nas"})

		require.NoError(t, err)
		opened := singleEvent[repository.Opened](t, events, repository.OpenedEventName)
		require.Equal(t, profile, opened.Profile)
		require.Equal(t, "nas", opened.AggregateRootID())
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

		_, err := app.NewOpenRepository(catalog).Handle(t.Context(), app.OpenRepositoryCommand{RepositoryName: "nas"})

		require.ErrorIs(t, err, errBackend)
	})
}

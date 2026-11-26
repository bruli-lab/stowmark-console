package repository_test

import (
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/stretchr/testify/require"
)

func TestEvents(t *testing.T) {
	t.Parallel()

	t.Run(`Given a profile
	When an opened event is built
	Then it is named and identified by the repository`, func(t *testing.T) {
		t.Parallel()
		profile := repository.NewProfile("nas", "smb://user@nas/backups", repository.NewInitOptions("", nil, nil))

		opened := repository.NewOpened(profile)

		require.Equal(t, repository.OpenedEventName, opened.EventName())
		require.Equal(t, "nas", opened.AggregateRootID())
		require.Equal(t, profile, opened.Profile)
		require.NotEmpty(t, opened.EventID())
	})

	t.Run(`Given init options
	When an initialized event is built
	Then it is named and keeps the options`, func(t *testing.T) {
		t.Parallel()
		level := 3
		options := repository.NewInitOptions("zstd", &level, nil)

		initialized := repository.NewInitialized(options)

		require.Equal(t, repository.InitializedEventName, initialized.EventName())
		require.Equal(t, options, initialized.Options)
	})
}

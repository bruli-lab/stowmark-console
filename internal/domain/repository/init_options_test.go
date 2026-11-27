package repository_test

import (
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/stretchr/testify/require"
)

func TestInitOptionsValidate(t *testing.T) {
	t.Parallel()

	level := func(value int) *int { return &value }

	tests := []struct {
		name    string
		options repository.InitOptions
		err     error
	}{
		{name: "no compression", options: repository.NewInitOptions("", nil, nil)},
		{name: "zstd at its highest level", options: repository.NewInitOptions("zstd", level(22), nil)},
		{name: "lz4 at level 0", options: repository.NewInitOptions("lz4", level(0), nil)},
		{name: "xz without level", options: repository.NewInitOptions("xz", nil, nil).WithFormatVersion(3)},
		{name: "an unknown compression", options: repository.NewInitOptions("brotli", nil, nil), err: repository.ErrInvalidCompression},
		{name: "zstd above its range", options: repository.NewInitOptions("zstd", level(23), nil), err: repository.ErrInvalidCompressionLevel},
		{name: "gzip below its range", options: repository.NewInitOptions("gzip", level(0), nil), err: repository.ErrInvalidCompressionLevel},
		{name: "a level for xz", options: repository.NewInitOptions("xz", level(3), nil), err: repository.ErrInvalidCompressionLevel},
		{name: "format version 4", options: repository.NewInitOptions("", nil, nil).WithFormatVersion(4), err: repository.ErrInvalidFormatVersion},
	}

	for _, tt := range tests {
		t.Run(`Given init options with `+tt.name+`
		When they are validated
		Then they are accepted only within the Stowmark ranges`, func(t *testing.T) {
			t.Parallel()

			err := tt.options.Validate()

			if tt.err == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.err)
		})
	}

	t.Run(`Given a compression with levels
	When its levels are asked
	Then the range and the default are returned, and none for xz`, func(t *testing.T) {
		t.Parallel()

		levels, ok := repository.LevelsOf("zstd")
		require.True(t, ok)
		require.Equal(t, repository.CompressionLevels{Minimum: 1, Maximum: 22, Default: 3}, levels)
		_, ok = repository.LevelsOf("xz")
		require.False(t, ok)
	})
}

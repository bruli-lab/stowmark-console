package console

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitSettings(t *testing.T) {
	t.Parallel()

	t.Run(`Given empty init settings
	When the init options are built
	Then the repository is created without compression nor encryption`, func(t *testing.T) {
		t.Parallel()
		settings := InitSettings{}

		options := settings.options()

		require.Equal(t, "none", options.Compression.Type)
		require.Nil(t, options.Compression.Level)
		require.Nil(t, options.PublicKey)
		require.Contains(t, settings.encryptionSummary(), "none")
		require.Equal(t, "none", settings.compressionSummary())
	})

	t.Run(`Given init settings with public key and compression level
	When the init options are built
	Then they are passed to stowmark and summarized`, func(t *testing.T) {
		t.Parallel()
		publicKey := "/keys/public.pem"
		level := 3
		settings := InitSettings{PublicKey: &publicKey, Compression: "zstd", CompressionLevel: &level}

		options := settings.options()

		require.Equal(t, "zstd", options.Compression.Type)
		require.Equal(t, &level, options.Compression.Level)
		require.Equal(t, &publicKey, options.PublicKey)
		require.Equal(t, "public key /keys/public.pem", settings.encryptionSummary())
		require.Equal(t, "zstd (level 3)", settings.compressionSummary())
	})
}

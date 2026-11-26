package config_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/config"
	"github.com/stretchr/testify/require"
)

func TestFromEnvironment(t *testing.T) {
	t.Parallel()

	t.Run(`Given an environment without repository
	When the configuration is read
	Then it returns an error`, func(t *testing.T) {
		t.Parallel()

		_, err := config.FromEnvironment(map[string]string{})

		require.ErrorContains(t, err, "STOWMARK_REPOSITORY")
	})

	t.Run(`Given an environment with only the repository
	When the configuration is read
	Then it uses the defaults`, func(t *testing.T) {
		t.Parallel()

		conf, err := config.FromEnvironment(map[string]string{"STOWMARK_REPOSITORY": "/backups"})

		require.NoError(t, err)
		require.Equal(t, "/backups", conf.Repository)
		require.Equal(t, "none", conf.Compression)
		require.Nil(t, conf.CompressionLevel)
		require.Nil(t, conf.PrivateKey)
		require.Nil(t, conf.PublicKey)
		require.True(t, strings.HasSuffix(conf.LogFile, filepath.Join("stowmark-console", "console.log")))
	})

	t.Run(`Given an environment with every option
	When the configuration is read
	Then every option is loaded`, func(t *testing.T) {
		t.Parallel()

		conf, err := config.FromEnvironment(map[string]string{
			"STOWMARK_REPOSITORY":        "ssh://user@host/backups",
			"STOWMARK_PRIVATE_KEY":       "/keys/private.pem",
			"STOWMARK_PUBLIC_KEY":        "/keys/public.pem",
			"STOWMARK_COMPRESSION":       "zstd",
			"STOWMARK_COMPRESSION_LEVEL": "3",
			"STOWMARK_CONSOLE_LOG_FILE":  "/tmp/console.log",
		})

		require.NoError(t, err)
		require.Equal(t, "ssh://user@host/backups", conf.Repository)
		require.Equal(t, "/keys/private.pem", *conf.PrivateKey)
		require.Equal(t, "/keys/public.pem", *conf.PublicKey)
		require.Equal(t, "zstd", conf.Compression)
		require.Equal(t, 3, *conf.CompressionLevel)
		require.Equal(t, "/tmp/console.log", conf.LogFile)
	})

	t.Run(`Given an unknown compression type
	When the configuration is read
	Then it returns an invalid compression error`, func(t *testing.T) {
		t.Parallel()

		_, err := config.FromEnvironment(map[string]string{
			"STOWMARK_REPOSITORY":  "/backups",
			"STOWMARK_COMPRESSION": "rar",
		})

		require.ErrorIs(t, err, config.ErrInvalidCompression)
	})

	t.Run(`Given a compression level that is not a number
	When the configuration is read
	Then it returns an error`, func(t *testing.T) {
		t.Parallel()

		_, err := config.FromEnvironment(map[string]string{
			"STOWMARK_REPOSITORY":        "/backups",
			"STOWMARK_COMPRESSION_LEVEL": "high",
		})

		require.ErrorContains(t, err, "CompressionLevel")
	})
}

package config_test

import (
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bruli-lab/stowmark-console/internal/config"
	"github.com/stretchr/testify/require"
)

func environment(t *testing.T, values map[string]string) map[string]string {
	t.Helper()

	result := map[string]string{"STOWMARK_CONSOLE_CONFIG": filepath.Join(t.TempDir(), "missing.yaml")}
	maps.Copy(result, values)

	return result
}

func repositoriesFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "repositories.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

func TestFromEnvironment(t *testing.T) {
	t.Parallel()

	t.Run(`Given an environment without repository and without repositories file
	When the configuration is read
	Then it returns a no repositories error`, func(t *testing.T) {
		t.Parallel()

		_, err := config.FromEnvironment(environment(t, nil))

		require.ErrorIs(t, err, config.ErrNoRepositories)
		require.ErrorContains(t, err, "STOWMARK_REPOSITORY")
	})

	t.Run(`Given an environment with only the repository
	When the configuration is read
	Then a default repository is configured with the defaults`, func(t *testing.T) {
		t.Parallel()

		conf, err := config.FromEnvironment(environment(t, map[string]string{"STOWMARK_REPOSITORY": "/backups"}))

		require.NoError(t, err)
		require.Len(t, conf.Repositories, 1)
		repo := conf.Repositories[0]
		require.Equal(t, config.DefaultRepositoryName, repo.Name)
		require.Equal(t, "/backups", repo.URL)
		require.Equal(t, "none", repo.Compression)
		require.Nil(t, repo.CompressionLevel)
		require.Nil(t, repo.PrivateKey)
		require.Nil(t, repo.PublicKey)
		require.Equal(t, 2, repo.FormatVersion)
		require.True(t, strings.HasSuffix(conf.LogFile, filepath.Join("stowmark-console", "console.log")))
	})

	t.Run(`Given an environment with every option
	When the configuration is read
	Then every option is loaded into the default repository`, func(t *testing.T) {
		t.Parallel()

		conf, err := config.FromEnvironment(environment(t, map[string]string{
			"STOWMARK_REPOSITORY":        "ssh://user@host/backups",
			"STOWMARK_PRIVATE_KEY":       "/keys/private.pem",
			"STOWMARK_PUBLIC_KEY":        "/keys/public.pem",
			"STOWMARK_COMPRESSION":       "zstd",
			"STOWMARK_COMPRESSION_LEVEL": "3",
			"STOWMARK_FORMAT_VERSION":    "3",
			"STOWMARK_CONSOLE_LOG_FILE":  "/tmp/console.log",
		}))

		require.NoError(t, err)
		repo := conf.Repositories[0]
		require.Equal(t, "ssh://user@host/backups", repo.URL)
		require.Equal(t, "/keys/private.pem", *repo.PrivateKey)
		require.Equal(t, "/keys/public.pem", *repo.PublicKey)
		require.Equal(t, "zstd", repo.Compression)
		require.Equal(t, 3, *repo.CompressionLevel)
		require.Equal(t, 3, repo.FormatVersion)
		require.Equal(t, "/tmp/console.log", conf.LogFile)
	})

	for _, version := range []string{"4", "-1"} {
		t.Run(`Given the format version `+version+`
		When the configuration is read
		Then it returns an invalid format version error`, func(t *testing.T) {
			t.Parallel()

			_, err := config.FromEnvironment(environment(t, map[string]string{
				"STOWMARK_REPOSITORY":     "/backups",
				"STOWMARK_FORMAT_VERSION": version,
			}))

			require.ErrorIs(t, err, config.ErrInvalidFormat)
		})
	}

	t.Run(`Given an unknown compression type
	When the configuration is read
	Then it returns an invalid compression error`, func(t *testing.T) {
		t.Parallel()

		_, err := config.FromEnvironment(environment(t, map[string]string{
			"STOWMARK_REPOSITORY":  "/backups",
			"STOWMARK_COMPRESSION": "rar",
		}))

		require.ErrorIs(t, err, config.ErrInvalidCompression)
	})

	t.Run(`Given a compression level that is not a number
	When the configuration is read
	Then it returns an error`, func(t *testing.T) {
		t.Parallel()

		_, err := config.FromEnvironment(environment(t, map[string]string{
			"STOWMARK_REPOSITORY":        "/backups",
			"STOWMARK_COMPRESSION_LEVEL": "high",
		}))

		require.ErrorContains(t, err, "CompressionLevel")
	})
}

func TestRepositoriesFile(t *testing.T) {
	t.Parallel()

	home, err := os.UserHomeDir()
	require.NoError(t, err)

	t.Run(`Given a repositories file and a repository in the environment
	When the configuration is read
	Then the environment repository comes first followed by the file ones`, func(t *testing.T) {
		t.Parallel()
		path := repositoriesFile(t, `
repositories:
  - name: nas
    url: smb://pablo@nas/backups
    compression: zstd
    compression_level: 5
    env:
      STOWMARK_SMB_PASSWORD: secret
  - name: laptop
    url: ~/backups
    private_key: ~/.config/stowmark/key
    format_version: 3
    retention:
      keep_last: 7
      keep_daily: 14
      keep_within: 2w
    exclude:
      - "*.tmp"
      - node_modules/
    skip_ignore_files: true
`)

		conf, err := config.FromEnvironment(map[string]string{
			"STOWMARK_REPOSITORY":     "/backups",
			"STOWMARK_CONSOLE_CONFIG": path,
		})

		require.NoError(t, err)
		require.Equal(t, path, conf.RepositoriesFile)
		require.Len(t, conf.Repositories, 3)
		require.Equal(t, config.DefaultRepositoryName, conf.Repositories[0].Name)
		nas := conf.Repositories[1]
		require.Equal(t, "smb://pablo@nas/backups", nas.URL)
		require.Equal(t, "zstd", nas.Compression)
		require.Equal(t, 5, *nas.CompressionLevel)
		require.Equal(t, map[string]string{"STOWMARK_SMB_PASSWORD": "secret"}, nas.Env)
		laptop := conf.Repositories[2]
		require.Equal(t, filepath.Join(home, "backups"), laptop.URL)
		require.Equal(t, filepath.Join(home, ".config", "stowmark", "key"), *laptop.PrivateKey)
		require.Equal(t, "none", laptop.Compression)
		require.Equal(t, 3, laptop.FormatVersion)
		require.Equal(t, 2, nas.FormatVersion)
		require.Nil(t, nas.Retention)
		require.Equal(t, 7, laptop.Retention.KeepLast)
		require.Equal(t, 14, laptop.Retention.KeepDaily)
		require.Equal(t, 14*24*time.Hour, laptop.Retention.Within)
		require.Equal(t, []string{"*.tmp", "node_modules/"}, laptop.Exclude)
		require.True(t, laptop.SkipIgnoreFiles)
		require.Empty(t, nas.Exclude)
		require.False(t, nas.SkipIgnoreFiles)
	})

	t.Run(`Given only a repositories file
	When the configuration is read
	Then its repositories are configured`, func(t *testing.T) {
		t.Parallel()
		path := repositoriesFile(t, "repositories:\n  - name: s3\n    url: s3://bucket/stowmark\n")

		conf, err := config.FromEnvironment(map[string]string{"STOWMARK_CONSOLE_CONFIG": path})

		require.NoError(t, err)
		require.Len(t, conf.Repositories, 1)
		require.Equal(t, "s3", conf.Repositories[0].Name)
	})

	tests := []struct {
		name     string
		content  string
		expected error
		message  string
	}{
		{
			name:     "a repository without name",
			content:  "repositories:\n  - url: /backups\n",
			expected: config.ErrInvalidRepository,
		},
		{
			name:     "a repository without url",
			content:  "repositories:\n  - name: nas\n",
			expected: config.ErrInvalidRepository,
		},
		{
			name:     "two repositories with the same name",
			content:  "repositories:\n  - name: nas\n    url: /a\n  - name: nas\n    url: /b\n",
			expected: config.ErrInvalidRepository,
		},
		{
			name:     "an unknown compression",
			content:  "repositories:\n  - name: nas\n    url: /a\n    compression: rar\n",
			expected: config.ErrInvalidCompression,
		},
		{
			name:     "a negative retention rule",
			content:  "repositories:\n  - name: nas\n    url: /a\n    retention:\n      keep_last: -1\n",
			expected: config.ErrInvalidRetention,
		},
		{
			name:     "an invalid keep within",
			content:  "repositories:\n  - name: nas\n    url: /a\n    retention:\n      keep_within: 3y\n",
			expected: config.ErrInvalidRetention,
			message:  "3y",
		},
		{
			name:    "an unknown retention rule",
			content: "repositories:\n  - name: nas\n    url: /a\n    retention:\n      keep_hourly: 3\n",
			message: "keep_hourly",
		},
		{
			name:    "an unknown field",
			content: "repositories:\n  - name: nas\n    url: /a\n    password: secret\n",
			message: "password",
		},
		{
			name:     "no repositories",
			content:  "",
			expected: config.ErrNoRepositories,
		},
	}

	for _, tt := range tests {
		t.Run(`Given a repositories file with `+tt.name+`
		When the configuration is read
		Then it returns an error`, func(t *testing.T) {
			t.Parallel()
			path := repositoriesFile(t, tt.content)

			_, err := config.FromEnvironment(map[string]string{"STOWMARK_CONSOLE_CONFIG": path})

			require.Error(t, err)
			if tt.expected != nil {
				require.ErrorIs(t, err, tt.expected)
			}
			require.ErrorContains(t, err, tt.message)
		})
	}
}

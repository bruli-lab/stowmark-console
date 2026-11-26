//go:build integration

package store_test

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var remoteBackends = []string{"ssh", "s3", "smb", "webdav", "gcs"}

func projectRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)
	return root
}

func loadBackendEnvironment(t *testing.T, root, backend string) map[string]string {
	t.Helper()
	file, err := os.Open(filepath.Join(root, "docker", "env", backend+".env"))
	require.NoError(t, err)
	defer func() { _ = file.Close() }()

	environment := map[string]string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key, value, found := strings.Cut(strings.TrimSpace(scanner.Text()), "=")
		if !found || strings.HasPrefix(key, "#") {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, value)); err == nil {
			value = filepath.Join(root, value)
		}
		environment[key] = value
	}
	require.NoError(t, scanner.Err())

	return environment
}

func TestStoreWithRemoteRepositories(t *testing.T) {
	root := projectRoot(t)
	runID := time.Now().UnixNano()

	for _, backend := range remoteBackends {
		t.Run(`Given a `+backend+` repository started with task env:up
		When the console store is used against it
		Then it behaves like a local repository`, func(t *testing.T) {
			t.Setenv("STOWMARK_LOG_LEVEL", "error")
			environment := loadBackendEnvironment(t, root, backend)
			for key, value := range environment {
				t.Setenv(key, value)
			}

			exerciseRepository(t, fmt.Sprintf("%s-it-%d", environment["STOWMARK_REPOSITORY"], runID))
		})
	}
}

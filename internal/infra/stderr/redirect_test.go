package stderr_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/infra/stderr"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestRedirectTo(t *testing.T) {
	t.Run(`Given a log file path in a missing directory
	When stderr is redirected and a line is written to stderr
	Then the line is stored in the log file and stderr is restored`, func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "logs", "console.log")

		redirection, err := stderr.RedirectTo(path)
		require.NoError(t, err)

		_, err = unix.Write(2, []byte("stowmark log line"))
		require.NoError(t, err)
		require.NoError(t, redirection.Restore())

		content, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Contains(t, string(content), "stowmark log line")
	})
}

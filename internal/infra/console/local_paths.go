package console

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
)

var errPathRequired = errors.New("a path is required")

func defaultRestoreDestination(snapshotID string) string {
	base, err := os.UserHomeDir()
	if err != nil {
		base = os.TempDir()
	}

	return filepath.Join(base, "stowmark-restore", snapshotID)
}

func resolveLocalPath(raw string) (string, error) {
	if raw == "" {
		return "", errPathRequired
	}

	return filepath.Abs(expandHome(raw))
}

func expandHome(raw string) string {
	if raw != "~" && !strings.HasPrefix(raw, "~/") {
		return raw
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return raw
	}

	return filepath.Join(home, strings.TrimPrefix(raw, "~")) + trailingSeparator(raw)
}

func trailingSeparator(raw string) string {
	if strings.HasSuffix(raw, "/") && raw != "~/" {
		return "/"
	}

	return ""
}

func isNonEmptyDirectory(path string) bool {
	entries, err := os.ReadDir(path)
	return err == nil && len(entries) > 0
}

func restoreDestinationWarning(destination string) string {
	if isNonEmptyDirectory(destination) {
		return "The destination is not empty: files with the same path will be overwritten."
	}

	return ""
}

func initRepositoryWarning(profile repository.Profile) string {
	if profile.Local() && isNonEmptyDirectory(profile.Location()) {
		return "The directory already exists and is not empty."
	}

	return ""
}

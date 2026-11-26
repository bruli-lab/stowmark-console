package console

import (
	"errors"
	"fmt"
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

	if raw == "~" || strings.HasPrefix(raw, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		raw = filepath.Join(home, strings.TrimPrefix(raw, "~"))
	}

	return filepath.Abs(raw)
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

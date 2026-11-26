package console

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

const maxPathSuggestions = 200

var (
	errDirectoryNotFound = errors.New("directory not found")
	errNotADirectory     = errors.New("not a directory")
)

func pathSuggestions(value string, known []string) []string {
	suggestions := slices.DeleteFunc(slices.Clone(known), func(origin string) bool { return origin == value })

	prefix, directory := "", "."
	if index := strings.LastIndex(value, "/"); index >= 0 {
		prefix = value[:index+1]
		directory = expandHome(prefix)
	}
	showHidden := strings.HasPrefix(value[len(prefix):], ".")

	entries, err := os.ReadDir(directory)
	if err != nil {
		return suggestions
	}

	for _, entry := range entries {
		if len(suggestions) >= maxPathSuggestions {
			break
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") && !showHidden {
			continue
		}
		suggestion := prefix + name + "/"
		if isDirectoryEntry(filepath.Join(directory, name), entry) && !slices.Contains(suggestions, suggestion) {
			suggestions = append(suggestions, suggestion)
		}
	}

	return suggestions
}

func isDirectoryEntry(path string, entry fs.DirEntry) bool {
	if entry.IsDir() {
		return true
	}

	if entry.Type()&fs.ModeSymlink == 0 {
		return false
	}

	info, err := os.Stat(path)

	return err == nil && info.IsDir()
}

func describeDirectory(value string) (string, error) {
	path, err := resolveLocalPath(value)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "", errDirectoryNotFound
	case err != nil:
		return "", err
	case !info.IsDir():
		return "", errNotADirectory
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Directory with %d entries", len(entries)), nil
}

func knownOrigins(snapshots []snapshot.Snapshot) []string {
	origins := make([]string, 0, len(snapshots))
	for i := range snapshots {
		source := snapshots[i].Source()
		if source == "" {
			continue
		}
		if origin := strings.TrimSuffix(source, "/") + "/"; !slices.Contains(origins, origin) {
			origins = append(origins, origin)
		}
	}

	return origins
}

package snapshot

import (
	"cmp"
	"path/filepath"
	"slices"
	"strings"
)

type File struct {
	path string
	size int64
}

func NewFile(path string, size int64) File {
	return File{path: path, size: size}
}

func (f File) Path() string {
	return f.path
}

func (f File) Size() int64 {
	return f.size
}

type Entry struct {
	name      string
	isDir     bool
	size      int64
	fileCount int
}

func (e Entry) Name() string {
	return e.name
}

func (e Entry) IsDir() bool {
	return e.isDir
}

func (e Entry) Size() int64 {
	return e.size
}

func (e Entry) FileCount() int {
	return e.fileCount
}

type Manifest struct {
	snapshotID string
	source     string
	files      []File
}

func NewManifest(snapshotID, source string, files []File) Manifest {
	return Manifest{snapshotID: snapshotID, source: source, files: slices.Clone(files)}
}

func (m Manifest) SnapshotID() string {
	return m.snapshotID
}

func (m Manifest) Source() string {
	return m.source
}

func (m Manifest) Files() []File {
	return slices.Clone(m.files)
}

func (m Manifest) Entries(directory string) []Entry {
	directory = cleanDirectory(directory)
	byName := map[string]*Entry{}

	for _, file := range m.files {
		rest, ok := m.pathInside(file.path, directory)
		if !ok {
			continue
		}

		name, remainder, nested := strings.Cut(rest, string(filepath.Separator))
		entry, exists := byName[name]
		if !exists {
			entry = &Entry{name: name, isDir: nested && remainder != ""}
			byName[name] = entry
		}

		entry.size += file.size
		entry.fileCount++
	}

	entries := make([]Entry, 0, len(byName))
	for _, entry := range byName {
		entries = append(entries, *entry)
	}

	slices.SortFunc(entries, func(a, b Entry) int {
		if a.isDir != b.isDir {
			if a.isDir {
				return -1
			}
			return 1
		}
		return cmp.Compare(a.name, b.name)
	})

	return entries
}

func (m Manifest) FileAt(directory, name string) (File, bool) {
	relative := filepath.Join(cleanDirectory(directory), name)

	for _, file := range m.files {
		if m.relativePath(file.path) == relative {
			return file, true
		}
	}

	return File{}, false
}

func (m Manifest) relativePath(path string) string {
	relative, err := filepath.Rel(m.source, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return strings.TrimPrefix(filepath.Clean(path), string(filepath.Separator))
	}

	return relative
}

func (m Manifest) pathInside(path, directory string) (string, bool) {
	relative := m.relativePath(path)
	if directory == "" {
		return relative, true
	}

	prefix := directory + string(filepath.Separator)
	if !strings.HasPrefix(relative, prefix) {
		return "", false
	}

	return strings.TrimPrefix(relative, prefix), true
}

func cleanDirectory(directory string) string {
	cleaned := strings.Trim(filepath.Clean(directory), string(filepath.Separator))
	if cleaned == "." {
		return ""
	}

	return cleaned
}

func ParentDirectory(directory string) string {
	parent := filepath.Dir(cleanDirectory(directory))
	if parent == "." {
		return ""
	}

	return parent
}

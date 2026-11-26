package snapshot_test

import (
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
	"github.com/bruli-lab/stowmark-console/internal/fixtures"
	"github.com/stretchr/testify/require"
)

type entryView struct {
	name      string
	isDir     bool
	size      int64
	fileCount int
}

func viewOf(entries []snapshot.Entry) []entryView {
	views := make([]entryView, len(entries))
	for i, entry := range entries {
		views[i] = entryView{entry.Name(), entry.IsDir(), entry.Size(), entry.FileCount()}
	}
	return views
}

func TestManifestEntries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		directory string
		expected  []entryView
	}{
		{
			name:      "the root directory, then it returns directories first with aggregated sizes and then files",
			directory: "",
			expected: []entryView{
				{name: "notes", isDir: true, size: 500, fileCount: 2},
				{name: "README.txt", size: 100, fileCount: 1},
			},
		},
		{
			name:      "a nested directory, then it returns only its direct children",
			directory: "notes",
			expected: []entryView{
				{name: "2026", isDir: true, size: 300, fileCount: 1},
				{name: "todo.md", size: 200, fileCount: 1},
			},
		},
		{
			name:      "a directory written with slashes around, then it is normalized",
			directory: "/notes/2026/",
			expected: []entryView{
				{name: "diary.txt", size: 300, fileCount: 1},
			},
		},
		{
			name:      "a directory that does not exist, then it returns no entries",
			directory: "missing",
			expected:  []entryView{},
		},
	}

	for _, tt := range tests {
		t.Run(`Given a manifest with nested files
		When its entries are listed for `+tt.name, func(t *testing.T) {
			t.Parallel()
			manifest := fixtures.Manifest().Build()

			entries := manifest.Entries(tt.directory)

			require.Equal(t, tt.expected, viewOf(entries))
		})
	}

	t.Run(`Given a manifest with a file outside its source
	When the root entries are listed
	Then the file is shown under its absolute path`, func(t *testing.T) {
		t.Parallel()
		manifest := fixtures.Manifest().
			WithSource("/home/user/documents").
			WithFiles(snapshot.NewFile("/etc/hosts", 10)).
			Build()

		entries := manifest.Entries("")

		require.Equal(t, []entryView{{name: "etc", isDir: true, size: 10, fileCount: 1}}, viewOf(entries))
		require.Equal(t, []entryView{{name: "hosts", size: 10, fileCount: 1}}, viewOf(manifest.Entries("etc")))
	})
}

func TestManifestFileAt(t *testing.T) {
	t.Parallel()

	t.Run(`Given a manifest
	When a file of a directory is looked up
	Then its absolute path in the source is returned`, func(t *testing.T) {
		t.Parallel()
		manifest := fixtures.Manifest().Build()

		file, ok := manifest.FileAt("notes", "todo.md")

		require.True(t, ok)
		require.Equal(t, "/home/user/documents/notes/todo.md", file.Path())
		require.Equal(t, int64(200), file.Size())
	})

	t.Run(`Given a manifest
	When a missing file is looked up
	Then it is not found`, func(t *testing.T) {
		t.Parallel()
		manifest := fixtures.Manifest().Build()

		_, ok := manifest.FileAt("notes", "missing.md")

		require.False(t, ok)
	})
}

func TestParentDirectory(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"notes/2026": "notes",
		"notes":      "",
		"":           "",
	}

	for directory, expected := range tests {
		t.Run(`Given the directory "`+directory+`"
		When its parent is requested
		Then it returns "`+expected+`"`, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, expected, snapshot.ParentDirectory(directory))
		})
	}
}

package snapshot_test

import (
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
	"github.com/stretchr/testify/require"
)

func TestNewFileExtraction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		snapshotID string
		path       string
		directory  string
		expected   error
	}{
		{name: "an empty snapshot id", path: "/data/a.txt", directory: "/tmp/open", expected: snapshot.ErrSnapshotIDRequired},
		{name: "an empty file path", snapshotID: "id", directory: "/tmp/open", expected: snapshot.ErrFilePathRequired},
		{name: "a relative directory", snapshotID: "id", path: "/data/a.txt", directory: "open", expected: snapshot.ErrExtractionDirectoryNotAbsolute},
	}

	for _, tt := range tests {
		t.Run(`Given `+tt.name+`
		When the file extraction is created
		Then it is rejected`, func(t *testing.T) {
			t.Parallel()

			_, err := snapshot.NewFileExtraction(tt.snapshotID, tt.path, tt.directory)

			require.ErrorIs(t, err, tt.expected)
		})
	}

	t.Run(`Given valid values
	When the file extraction is created
	Then it keeps them and has no events yet`, func(t *testing.T) {
		t.Parallel()

		extraction, err := snapshot.NewFileExtraction("id", "/data/a.txt", "/tmp/open")

		require.NoError(t, err)
		require.Equal(t, "id", extraction.SnapshotID())
		require.Equal(t, "/data/a.txt", extraction.Path())
		require.Equal(t, "/tmp/open", extraction.Directory())
		require.Empty(t, extraction.Events())
	})
}

func TestFileExtractionExtracted(t *testing.T) {
	t.Parallel()

	t.Run(`Given a file extraction
	When it is marked as extracted to a location
	Then a file extracted event is recorded`, func(t *testing.T) {
		t.Parallel()
		extraction, err := snapshot.NewFileExtraction("id", "/data/a.txt", "/tmp/open")
		require.NoError(t, err)

		require.NoError(t, extraction.Extracted("/tmp/open/extract-1/a.txt"))

		events := extraction.Events()
		require.Len(t, events, 1)
		extracted, ok := events[0].(snapshot.FileExtracted)
		require.True(t, ok)
		require.Equal(t, snapshot.FileExtractedEventName, extracted.EventName())
		require.Equal(t, "id", extracted.AggregateRootID())
		require.Equal(t, "/data/a.txt", extracted.Path)
		require.Equal(t, "/tmp/open/extract-1/a.txt", extracted.Location)
	})

	t.Run(`Given a file extraction
	When it is marked as extracted without a location
	Then it is rejected and no event is recorded`, func(t *testing.T) {
		t.Parallel()
		extraction, err := snapshot.NewFileExtraction("id", "/data/a.txt", "/tmp/open")
		require.NoError(t, err)

		require.ErrorIs(t, extraction.Extracted(""), snapshot.ErrExtractedLocationRequired)
		require.Empty(t, extraction.Events())
	})
}

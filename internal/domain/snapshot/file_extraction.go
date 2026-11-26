package snapshot

import (
	"errors"
	"path/filepath"

	"github.com/bruli-lab/go-core/event"
)

var (
	ErrFilePathRequired               = errors.New("file path is required")
	ErrExtractionDirectoryNotAbsolute = errors.New("extraction directory must be an absolute path")
	ErrExtractedLocationRequired      = errors.New("extracted location is required")
)

type FileExtraction struct {
	event.BasicAggregateRoot
	snapshotID string
	path       string
	directory  string
}

func NewFileExtraction(snapshotID, path, directory string) (*FileExtraction, error) {
	switch {
	case snapshotID == "":
		return nil, ErrSnapshotIDRequired
	case path == "":
		return nil, ErrFilePathRequired
	case !filepath.IsAbs(directory):
		return nil, ErrExtractionDirectoryNotAbsolute
	}

	return &FileExtraction{
		BasicAggregateRoot: event.NewBasicAggregateRoot(),
		snapshotID:         snapshotID,
		path:               path,
		directory:          directory,
	}, nil
}

func (f *FileExtraction) SnapshotID() string {
	return f.snapshotID
}

func (f *FileExtraction) Path() string {
	return f.path
}

func (f *FileExtraction) Directory() string {
	return f.directory
}

func (f *FileExtraction) Extracted(location string) error {
	if location == "" {
		return ErrExtractedLocationRequired
	}

	f.Record(NewFileExtracted(f.snapshotID, f.path, location))

	return nil
}

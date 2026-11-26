package snapshot

import (
	"errors"
	"path/filepath"
)

var (
	ErrSnapshotIDRequired     = errors.New("snapshot id is required")
	ErrSourceRequired         = errors.New("source is required")
	ErrDestinationRequired    = errors.New("destination is required")
	ErrDestinationNotAbsolute = errors.New("destination must be an absolute path")
)

type Failure struct {
	Path   string
	Reason string
}

func ValidateDestination(destination string) error {
	if destination == "" {
		return ErrDestinationRequired
	}

	if !filepath.IsAbs(destination) {
		return ErrDestinationNotAbsolute
	}

	return nil
}

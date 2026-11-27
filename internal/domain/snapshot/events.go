package snapshot

import (
	"github.com/bruli-lab/go-core/event"
	"github.com/google/uuid"
)

const (
	CreatedEventName       = "snapshot.created"
	VerifiedEventName      = "snapshot.verified"
	RestoredEventName      = "snapshot.restored"
	FileExtractedEventName = "snapshot.file_extracted"
)

type Created struct {
	event.BasicEvent
	Snapshot Snapshot
	Skipped  []SkippedFile
}

func NewCreated(created Snapshot, skipped []SkippedFile) Created {
	return Created{
		BasicEvent: event.NewBasicEvent(CreatedEventName, uuid.New(), created.ID()),
		Snapshot:   created,
		Skipped:    skipped,
	}
}

type Verified struct {
	event.BasicEvent
	SnapshotID string
	Failures   []Failure
}

func NewVerified(snapshotID string, failures []Failure) Verified {
	return Verified{
		BasicEvent: event.NewBasicEvent(VerifiedEventName, uuid.New(), snapshotID),
		SnapshotID: snapshotID,
		Failures:   failures,
	}
}

func (v *Verified) Succeeded() bool {
	return len(v.Failures) == 0
}

type Restored struct {
	event.BasicEvent
	SnapshotID  string
	Destination string
	Failures    []Failure
}

func NewRestored(snapshotID, destination string, failures []Failure) Restored {
	return Restored{
		BasicEvent:  event.NewBasicEvent(RestoredEventName, uuid.New(), snapshotID),
		SnapshotID:  snapshotID,
		Destination: destination,
		Failures:    failures,
	}
}

func (r *Restored) Succeeded() bool {
	return len(r.Failures) == 0
}

type FileExtracted struct {
	event.BasicEvent
	SnapshotID string
	Path       string
	Location   string
}

func NewFileExtracted(snapshotID, path, location string) FileExtracted {
	return FileExtracted{
		BasicEvent: event.NewBasicEvent(FileExtractedEventName, uuid.New(), snapshotID),
		SnapshotID: snapshotID,
		Path:       path,
		Location:   location,
	}
}

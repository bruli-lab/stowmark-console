package snapshot

import "github.com/bruli-lab/go-core/event"

type Restoration struct {
	event.BasicAggregateRoot
	snapshotID  string
	destination string
}

func NewRestoration(snapshotID, destination string) (*Restoration, error) {
	if snapshotID == "" {
		return nil, ErrSnapshotIDRequired
	}

	if err := ValidateDestination(destination); err != nil {
		return nil, err
	}

	return &Restoration{
		BasicAggregateRoot: event.NewBasicAggregateRoot(),
		snapshotID:         snapshotID,
		destination:        destination,
	}, nil
}

func (r *Restoration) SnapshotID() string {
	return r.snapshotID
}

func (r *Restoration) Destination() string {
	return r.destination
}

func (r *Restoration) Restored(failures []Failure) {
	r.Record(NewRestored(r.snapshotID, r.destination, failures))
}

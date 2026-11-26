package snapshot

import "github.com/bruli-lab/go-core/event"

type Verification struct {
	event.BasicAggregateRoot
	snapshotID string
}

func NewVerification(snapshotID string) (*Verification, error) {
	if snapshotID == "" {
		return nil, ErrSnapshotIDRequired
	}

	return &Verification{BasicAggregateRoot: event.NewBasicAggregateRoot(), snapshotID: snapshotID}, nil
}

func (v *Verification) SnapshotID() string {
	return v.snapshotID
}

func (v *Verification) Verified(failures []Failure) {
	v.Record(NewVerified(v.snapshotID, failures))
}

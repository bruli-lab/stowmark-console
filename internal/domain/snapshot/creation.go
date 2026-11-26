package snapshot

import "github.com/bruli-lab/go-core/event"

type Creation struct {
	event.BasicAggregateRoot
	source string
}

func NewCreation(source string) (*Creation, error) {
	if source == "" {
		return nil, ErrSourceRequired
	}

	return &Creation{BasicAggregateRoot: event.NewBasicAggregateRoot(), source: source}, nil
}

func (c *Creation) Source() string {
	return c.source
}

func (c *Creation) Created(created Snapshot) error {
	if created.ID() == "" {
		return ErrSnapshotIDRequired
	}

	c.Record(NewCreated(created))

	return nil
}

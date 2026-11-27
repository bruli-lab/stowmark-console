package snapshot

import (
	"slices"

	"github.com/bruli-lab/go-core/event"
)

type Creation struct {
	event.BasicAggregateRoot
	source     string
	exclusions Exclusions
}

func NewCreation(source string, exclusions Exclusions) (*Creation, error) {
	if source == "" {
		return nil, ErrSourceRequired
	}

	return &Creation{BasicAggregateRoot: event.NewBasicAggregateRoot(), source: source, exclusions: exclusions}, nil
}

func (c *Creation) Source() string {
	return c.source
}

func (c *Creation) Exclusions() Exclusions {
	return c.exclusions
}

func (c *Creation) Created(created Snapshot, skipped []SkippedFile) error {
	if created.ID() == "" {
		return ErrSnapshotIDRequired
	}

	c.Record(NewCreated(created, slices.Clone(skipped)))

	return nil
}

package retention

import (
	"time"

	"github.com/bruli-lab/go-core/event"
)

type ForgottenSnapshot struct {
	ID        string
	CreatedAt time.Time
	Source    string
}

type PostponedSnapshot struct {
	ID            string
	CreatedAt     time.Time
	Source        string
	RetainedUntil time.Time
}

type ForgetResult struct {
	Forgotten []ForgottenSnapshot
	Postponed []PostponedSnapshot
}

type Forgetting struct {
	event.BasicAggregateRoot
	policy Policy
}

func NewForgetting(rules Rules) (*Forgetting, error) {
	policy, err := NewPolicy(rules)
	if err != nil {
		return nil, err
	}

	return &Forgetting{BasicAggregateRoot: event.NewBasicAggregateRoot(), policy: policy}, nil
}

func (f *Forgetting) Policy() Policy {
	return f.policy
}

func (f *Forgetting) Forgot(result ForgetResult) {
	f.Record(NewSnapshotsForgotten(f.policy, result))
}

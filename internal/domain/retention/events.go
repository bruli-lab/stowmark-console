package retention

import (
	"github.com/bruli-lab/go-core/event"
	"github.com/google/uuid"
)

const (
	SnapshotsForgottenEventName = "retention.snapshots_forgotten"
	RepositoryPrunedEventName   = "retention.repository_pruned"
)

type SnapshotsForgotten struct {
	event.BasicEvent
	Policy Policy
	Result ForgetResult
}

func NewSnapshotsForgotten(policy Policy, result ForgetResult) SnapshotsForgotten {
	return SnapshotsForgotten{
		BasicEvent: event.NewBasicEvent(SnapshotsForgottenEventName, uuid.New(), ""),
		Policy:     policy,
		Result:     result,
	}
}

type RepositoryPruned struct {
	event.BasicEvent
	Result PruneResult
}

func NewRepositoryPruned(result *PruneResult) RepositoryPruned {
	return RepositoryPruned{
		BasicEvent: event.NewBasicEvent(RepositoryPrunedEventName, uuid.New(), ""),
		Result:     *result,
	}
}

package retention

import (
	"time"

	"github.com/bruli-lab/go-core/event"
)

type PrunePlan struct {
	StoredObjects       int
	ReferencedObjects   int
	UnreferencedObjects int
	UnreferencedBytes   int64
	MissingObjects      int
	OldKeyGenerations   []uint64
}

func (p PrunePlan) Empty() bool {
	return p.UnreferencedObjects == 0 && len(p.OldKeyGenerations) == 0
}

type PruneResult struct {
	RemovedObjects          int
	FreedBytes              int64
	MissingObjects          int
	RemovedKeyGenerations   []uint64
	RemovedTemporaryFiles   int
	PostponedObjects        int
	PostponedBytes          int64
	PostponedKeyGenerations []uint64
	RetainedUntil           time.Time
}

type Pruning struct {
	event.BasicAggregateRoot
}

func NewPruning() *Pruning {
	return &Pruning{BasicAggregateRoot: event.NewBasicAggregateRoot()}
}

func (p *Pruning) Pruned(result *PruneResult) {
	p.Record(NewRepositoryPruned(result))
}

package snapshot

import (
	"slices"
	"time"
)

type Snapshot struct {
	id        string
	createdAt time.Time
	fileCount int
	totalSize int64
	source    string
}

func New(id string, createdAt time.Time, fileCount int, totalSize int64, source string) Snapshot {
	return Snapshot{
		id:        id,
		createdAt: createdAt,
		fileCount: fileCount,
		totalSize: totalSize,
		source:    source,
	}
}

func (s Snapshot) ID() string {
	return s.id
}

func (s Snapshot) CreatedAt() time.Time {
	return s.createdAt
}

func (s Snapshot) FileCount() int {
	return s.fileCount
}

func (s Snapshot) TotalSize() int64 {
	return s.totalSize
}

func (s Snapshot) Source() string {
	return s.source
}

func SortByNewest(snapshots []Snapshot) []Snapshot {
	sorted := slices.Clone(snapshots)
	slices.SortStableFunc(sorted, func(a, b Snapshot) int {
		return b.createdAt.Compare(a.createdAt)
	})

	return sorted
}

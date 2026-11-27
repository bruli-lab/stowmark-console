package integrity

import (
	"context"
	"slices"
)

type Problem struct {
	ID     string
	Reason string
}

type MissingObject struct {
	Hash      string
	Snapshots []string
}

type Report struct {
	Scope               Scope
	Snapshots           int
	CorruptManifests    []Problem
	StoredObjects       int
	ReferencedObjects   int
	UnreferencedObjects int
	MissingObjects      []MissingObject
	StaleLocks          int
	OldKeyGenerations   []uint64
	ReadObjects         int
	CorruptObjects      []Problem
	UnprotectedItems    int
}

func (r *Report) HasErrors() bool {
	return len(r.CorruptManifests) > 0 || len(r.MissingObjects) > 0 || len(r.CorruptObjects) > 0
}

func (r *Report) HasWarnings() bool {
	return r.UnreferencedObjects > 0 || r.StaleLocks > 0 || len(r.OldKeyGenerations) > 0 || r.UnprotectedItems > 0
}

func (r *Report) Prunable() bool {
	return r.UnreferencedObjects > 0 || len(r.OldKeyGenerations) > 0
}

func (r *Report) DamagedSnapshots() []string {
	var damaged []string
	for _, problem := range r.CorruptManifests {
		damaged = append(damaged, problem.ID)
	}
	for _, missing := range r.MissingObjects {
		damaged = append(damaged, missing.Snapshots...)
	}

	slices.Sort(damaged)

	return slices.Compact(damaged)
}

type Checker interface {
	Check(ctx context.Context, scope Scope) (Report, error)
}

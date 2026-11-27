package store

import (
	"context"

	"github.com/bruli-lab/stowmark-console/internal/domain/integrity"
	"github.com/bruli-lab/stowmark/pkg/stowmark"
)

type Checker struct {
	session *Session
}

func NewChecker(session *Session) *Checker {
	return &Checker{session: session}
}

func (c *Checker) Check(ctx context.Context, scope integrity.Scope) (integrity.Report, error) {
	handler, err := c.session.current()
	if err != nil {
		return integrity.Report{}, err
	}

	result, err := handler.Check(ctx, stowmark.CheckOptions{
		ReadData:       scope.ReadsAllData(),
		ReadDataSubset: scope.Subset(),
	})
	if err != nil {
		return integrity.Report{}, translate(err)
	}

	missing := make([]integrity.MissingObject, len(result.MissingObjects))
	for i, object := range result.MissingObjects {
		missing[i] = integrity.MissingObject{Hash: object.Hash, Snapshots: object.Snapshots}
	}

	return integrity.Report{
		Scope:               scope,
		Snapshots:           result.Snapshots,
		CorruptManifests:    problems(result.CorruptManifests),
		StoredObjects:       result.StoredObjects,
		ReferencedObjects:   result.ReferencedObjects,
		UnreferencedObjects: result.UnreferencedObjects,
		MissingObjects:      missing,
		StaleLocks:          result.StaleLocks,
		OldKeyGenerations:   result.OldKeyGenerations,
		ReadObjects:         result.ReadObjects,
		CorruptObjects:      problems(result.CorruptObjects),
		UnprotectedItems:    result.UnprotectedItems,
	}, nil
}

func problems(found []stowmark.CheckProblem) []integrity.Problem {
	converted := make([]integrity.Problem, len(found))
	for i, problem := range found {
		converted[i] = integrity.Problem{ID: problem.ID, Reason: problem.Reason}
	}

	return converted
}

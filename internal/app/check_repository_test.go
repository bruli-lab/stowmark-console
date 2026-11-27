package app_test

import (
	"context"
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/app"
	"github.com/bruli-lab/stowmark-console/internal/domain/integrity"
	"github.com/bruli-lab/stowmark-console/internal/mocks"
	"github.com/stretchr/testify/require"
)

func TestCheckRepository(t *testing.T) {
	t.Parallel()

	t.Run(`Given a query that is not a check repository query
	When it is handled
	Then an invalid query error is returned`, func(t *testing.T) {
		t.Parallel()

		requireInvalidQuery(t, app.NewCheckRepository(&mocks.CheckerMock{}))
	})

	t.Run(`Given a scope
	When the repository is checked
	Then the scope reaches the checker and the report carries it`, func(t *testing.T) {
		t.Parallel()
		checker := &mocks.CheckerMock{
			CheckFunc: func(context.Context, integrity.Scope) (integrity.Report, error) {
				return integrity.Report{Snapshots: 3}, nil
			},
		}

		result, err := app.NewCheckRepository(checker).Handle(t.Context(), app.CheckRepositoryQuery{Scope: integrity.AllData()})

		require.NoError(t, err)
		report, ok := result.(integrity.Report)
		require.True(t, ok)
		require.Equal(t, 3, report.Snapshots)
		require.True(t, report.Scope.ReadsAllData())
		require.True(t, checker.CheckCalls()[0].Scope.ReadsAllData())
	})

	t.Run(`Given a checker that fails
	When the repository is checked
	Then the error is returned`, func(t *testing.T) {
		t.Parallel()
		checker := &mocks.CheckerMock{
			CheckFunc: func(context.Context, integrity.Scope) (integrity.Report, error) {
				return integrity.Report{}, errBackend
			},
		}

		_, err := app.NewCheckRepository(checker).Handle(t.Context(), app.CheckRepositoryQuery{})

		require.ErrorIs(t, err, errBackend)
	})
}

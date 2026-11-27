package integrity_test

import (
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/domain/integrity"
	"github.com/stretchr/testify/require"
)

func TestScope(t *testing.T) {
	t.Parallel()

	t.Run(`Given the structure only scope
	When it is described
	Then it reads no data`, func(t *testing.T) {
		t.Parallel()

		scope := integrity.StructureOnly()

		require.False(t, scope.ReadsData())
		require.Equal(t, "structure", scope.String())
	})

	t.Run(`Given the all data scope
	When it is described
	Then it reads every object`, func(t *testing.T) {
		t.Parallel()

		scope := integrity.AllData()

		require.True(t, scope.ReadsData())
		require.True(t, scope.ReadsAllData())
		require.Equal(t, "structure and all data", scope.String())
	})

	for _, value := range []string{"10%", "0.5%", "100%", "2/7", " 1/1 "} {
		t.Run(`Given the subset `+value+`
		When a data subset scope is built
		Then it is accepted`, func(t *testing.T) {
			t.Parallel()

			scope, err := integrity.DataSubset(value)

			require.NoError(t, err)
			require.True(t, scope.ReadsData())
			require.False(t, scope.ReadsAllData())
			require.Contains(t, scope.String(), scope.Subset()+" of the data")
		})
	}

	for _, value := range []string{"", "0%", "101%", "x%", "NaN%", "0/3", "4/3", "a/b", "10"} {
		t.Run(`Given the subset "`+value+`"
		When a data subset scope is built
		Then it is rejected`, func(t *testing.T) {
			t.Parallel()

			_, err := integrity.DataSubset(value)

			require.ErrorIs(t, err, integrity.ErrInvalidSubset)
		})
	}
}

func TestReport(t *testing.T) {
	t.Parallel()

	t.Run(`Given a clean report
	When it is checked
	Then it has neither errors nor warnings`, func(t *testing.T) {
		t.Parallel()
		report := integrity.Report{Snapshots: 2, StoredObjects: 5, ReferencedObjects: 5}

		require.False(t, report.HasErrors())
		require.False(t, report.HasWarnings())
		require.False(t, report.Prunable())
		require.Empty(t, report.DamagedSnapshots())
	})

	t.Run(`Given a report with corrupt manifests, missing and corrupt objects
	When it is checked
	Then it has errors and lists each damaged snapshot once`, func(t *testing.T) {
		t.Parallel()
		report := integrity.Report{
			CorruptManifests: []integrity.Problem{{ID: "c", Reason: "bad"}},
			MissingObjects: []integrity.MissingObject{
				{Hash: "h1", Snapshots: []string{"b", "a"}},
				{Hash: "h2", Snapshots: []string{"a"}},
			},
		}

		require.True(t, report.HasErrors())
		require.Equal(t, []string{"a", "b", "c"}, report.DamagedSnapshots())
	})

	t.Run(`Given a report with unreferenced objects, stale locks, old generations and unprotected items
	When it is checked
	Then it has warnings and can be pruned`, func(t *testing.T) {
		t.Parallel()

		require.True(t, (&integrity.Report{UnreferencedObjects: 1}).Prunable())
		require.True(t, (&integrity.Report{OldKeyGenerations: []uint64{1}}).Prunable())
		require.True(t, (&integrity.Report{StaleLocks: 1}).HasWarnings())
		require.False(t, (&integrity.Report{StaleLocks: 1}).Prunable())
		require.True(t, (&integrity.Report{UnprotectedItems: 1}).HasWarnings())
		require.True(t, (&integrity.Report{CorruptObjects: []integrity.Problem{{ID: "h"}}}).HasErrors())
	})
}

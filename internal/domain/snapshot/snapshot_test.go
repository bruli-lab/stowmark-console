package snapshot_test

import (
	"testing"
	"time"

	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
	"github.com/bruli-lab/stowmark-console/internal/fixtures"
	"github.com/stretchr/testify/require"
)

func TestSortByNewest(t *testing.T) {
	t.Parallel()

	t.Run(`Given snapshots in any order
	When they are sorted by newest
	Then the most recent comes first and the input is not modified`, func(t *testing.T) {
		t.Parallel()
		base := time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)
		oldest := fixtures.Snapshot().WithID("oldest").WithCreatedAt(base).Build()
		newest := fixtures.Snapshot().WithID("newest").WithCreatedAt(base.Add(2 * time.Hour)).Build()
		middle := fixtures.Snapshot().WithID("middle").WithCreatedAt(base.Add(time.Hour)).Build()
		input := []snapshot.Snapshot{oldest, newest, middle}

		sorted := snapshot.SortByNewest(input)

		require.Equal(t, []snapshot.Snapshot{newest, middle, oldest}, sorted)
		require.Equal(t, []snapshot.Snapshot{oldest, newest, middle}, input)
	})
}

func TestValidateDestination(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		destination string
		expectedErr error
	}{
		{name: "an empty destination, then it is required", destination: "", expectedErr: snapshot.ErrDestinationRequired},
		{name: "a relative destination, then it must be absolute", destination: "restore", expectedErr: snapshot.ErrDestinationNotAbsolute},
		{name: "an absolute destination, then it is valid", destination: "/tmp/restore"},
	}

	for _, tt := range tests {
		t.Run(`Given a restore destination
		When it is validated with `+tt.name, func(t *testing.T) {
			t.Parallel()

			err := snapshot.ValidateDestination(tt.destination)

			require.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

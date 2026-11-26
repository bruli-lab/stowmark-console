package snapshot_test

import (
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
	"github.com/bruli-lab/stowmark-console/internal/fixtures"
	"github.com/stretchr/testify/require"
)

func TestCreation(t *testing.T) {
	t.Parallel()

	t.Run(`Given an empty source
	When the creation is built
	Then it is rejected`, func(t *testing.T) {
		t.Parallel()

		_, err := snapshot.NewCreation("")

		require.ErrorIs(t, err, snapshot.ErrSourceRequired)
	})

	t.Run(`Given a creation
	When it is marked as created with a snapshot
	Then a created event for that snapshot is recorded`, func(t *testing.T) {
		t.Parallel()
		creation, err := snapshot.NewCreation("/data")
		require.NoError(t, err)
		require.Equal(t, "/data", creation.Source())
		require.Empty(t, creation.Events())
		created := fixtures.Snapshot().WithID("snap").WithSource("/data").Build()

		require.NoError(t, creation.Created(created))

		events := creation.Events()
		require.Len(t, events, 1)
		event, ok := events[0].(snapshot.Created)
		require.True(t, ok)
		require.Equal(t, snapshot.CreatedEventName, event.EventName())
		require.Equal(t, "snap", event.AggregateRootID())
		require.Equal(t, created, event.Snapshot)
	})

	t.Run(`Given a creation
	When it is marked as created with a snapshot without id
	Then it is rejected and no event is recorded`, func(t *testing.T) {
		t.Parallel()
		creation, err := snapshot.NewCreation("/data")
		require.NoError(t, err)

		require.ErrorIs(t, creation.Created(snapshot.Snapshot{}), snapshot.ErrSnapshotIDRequired)
		require.Empty(t, creation.Events())
	})
}

func TestVerification(t *testing.T) {
	t.Parallel()

	t.Run(`Given an empty snapshot id
	When the verification is built
	Then it is rejected`, func(t *testing.T) {
		t.Parallel()

		_, err := snapshot.NewVerification("")

		require.ErrorIs(t, err, snapshot.ErrSnapshotIDRequired)
	})

	t.Run(`Given a verification
	When it is marked as verified with failures
	Then a verified event with the failures is recorded`, func(t *testing.T) {
		t.Parallel()
		verification, err := snapshot.NewVerification("snap")
		require.NoError(t, err)
		require.Equal(t, "snap", verification.SnapshotID())
		failures := []snapshot.Failure{{Path: "/data/a.txt", Reason: "hash mismatch"}}

		verification.Verified(failures)

		events := verification.Events()
		require.Len(t, events, 1)
		event, ok := events[0].(snapshot.Verified)
		require.True(t, ok)
		require.Equal(t, snapshot.VerifiedEventName, event.EventName())
		require.Equal(t, "snap", event.AggregateRootID())
		require.Equal(t, failures, event.Failures)
		require.False(t, event.Succeeded())
	})
}

func TestRestoration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		snapshotID  string
		destination string
		expected    error
	}{
		{name: "an empty snapshot id", destination: "/restore", expected: snapshot.ErrSnapshotIDRequired},
		{name: "no destination", snapshotID: "snap", expected: snapshot.ErrDestinationRequired},
		{name: "a relative destination", snapshotID: "snap", destination: "restore", expected: snapshot.ErrDestinationNotAbsolute},
	}

	for _, tt := range tests {
		t.Run(`Given `+tt.name+`
		When the restoration is built
		Then it is rejected`, func(t *testing.T) {
			t.Parallel()

			_, err := snapshot.NewRestoration(tt.snapshotID, tt.destination)

			require.ErrorIs(t, err, tt.expected)
		})
	}

	t.Run(`Given a restoration
	When it is marked as restored without failures
	Then a successful restored event is recorded`, func(t *testing.T) {
		t.Parallel()
		restoration, err := snapshot.NewRestoration("snap", "/restore")
		require.NoError(t, err)
		require.Equal(t, "snap", restoration.SnapshotID())
		require.Equal(t, "/restore", restoration.Destination())

		restoration.Restored(nil)

		events := restoration.Events()
		require.Len(t, events, 1)
		event, ok := events[0].(snapshot.Restored)
		require.True(t, ok)
		require.Equal(t, snapshot.RestoredEventName, event.EventName())
		require.Equal(t, "snap", event.AggregateRootID())
		require.Equal(t, "/restore", event.Destination)
		require.True(t, event.Succeeded())
	})
}

package progress_test

import (
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/domain/progress"
	"github.com/stretchr/testify/require"
)

func TestUpdate(t *testing.T) {
	t.Parallel()

	t.Run(`Given a started stage with bytes
	When it advances
	Then the fraction follows the bytes and the items are counted`, func(t *testing.T) {
		t.Parallel()

		update := progress.Started(progress.StageBackingUp, 4, 200).Advanced(1, 0).Advanced(0, 50)

		require.Equal(t, 1, update.Items)
		require.Equal(t, int64(50), update.Bytes)
		require.InDelta(t, 0.25, update.Fraction(), 0.0001)
		require.True(t, update.MeasuresBytes())
	})

	t.Run(`Given a started stage without bytes
	When it advances
	Then the fraction follows the items`, func(t *testing.T) {
		t.Parallel()

		update := progress.Started(progress.StageRekeying, 10, 0).Advanced(4, 0)

		require.InDelta(t, 0.4, update.Fraction(), 0.0001)
		require.False(t, update.MeasuresBytes())
	})

	t.Run(`Given a stage without totals
	When its fraction is asked
	Then it is zero until it is completed`, func(t *testing.T) {
		t.Parallel()

		update := progress.Started(progress.StageRetaining, 0, 0)

		require.Zero(t, update.Fraction())
		require.InDelta(t, 1, update.Completed().Fraction(), 0.0001)
	})

	t.Run(`Given a stage that advanced past its totals
	When its fraction is asked
	Then it is capped at one`, func(t *testing.T) {
		t.Parallel()

		update := progress.Started(progress.StageVerifying, 1, 10).Advanced(2, 20)

		require.InDelta(t, 1, update.Fraction(), 0.0001)
	})

	t.Run(`Given every stage
	When its unit is asked
	Then file stages count files and the others objects`, func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "files", progress.StageBackingUp.Unit())
		require.Equal(t, "files", progress.StageRestoring.Unit())
		require.Equal(t, "files", progress.StageVerifying.Unit())
		require.Equal(t, "objects", progress.StageRekeying.Unit())
		require.Equal(t, "objects", progress.StageReadingData.Unit())
	})
}

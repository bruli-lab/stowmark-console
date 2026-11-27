package snapshot_test

import (
	"testing"

	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
	"github.com/stretchr/testify/require"
)

func TestExclusions(t *testing.T) {
	t.Parallel()

	t.Run(`Given patterns with blanks and repetitions
	When the exclusions are built
	Then they keep each pattern once, trimmed and in order`, func(t *testing.T) {
		t.Parallel()

		exclusions := snapshot.NewExclusions([]string{" *.tmp ", "", "node_modules/", "*.tmp"}, false)

		require.Equal(t, []string{"*.tmp", "node_modules/"}, exclusions.Patterns())
		require.False(t, exclusions.SkipsIgnoreFiles())
		require.False(t, exclusions.Empty())
	})

	t.Run(`Given no patterns
	When the exclusions skip the ignore files or not
	Then they are empty only when they read the ignore files`, func(t *testing.T) {
		t.Parallel()

		require.True(t, snapshot.NewExclusions(nil, false).Empty())
		require.False(t, snapshot.NewExclusions(nil, true).Empty())
	})

	t.Run(`Given a text with patterns separated by commas
	When it is parsed
	Then it returns the patterns`, func(t *testing.T) {
		t.Parallel()

		require.Equal(t, []string{"*.log", "cache/", "!keep.log"}, snapshot.ParseExcludePatterns("*.log, cache/ ,, !keep.log"))
		require.Empty(t, snapshot.ParseExcludePatterns("  "))
	})

	t.Run(`Given skipped files of several kinds
	When they are counted by kind
	Then each kind has its count`, func(t *testing.T) {
		t.Parallel()

		kinds := snapshot.SkippedKinds([]snapshot.SkippedFile{
			{Path: "a.sock", Kind: "socket"},
			{Path: "b.sock", Kind: "socket"},
			{Path: "tty", Kind: "device"},
		})

		require.Equal(t, map[string]int{"socket": 2, "device": 1}, kinds)
	})
}

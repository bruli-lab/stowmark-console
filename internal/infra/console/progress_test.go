package console

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/bruli-lab/stowmark-console/internal/domain/progress"
	"github.com/stretchr/testify/require"
)

type fakeClock struct {
	now time.Time
}

func (c *fakeClock) read() time.Time {
	return c.now
}

func (h *harness) withClock() *fakeClock {
	clock := &fakeClock{now: time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC)}
	h.model.clock = clock.read
	return clock
}

func (h *harness) report(update progress.Update) {
	h.model.Update(progressReportedMsg{update: update})
}

func TestConsoleProgress(t *testing.T) {
	t.Parallel()

	t.Run(`Given a running verification
	When its progress is reported
	Then the footer shows the stage, a bar, the files, the bytes, the speed and the remaining time`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		clock := h.withClock()
		h.start(runes("v"))
		h.report(progress.Started(progress.StageVerifying, 4, 2048))
		clock.now = clock.now.Add(10 * time.Second)

		h.report(progress.Started(progress.StageVerifying, 4, 2048).Advanced(2, 1024))

		footer := h.model.renderFooter()
		require.Contains(t, footer, "Verifying")
		require.Contains(t, footer, " 50%")
		require.Contains(t, footer, "2/4 files")
		require.Contains(t, footer, "1.0 KiB/2.0 KiB")
		require.Contains(t, footer, "102 B/s")
		require.Contains(t, footer, "ETA 10s")
		require.Contains(t, footer, "[esc] Cancel")
	})

	t.Run(`Given a running key rotation
	When its progress is reported without bytes
	Then the footer counts objects and their speed`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		clock := h.withClock()
		h.start(runes("v"))
		h.report(progress.Started(progress.StageRekeying, 100, 0))
		clock.now = clock.now.Add(2 * time.Second)

		h.report(progress.Started(progress.StageRekeying, 100, 0).Advanced(40, 0))

		footer := h.model.renderFooter()
		require.Contains(t, footer, "40/100 objects")
		require.Contains(t, footer, "20 objects/s")
		require.Contains(t, footer, "ETA 3s")
		require.NotContains(t, footer, "B/s")
	})

	t.Run(`Given a stage that was running
	When a new stage starts
	Then its timing starts again`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		clock := h.withClock()
		h.start(runes("v"))
		h.report(progress.Started(progress.StageVerifying, 2, 0).Advanced(2, 0).Completed())
		clock.now = clock.now.Add(time.Minute)

		h.report(progress.Started(progress.StageRestoring, 2, 0))

		require.Equal(t, clock.now, h.model.transfer.started)
		require.NotContains(t, h.model.renderFooter(), "files/s")
	})

	t.Run(`Given no running operation
	When a progress update arrives
	Then it is ignored`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()

		h.report(progress.Started(progress.StageVerifying, 2, 0))

		require.Nil(t, h.model.transfer)
		require.NotContains(t, h.model.renderFooter(), "Verifying")
	})

	t.Run(`Given an operation with progress
	When the user cancels it
	Then the footer says it is cancelling and ignores later progress`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		h.start(runes("v"))
		h.report(progress.Started(progress.StageVerifying, 2, 0))

		h.press(key(tea.KeyEscape))
		h.report(progress.Started(progress.StageVerifying, 2, 0).Advanced(1, 0))

		require.Contains(t, h.model.renderFooter(), "Cancelling...")
		require.NotContains(t, h.model.renderFooter(), "1/2 files")
	})

	t.Run(`Given an operation with progress
	When it finishes
	Then the progress is cleared`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()
		operation := h.start(runes("v"))
		h.report(progress.Started(progress.StageVerifying, 2, 0))

		h.run(operation)

		require.Nil(t, h.model.transfer)
		require.NotContains(t, h.model.renderFooter(), "Verifying")
	})
}

func TestProgressFeed(t *testing.T) {
	t.Parallel()

	t.Run(`Given a feed nobody reads
	When several updates are reported
	Then reporting does not block and the reader gets the latest one`, func(t *testing.T) {
		t.Parallel()
		feed := NewProgressFeed()

		for items := range 5 {
			feed.Report(progress.Started(progress.StageBackingUp, 10, 0).Advanced(items, 0))
		}

		msg, ok := feed.next()().(progressReportedMsg)
		require.True(t, ok)
		require.Equal(t, 4, msg.update.Items)
	})

	t.Run(`Given a console with a feed
	When it starts
	Then it waits for progress and keeps waiting after each update`, func(t *testing.T) {
		t.Parallel()
		feed := NewProgressFeed()
		h := newHarness(t, true)
		WithProgressFeed(feed)(h.model)
		feed.Report(progress.Started(progress.StageBackingUp, 1, 0))

		_, cmd := h.model.Update(feed.next()())

		require.NotNil(t, h.model.Init())
		require.NotNil(t, cmd)
	})
}

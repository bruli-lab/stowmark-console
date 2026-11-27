package console

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
	domain "github.com/bruli-lab/stowmark-console/internal/domain/progress"
)

const (
	progressBarWidth = 24
	minEstimateTime  = time.Second
)

type ProgressFeed struct {
	mu     sync.Mutex
	latest domain.Update
	signal chan struct{}
}

func NewProgressFeed() *ProgressFeed {
	return &ProgressFeed{signal: make(chan struct{}, 1)}
}

func (f *ProgressFeed) Report(update domain.Update) {
	f.mu.Lock()
	f.latest = update
	f.mu.Unlock()

	select {
	case f.signal <- struct{}{}:
	default:
	}
}

func (f *ProgressFeed) next() tea.Cmd {
	return func() tea.Msg {
		<-f.signal

		f.mu.Lock()
		defer f.mu.Unlock()

		return progressReportedMsg{update: f.latest}
	}
}

type progressReportedMsg struct {
	update domain.Update
}

func WithProgressFeed(feed *ProgressFeed) Option {
	return func(m *Model) {
		m.feed = feed
	}
}

type transfer struct {
	update  domain.Update
	started time.Time
}

func (m *Model) applyProgress(update domain.Update) {
	if !m.loading || m.cancelling {
		return
	}

	if m.transfer == nil || m.transfer.restartedBy(update) {
		m.transfer = &transfer{started: m.clock()}
	}
	m.transfer.update = update
}

func (t *transfer) restartedBy(update domain.Update) bool {
	return update.Stage != t.update.Stage || update.Items < t.update.Items || update.Bytes < t.update.Bytes
}

func (t *transfer) view(now time.Time) string {
	update := t.update
	fraction := update.Fraction()
	bar := progress.New(
		progress.WithWidth(progressBarWidth),
		progress.WithoutPercentage(),
		progress.WithColors(colorPrimary, colorSecondary),
	)

	parts := []string{string(update.Stage), bar.ViewAs(fraction), fmt.Sprintf("%3.0f%%", fraction*100)}

	if update.TotalItems > 0 {
		parts = append(parts, fmt.Sprintf("%d/%d %s", update.Items, update.TotalItems, update.Stage.Unit()))
	}

	if update.MeasuresBytes() {
		parts = append(parts, formatSize(update.Bytes)+"/"+formatSize(update.TotalBytes))
	}

	elapsed := now.Sub(t.started)
	if elapsed >= minEstimateTime {
		parts = append(parts, speed(update, elapsed))
		if fraction > 0 && fraction < 1 {
			remaining := time.Duration(float64(elapsed) * (1 - fraction) / fraction)
			parts = append(parts, "ETA "+remaining.Round(time.Second).String())
		}
	}

	return strings.Join(parts, "  ")
}

func speed(update domain.Update, elapsed time.Duration) string {
	if update.MeasuresBytes() {
		return formatSize(int64(float64(update.Bytes)/elapsed.Seconds())) + "/s"
	}

	return fmt.Sprintf("%.0f %s/s", float64(update.Items)/elapsed.Seconds(), update.Stage.Unit())
}

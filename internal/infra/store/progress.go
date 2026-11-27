package store

import (
	"sync"

	"github.com/bruli-lab/stowmark-console/internal/domain/progress"
	"github.com/bruli-lab/stowmark/pkg/stowmark"
)

type progressReporter struct {
	mu       sync.Mutex
	listener progress.Listener
	current  progress.Update
}

func newProgressReporter(listener progress.Listener) *progressReporter {
	return &progressReporter{listener: listener}
}

func (p *progressReporter) Start(stage stowmark.ProgressStage, items int, bytes int64) {
	p.report(func(progress.Update) progress.Update {
		return progress.Started(progress.Stage(stage), items, bytes)
	})
}

func (p *progressReporter) Advance(items int, bytes int64) {
	p.report(func(current progress.Update) progress.Update { return current.Advanced(items, bytes) })
}

func (p *progressReporter) Finish() {
	p.report(progress.Update.Completed)
}

func (p *progressReporter) report(change func(progress.Update) progress.Update) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current = change(p.current)
	p.listener.Report(p.current)
}

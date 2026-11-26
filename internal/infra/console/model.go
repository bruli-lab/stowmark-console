package console

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/bruli-lab/stowmark-console/internal/app"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

type view int

const (
	snapshotsView view = iota
	filesView
)

type Model struct {
	ctx             context.Context
	handlers        app.Handlers
	settings        Settings
	view            view
	snapshots       []snapshot.Snapshot
	snapshotCursor  int
	manifest        snapshot.Manifest
	directory       string
	entries         []snapshot.Entry
	fileCursor      int
	preview         *filePreview
	width           int
	height          int
	loading         bool
	cancelOperation context.CancelFunc
	err             error
	commandMode     bool
	commandInput    string
	message         string
	preserveMessage bool
	confirmingInit  bool
	initWarning     string
	snapshotForm    *textPrompt
	restoreForm     *textPrompt
	restore         *restoreRequest
}

type filePreview struct {
	name      string
	lines     []string
	offset    int
	truncated bool
}

type restoreRequest struct {
	snapshotID  string
	destination string
	warning     string
}

func New(ctx context.Context, settings Settings, handlers app.Handlers) Model {
	return Model{
		ctx:      ctx,
		handlers: handlers,
		settings: settings,
		loading:  true,
	}
}

func (m *Model) confirmingRestore() bool {
	return m.restore != nil && m.restoreForm == nil
}

func (m *Model) selectedSnapshot() (snapshot.Snapshot, bool) {
	if m.snapshotCursor < 0 || m.snapshotCursor >= len(m.snapshots) {
		return snapshot.Snapshot{}, false
	}

	return m.snapshots[m.snapshotCursor], true
}

func (m *Model) currentSnapshotID() (string, bool) {
	if m.view == filesView {
		return m.manifest.SnapshotID(), m.manifest.SnapshotID() != ""
	}

	selected, ok := m.selectedSnapshot()

	return selected.ID(), ok
}

func (m *Model) selectedEntry() (snapshot.Entry, bool) {
	if m.fileCursor < 0 || m.fileCursor >= len(m.entries) {
		return snapshot.Entry{}, false
	}

	return m.entries[m.fileCursor], true
}

func (m *Model) showDirectory(directory, selectName string) {
	m.directory = directory
	m.entries = m.manifest.Entries(directory)
	m.fileCursor = 0

	for index, entry := range m.entries {
		if entry.Name() == selectName {
			m.fileCursor = index
			return
		}
	}
}

func (m *Model) enterSelectedEntry() {
	entry, ok := m.selectedEntry()
	if !ok || !entry.IsDir() {
		return
	}

	m.showDirectory(filepath.Join(m.directory, entry.Name()), "")
}

func (m *Model) leaveDirectory() {
	if m.directory == "" {
		m.view = snapshotsView
		m.manifest = snapshot.Manifest{}
		m.entries = nil
		return
	}

	m.showDirectory(snapshot.ParentDirectory(m.directory), filepath.Base(m.directory))
}

func (m *Model) selectSnapshot(snapshotID string) {
	for index, item := range m.snapshots {
		if item.ID() == snapshotID {
			m.snapshotCursor = index
			return
		}
	}

	m.snapshotCursor = min(m.snapshotCursor, max(0, len(m.snapshots)-1))
}

func (m *Model) openPreview(name string, content []byte, truncated bool) {
	text := strings.TrimSuffix(strings.ToValidUTF8(string(content), ""), "\n")
	m.preview = &filePreview{
		name:      name,
		lines:     strings.Split(strings.ReplaceAll(text, "\t", "    "), "\n"),
		truncated: truncated,
	}
}

func (m *Model) contentHeight() int {
	return max(1, m.height-2-contentStyle.GetVerticalFrameSize())
}

func (m *Model) previewVisibleLines() int {
	return max(1, m.contentHeight()-2)
}

func (m *Model) maxPreviewOffset() int {
	if m.preview == nil {
		return 0
	}

	return max(0, len(m.preview.lines)-m.previewVisibleLines())
}

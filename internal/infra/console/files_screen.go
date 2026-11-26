package console

import (
	"fmt"
	"path"
	"path/filepath"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

var entryColumns = []columnSpec{
	{title: "NAME"},
	{title: "TYPE", width: 14},
	{title: "SIZE", width: 10, alignRight: true},
}

type filesScreen struct {
	manifest  snapshot.Manifest
	directory string
	entries   []snapshot.Entry
	table     table.Model
	width     int
}

func newFilesScreen(manifest snapshot.Manifest) *filesScreen {
	screen := &filesScreen{manifest: manifest, table: newTable()}
	screen.show("", "")

	return screen
}

func (s *filesScreen) name() string {
	return "Files"
}

func (s *filesScreen) snapshotID() string {
	return s.manifest.SnapshotID()
}

func (s *filesScreen) location() string {
	return s.snapshotID() + "/" + s.directory
}

func (s *filesScreen) help() string {
	return "↑↓ Navigate   ↵ Preview   [o] Open   [esc] Back   [v] Verify   [x] Restore   [r] Refresh"
}

func (s *filesScreen) resize(width, height int) {
	s.width = width
	resizeTable(&s.table, entryColumns, width, height)
}

func (s *filesScreen) setManifest(manifest snapshot.Manifest) {
	selected, _ := s.selected()
	s.manifest = manifest
	s.show(s.directory, selected.Name())
}

func (s *filesScreen) show(directory, selectName string) {
	s.directory = directory
	s.entries = s.manifest.Entries(directory)

	cursor := 0
	rows := make([]table.Row, len(s.entries))

	for i, entry := range s.entries {
		entryType := "file"
		if entry.IsDir() {
			entryType = fmt.Sprintf("dir (%d)", entry.FileCount())
		}
		rows[i] = tableRow(entryColumns, entry.Name(), entryType, formatSize(entry.Size()))
		if entry.Name() == selectName {
			cursor = i
		}
	}

	selectRow(&s.table, rows, cursor)
}

func (s *filesScreen) selected() (snapshot.Entry, bool) {
	cursor := s.table.Cursor()
	if cursor < 0 || cursor >= len(s.entries) {
		return snapshot.Entry{}, false
	}

	return s.entries[cursor], true
}

func (s *filesScreen) update(msg tea.KeyPressMsg) tea.Cmd {
	if cmd, handled := snapshotActionKey(msg.String(), s.snapshotID()); handled {
		return cmd
	}

	switch msg.String() {
	case "enter", "right", "l":
		return s.open()

	case "esc", "backspace", "left", "h":
		if s.directory == "" {
			return emit(screenCloseRequested{})
		}
		s.show(snapshot.ParentDirectory(s.directory), filepath.Base(s.directory))
		return nil

	case "r":
		return emit(snapshotOpenRequested{snapshotID: s.snapshotID()})

	case "o":
		request, ok := s.selectedFile()
		if !ok {
			return nil
		}
		return emit(fileOpenRequested{request: request})
	}

	var cmd tea.Cmd
	s.table, cmd = s.table.Update(msg)

	return cmd
}

func (s *filesScreen) open() tea.Cmd {
	entry, ok := s.selected()
	if !ok {
		return nil
	}

	if entry.IsDir() {
		s.show(filepath.Join(s.directory, entry.Name()), "")
		return nil
	}

	request, found := s.selectedFile()
	if !found {
		return nil
	}

	return emit(request)
}

func (s *filesScreen) selectedFile() (filePreviewRequested, bool) {
	entry, ok := s.selected()
	if !ok || entry.IsDir() {
		return filePreviewRequested{}, false
	}

	file, found := s.manifest.FileAt(s.directory, entry.Name())
	if !found {
		return filePreviewRequested{}, false
	}

	return filePreviewRequested{
		snapshotID: s.snapshotID(),
		location:   path.Join(s.snapshotID(), s.directory, entry.Name()),
		name:       entry.Name(),
		file:       file,
	}, true
}

func (s *filesScreen) view() string {
	title := truncate(headerStyle.Render("Snapshot "+s.snapshotID())+"  "+
		mutedStyle.Render(filepath.Join(s.manifest.Source(), s.directory)), s.width)

	if len(s.entries) == 0 {
		return title + "\n\nThis directory is empty."
	}

	return title + "\n\n" + s.table.View()
}

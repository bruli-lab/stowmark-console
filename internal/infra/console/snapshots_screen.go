package console

import (
	"fmt"
	"strconv"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

const dateTimeFormat = "2006-01-02 15:04:05"

var snapshotColumns = []columnSpec{
	{title: "CREATED", width: 19},
	{title: "ID", width: 26},
	{title: "FILES", width: 7, alignRight: true},
	{title: "SIZE", width: 10, alignRight: true},
	{title: "SOURCE"},
}

type snapshotsScreen struct {
	snapshots []snapshot.Snapshot
	table     table.Model
}

func newSnapshotsScreen() *snapshotsScreen {
	return &snapshotsScreen{table: newTable()}
}

func (s *snapshotsScreen) name() string {
	return "Snapshots"
}

func (s *snapshotsScreen) location() string {
	return ""
}

func (s *snapshotsScreen) help() string {
	return "↑↓ Navigate   ↵ Open   [c] Create   [v] Verify   [x] Restore   [r] Refresh"
}

func (s *snapshotsScreen) resize(width, height int) {
	resizeTable(&s.table, snapshotColumns, width, height)
}

func (s *snapshotsScreen) setSnapshots(snapshots []snapshot.Snapshot, selectID string) {
	cursor := s.table.Cursor()
	rows := make([]table.Row, len(snapshots))

	for i, item := range snapshots {
		rows[i] = tableRow(snapshotColumns,
			item.CreatedAt().Local().Format(dateTimeFormat),
			item.ID(),
			strconv.Itoa(item.FileCount()),
			formatSize(item.TotalSize()),
			item.Source(),
		)
		if item.ID() == selectID {
			cursor = i
		}
	}

	s.snapshots = snapshots
	selectRow(&s.table, rows, cursor)
}

func (s *snapshotsScreen) selected() (snapshot.Snapshot, bool) {
	cursor := s.table.Cursor()
	if cursor < 0 || cursor >= len(s.snapshots) {
		return snapshot.Snapshot{}, false
	}

	return s.snapshots[cursor], true
}

func (s *snapshotsScreen) update(msg tea.KeyPressMsg) tea.Cmd {
	selected, ok := s.selected()

	if cmd, handled := snapshotActionKey(msg.String(), selected.ID()); handled {
		return cmd
	}

	switch msg.String() {
	case "enter", "right", "l":
		if !ok {
			return nil
		}
		return emit(snapshotOpenRequested{snapshotID: selected.ID()})

	case "r":
		return emit(snapshotsRefreshRequested{selectID: selected.ID()})
	}

	var cmd tea.Cmd
	s.table, cmd = s.table.Update(msg)

	return cmd
}

func (s *snapshotsScreen) view() string {
	title := headerStyle.Render(fmt.Sprintf("Snapshots (%d)", len(s.snapshots)))

	if len(s.snapshots) == 0 {
		return title + "\n\nNo snapshots yet. Press [c] to create one."
	}

	return title + "\n\n" + s.table.View()
}

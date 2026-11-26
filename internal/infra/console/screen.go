package console

import (
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
)

const (
	cellPadding       = 2
	minFlexibleWidth  = 8
	screenTitleHeight = 2
)

type screen interface {
	name() string
	location() string
	help() string
	resize(width, height int)
	update(msg tea.KeyPressMsg) tea.Cmd
	view() string
}

type columnSpec struct {
	title      string
	width      int
	alignRight bool
}

func newTable() table.Model {
	return table.New(
		table.WithFocused(true),
		table.WithStyles(table.Styles{
			Header:   tableHeaderStyle,
			Cell:     tableCellStyle,
			Selected: tableSelectedStyle,
		}),
	)
}

func resizeTable(model *table.Model, specs []columnSpec, width, height int) {
	model.SetColumns(fitColumns(specs, width))
	model.SetWidth(width)
	model.SetHeight(max(1, height-screenTitleHeight))
}

func fitColumns(specs []columnSpec, width int) []table.Column {
	columns := make([]table.Column, len(specs))
	remaining := width
	fits := true

	for i, spec := range specs {
		columns[i].Title = spec.title
		if spec.width == 0 {
			continue
		}
		fits = fits && spec.width+cellPadding <= remaining
		if fits {
			columns[i].Width = spec.width
			remaining -= spec.width + cellPadding
		}
	}

	for i, spec := range specs {
		if spec.width == 0 && remaining-cellPadding >= minFlexibleWidth {
			columns[i].Width = remaining - cellPadding
		}
	}

	return columns
}

func tableRow(specs []columnSpec, cells ...string) table.Row {
	row := make(table.Row, len(cells))
	for i, cell := range cells {
		if specs[i].alignRight {
			cell = padLeft(cell, specs[i].width)
		}
		row[i] = cell
	}

	return row
}

func selectRow(model *table.Model, rows []table.Row, cursor int) {
	model.SetRows(rows)
	model.SetCursor(min(max(cursor, 0), max(0, len(rows)-1)))
}

func snapshotActionKey(key, snapshotID string) (tea.Cmd, bool) {
	switch key {
	case "c":
		return emit(snapshotCreateRequested{}), true
	case "v":
		if snapshotID == "" {
			return nil, true
		}
		return emit(snapshotVerifyRequested{snapshotID: snapshotID}), true
	case "x":
		if snapshotID == "" {
			return nil, true
		}
		return emit(snapshotRestoreRequested{snapshotID: snapshotID}), true
	}

	return nil, false
}

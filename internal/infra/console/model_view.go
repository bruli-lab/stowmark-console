package console

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const sidebarTotalWidth = 28

type column struct {
	title      string
	width      int
	alignRight bool
}

func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Starting Stowmark Console..."
	}

	switch {
	case m.confirmingInit:
		return m.renderDialog(m.renderInitConfirmation())
	case m.snapshotForm != nil:
		return m.renderDialog(m.snapshotForm.render("[Enter] Create  [Esc] Cancel"))
	case m.restoreForm != nil:
		return m.renderDialog(m.restoreForm.render("[Enter] Continue  [Esc] Cancel"))
	case m.confirmingRestore():
		return m.renderDialog(m.renderRestoreConfirmation())
	default:
		return m.renderLayout()
	}
}

func (m *Model) renderLayout() string {
	sidebarWidth := max(1, sidebarTotalWidth-sidebarStyle.GetHorizontalFrameSize())
	contentWidth := max(1, m.width-sidebarTotalWidth-contentStyle.GetHorizontalFrameSize())
	bodyHeight := max(1, m.height-2)

	sidebar := sidebarStyle.
		Width(sidebarWidth).
		Height(max(1, bodyHeight-sidebarStyle.GetVerticalFrameSize())).
		Render(m.renderSidebar(bodyHeight, sidebarWidth-sidebarStyle.GetHorizontalPadding()))

	content := contentStyle.
		Width(contentWidth).
		Height(m.contentHeight()).
		Render(m.renderContent(contentWidth - contentStyle.GetHorizontalPadding()))

	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)

	return lipgloss.JoinVertical(lipgloss.Left, m.renderHeader(), body, m.renderFooter())
}

func (m *Model) renderHeader() string {
	title := titleStyle.Render("STOWMARK") + " " + kindStyle.Render(repositoryKind(m.settings.RepositoryPath))
	location := m.settings.RepositoryPath

	if m.view == filesView {
		location += "  ›  " + m.manifest.SnapshotID() + "/" + m.directory
	}

	maxLocationWidth := max(1, m.width-lipgloss.Width(title)-2)
	locationText := repositoryStyle.Render(truncate(location, maxLocationWidth))
	spaces := max(1, m.width-lipgloss.Width(title)-lipgloss.Width(locationText)-1)

	return title + strings.Repeat(" ", spaces) + locationText
}

func (m *Model) renderFooter() string {
	var help string

	switch {
	case m.commandMode:
		help = ":" + m.commandInput
	case m.message != "":
		help = m.message
	case m.loading:
		help = "[esc] Cancel"
	case m.preview != nil:
		help = "↑↓ Scroll   [pgup/pgdown] Page   [g/G] Top/Bottom   [esc] Close   [:q] Exit"
	case m.view == filesView:
		help = "↑↓ Navigate   ↵ Open   [esc] Back   [v] Verify   [x] Restore   [r] Refresh   [:q] Exit"
	default:
		help = "↑↓ Navigate   ↵ Open   [c] Create   [v] Verify   [x] Restore   [r] Refresh   [:q] Exit"
	}

	return footerStyle.Width(m.width).Render(truncate(help, max(1, m.width-2)))
}

func (m *Model) renderSidebar(availableHeight, width int) string {
	var builder strings.Builder

	if availableHeight >= 24 {
		builder.WriteString(logoStyle.Render(stowmarkLogo))
		builder.WriteString("\n\n")
	}

	builder.WriteString(headerStyle.Render("Navigation"))
	builder.WriteString("\n\n")
	builder.WriteString(renderMenuItem("Snapshots", m.view == snapshotsView))
	builder.WriteString(renderMenuItem("Files", m.view == filesView))

	if selected, ok := m.sidebarSnapshot(); ok {
		builder.WriteString("\n")
		builder.WriteString(headerStyle.Render("Snapshot"))
		builder.WriteString("\n\n")
		builder.WriteString(mutedStyle.Render("ID") + "\n" + truncate(selected.id, width) + "\n")
		builder.WriteString(mutedStyle.Render("Created") + "\n" + truncate(selected.created, width) + "\n")
		builder.WriteString(mutedStyle.Render("Files") + "\n" + truncate(selected.files, width) + "\n")
		builder.WriteString(mutedStyle.Render("Source") + "\n" + truncate(selected.source, width) + "\n")
	}

	return builder.String()
}

type snapshotSummary struct {
	id      string
	created string
	files   string
	source  string
}

func (m *Model) sidebarSnapshot() (snapshotSummary, bool) {
	selected, ok := m.selectedSnapshot()
	if !ok {
		return snapshotSummary{}, false
	}

	return snapshotSummary{
		id:      selected.ID(),
		created: selected.CreatedAt().Local().Format("2006-01-02 15:04:05"),
		files:   fmt.Sprintf("%d (%s)", selected.FileCount(), formatSize(selected.TotalSize())),
		source:  selected.Source(),
	}, true
}

func renderMenuItem(label string, selected bool) string {
	if selected {
		return selectedMenuStyle.Render("› "+label) + "\n"
	}

	return menuStyle.Render("  "+label) + "\n"
}

func (m *Model) renderContent(width int) string {
	switch {
	case m.preview != nil:
		return m.renderPreview(width)
	case m.view == filesView:
		return m.renderFiles(width)
	default:
		return m.renderSnapshots(width)
	}
}

func (m *Model) renderStatus(builder *strings.Builder) bool {
	if m.err != nil {
		builder.WriteString(errorStyle.Render(m.err.Error()))
		builder.WriteString("\n\n")
	}

	if m.loading {
		builder.WriteString(mutedStyle.Render("Working..."))
		return true
	}

	return false
}

func (m *Model) renderSnapshots(width int) string {
	var builder strings.Builder

	builder.WriteString(headerStyle.Render(fmt.Sprintf("Snapshots (%d)", len(m.snapshots))))
	builder.WriteString("\n\n")

	if m.renderStatus(&builder) {
		return builder.String()
	}

	if len(m.snapshots) == 0 {
		builder.WriteString("No snapshots yet. Press [c] to create one.")
		return builder.String()
	}

	columns := []column{
		{title: "CREATED", width: 19},
		{title: "ID", width: 26},
		{title: "FILES", width: 7, alignRight: true},
		{title: "SIZE", width: 10, alignRight: true},
		{title: "SOURCE", width: 0},
	}

	rows := make([][]string, len(m.snapshots))
	for i, item := range m.snapshots {
		rows[i] = []string{
			item.CreatedAt().Local().Format("2006-01-02 15:04:05"),
			item.ID(),
			fmt.Sprintf("%d", item.FileCount()),
			formatSize(item.TotalSize()),
			item.Source(),
		}
	}

	builder.WriteString(m.renderTable(columns, rows, m.snapshotCursor, width))

	return builder.String()
}

func (m *Model) renderFiles(width int) string {
	var builder strings.Builder

	title := "Snapshot " + m.manifest.SnapshotID()
	builder.WriteString(headerStyle.Render(title))
	builder.WriteString("  ")
	builder.WriteString(mutedStyle.Render(filepath.Join(m.manifest.Source(), m.directory)))
	builder.WriteString("\n\n")

	if m.renderStatus(&builder) {
		return builder.String()
	}

	if len(m.entries) == 0 {
		builder.WriteString("This directory is empty.")
		return builder.String()
	}

	columns := []column{
		{title: "NAME", width: 0},
		{title: "TYPE", width: 14},
		{title: "SIZE", width: 10, alignRight: true},
	}

	rows := make([][]string, len(m.entries))
	for i, entry := range m.entries {
		entryType := "file"
		if entry.IsDir() {
			entryType = fmt.Sprintf("dir (%d)", entry.FileCount())
		}
		rows[i] = []string{entry.Name(), entryType, formatSize(entry.Size())}
	}

	builder.WriteString(m.renderTable(columns, rows, m.fileCursor, width))

	return builder.String()
}

func (m *Model) renderTable(columns []column, rows [][]string, cursor, width int) string {
	const gap = 2

	fixed := 2
	for _, col := range columns {
		fixed += col.width + gap
	}
	flexible := max(8, width-fixed)

	format := func(cells []string) string {
		parts := make([]string, len(columns))
		for i, col := range columns {
			cellWidth := col.width
			if cellWidth == 0 {
				cellWidth = flexible
			}
			if col.alignRight {
				parts[i] = padLeft(cells[i], cellWidth)
			} else {
				parts[i] = padRight(cells[i], cellWidth)
			}
		}
		return strings.Join(parts, strings.Repeat(" ", gap))
	}

	titles := make([]string, len(columns))
	for i, col := range columns {
		titles[i] = col.title
	}

	var builder strings.Builder
	builder.WriteString(mutedStyle.Render("  " + format(titles)))
	builder.WriteString("\n")

	maxRows := max(1, m.contentHeight()-4)
	start := max(0, cursor-maxRows+1)
	end := min(len(rows), start+maxRows)

	for index := start; index < end; index++ {
		row := format(rows[index])
		if index == cursor {
			builder.WriteString(selectedRowStyle.Render("› " + row))
		} else {
			builder.WriteString("  " + row)
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

func (m *Model) renderPreview(width int) string {
	var builder strings.Builder

	builder.WriteString(headerStyle.Render(m.preview.name))
	if m.preview.truncated {
		builder.WriteString("  ")
		builder.WriteString(mutedStyle.Render(fmt.Sprintf("(showing the first %s)", formatSize(maxPreviewSize))))
	}
	builder.WriteString("\n\n")

	lines := m.preview.lines
	start := min(m.preview.offset, len(lines))
	end := min(len(lines), start+m.previewVisibleLines())
	numberWidth := len(fmt.Sprintf("%d", len(lines)))
	lineWidth := max(1, width-numberWidth-3)

	for index := start; index < end; index++ {
		builder.WriteString(mutedStyle.Render(fmt.Sprintf("%*d │ ", numberWidth, index+1)))
		builder.WriteString(truncate(lines[index], lineWidth))
		builder.WriteString("\n")
	}

	return builder.String()
}

func (m *Model) renderDialog(content string) string {
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialogStyle.Render(content))
}

func (m *Model) renderInitConfirmation() string {
	var builder strings.Builder

	builder.WriteString(headerStyle.Render("Initialize repository"))
	builder.WriteString("\n\nNo Stowmark repository found at:\n")
	builder.WriteString(highlightStyle.Render(m.settings.RepositoryPath))
	builder.WriteString("\n\nA new repository will be created with:")
	builder.WriteString("\n  Encryption:  " + encryptionSummary(m.settings.Init))
	builder.WriteString("\n  Compression: " + compressionSummary(m.settings.Init))

	if m.initWarning != "" {
		builder.WriteString("\n\n")
		builder.WriteString(errorStyle.Render(m.initWarning))
	}

	builder.WriteString("\n\n[y] Initialize  [n/Esc] Quit")

	return builder.String()
}

func (m *Model) renderRestoreConfirmation() string {
	var builder strings.Builder

	builder.WriteString(headerStyle.Render("Restore snapshot " + m.restore.snapshotID + "?"))
	builder.WriteString("\n\nDestination:\n")
	builder.WriteString(highlightStyle.Render(m.restore.destination))

	if m.restore.warning != "" {
		builder.WriteString("\n\n")
		builder.WriteString(errorStyle.Render(m.restore.warning))
	}

	builder.WriteString("\n\n[y] Confirm  [n/Esc] Cancel")

	return builder.String()
}

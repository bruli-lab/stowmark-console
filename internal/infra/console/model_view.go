package console

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
	header := m.renderHeader()
	footer := m.renderFooter()

	bodyHeight := max(
		1,
		m.height-lipgloss.Height(header)-lipgloss.Height(footer),
	)

	const sidebarTotalWidth = 28

	sidebarFrameWidth := sidebarStyle.GetHorizontalFrameSize()
	sidebarFrameHeight := sidebarStyle.GetVerticalFrameSize()
	contentFrameWidth := contentStyle.GetHorizontalFrameSize()
	contentFrameHeight := contentStyle.GetVerticalFrameSize()

	sidebarWidth := max(1, sidebarTotalWidth-sidebarFrameWidth)
	contentWidth := max(1, m.width-sidebarTotalWidth-contentFrameWidth)
	sidebarHeight := max(1, bodyHeight-sidebarFrameHeight)
	contentHeight := max(1, bodyHeight-contentFrameHeight)

	sidebar := sidebarStyle.
		Width(sidebarWidth).
		Height(sidebarHeight).
		Render(m.renderSidebar(sidebarHeight))

	content := contentStyle.
		Width(contentWidth).
		Height(contentHeight).
		Render(m.renderContent(contentHeight))

	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m *Model) renderDialog(content string) string {
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		dialogStyle.Render(content),
	)
}

func (m *Model) renderInitConfirmation() string {
	var builder strings.Builder

	builder.WriteString(headerStyle.Render("Initialize repository"))
	builder.WriteString("\n\nNo Stowmark repository found at:\n")
	builder.WriteString(highlightStyle.Render(m.repoPath))
	builder.WriteString("\n\nA new repository will be created with:")
	builder.WriteString("\n  Encryption:  " + m.initSettings.encryptionSummary())
	builder.WriteString("\n  Compression: " + m.initSettings.compressionSummary())

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

func (m *Model) renderHeader() string {
	title := titleStyle.Render("STOWMARK")

	repositoryPath := repositoryStyle.Render(m.currentPath)

	spaces := max(
		1,
		m.width-lipgloss.Width(title)-lipgloss.Width(repositoryPath)-2,
	)

	return title + strings.Repeat(" ", spaces) + repositoryPath
}

func (m *Model) renderFooter() string {
	var help string

	if m.isSnapshotsDirectory() {
		help = "↑↓ Navigate   ↵ Open   [c] Create   [v] Verify   [x] Restore   [esc] Back   [:q] Exit"
	} else {
		help = "↑↓ Navigate   ↵ Open   [r] Refresh   [:q] Exit"
	}

	if m.commandMode {
		help = ":" + m.commandInput
	} else if m.message != "" {
		help = m.message
	}

	return footerStyle.
		Width(m.width).
		Render(help)
}

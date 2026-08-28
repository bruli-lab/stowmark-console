package console

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Starting Stowmark Console..."
	}

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

	sidebarWidth := max(
		1,
		sidebarTotalWidth-sidebarFrameWidth,
	)
	contentWidth := max(
		1,
		m.width-sidebarTotalWidth-contentFrameWidth,
	)

	sidebarHeight := max(
		1,
		bodyHeight-sidebarFrameHeight,
	)
	contentHeight := max(
		1,
		bodyHeight-contentFrameHeight,
	)

	sidebar := sidebarStyle.
		Width(sidebarWidth).
		Height(sidebarHeight).
		Render(m.renderSidebar(sidebarHeight))

	content := contentStyle.
		Width(contentWidth).
		Height(contentHeight).
		Render(m.renderContent(contentHeight))

	body := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sidebar,
		content,
	)

	view := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		body,
		footer,
	)

	if m.confirmingRestore {
		confirmation := confirmRestoreStyle.Render(
			fmt.Sprintf(
				"Restore snapshot %s?\n\n[y/Enter] Confirm  [n/Esc] Cancel",
				m.pendingSnapshotID,
			),
		)

		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			confirmation,
		)
	}

	if m.creatingSnapshot {
		origin := m.snapshotOrigin
		if origin == "" {
			origin = " "
		}

		var form strings.Builder
		form.WriteString(headerStyle.Render("Create snapshot"))
		form.WriteString("\n\nOrigin\n")
		form.WriteString(snapshotInputStyle.Render(origin + "█"))

		if m.snapshotFormError != "" {
			form.WriteString("\n\n")
			form.WriteString(errorStyle.Render(m.snapshotFormError))
		}

		form.WriteString("\n\n[Enter] Create  [Esc] Cancel")

		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			snapshotFormStyle.Render(form.String()),
		)
	}

	return view
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
		help = "↑↓ Navigate   ↵ Open   [e] Edit   [r] Refresh   [:q] Exit"
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

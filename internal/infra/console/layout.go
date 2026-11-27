package console

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
)

const (
	lockBadge         = "🔒"
	minDialogWidth    = 76
	sidebarTotalWidth = 28
	minLogoHeight     = 32
	maxErrorLines     = 3
)

var menuScreens = []string{"Repositories", "Snapshots", "Files", "Preview"}

func (m *Model) View() tea.View {
	view := tea.NewView(m.render())
	view.AltScreen = true

	return view
}

func (m *Model) render() string {
	if m.width == 0 || m.height == 0 {
		return "Starting Stowmark Console..."
	}

	if m.splash != nil {
		return m.splash.view(m.width, m.height)
	}

	if m.dialog != nil {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, renderDialog(m.dialog.view(), m.width))
	}

	m.resizeScreens()

	sections := []string{m.renderHeader(), m.renderBody()}
	if m.err != nil {
		sections = append(sections, m.renderError())
	}
	sections = append(sections, m.renderFooter())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func renderDialog(content string, maxWidth int) string {
	width := max(minDialogWidth, lipgloss.Width(content)+dialogStyle.GetHorizontalFrameSize())

	return dialogStyle.Width(min(width, maxWidth)).Render(content)
}

func (m *Model) contentSize() (width, height int) {
	width = max(1, m.width-sidebarTotalWidth-contentStyle.GetHorizontalFrameSize())
	height = max(1, m.bodyHeight()-contentStyle.GetVerticalFrameSize())

	return width, height
}

func (m *Model) bodyHeight() int {
	errorHeight := 0
	if m.err != nil {
		errorHeight = lipgloss.Height(m.renderError())
	}

	return max(1, m.height-2-errorHeight)
}

func (m *Model) renderBody() string {
	height := m.bodyHeight()
	sidebarWidth := sidebarTotalWidth - sidebarStyle.GetHorizontalFrameSize()

	sidebar := sidebarStyle.
		Width(sidebarTotalWidth).
		Height(height).
		MaxHeight(height).
		Render(m.renderSidebar(height, sidebarWidth))

	content := contentStyle.
		Width(max(1, m.width-sidebarTotalWidth)).
		Height(height).
		MaxHeight(height).
		Render(m.top().view())

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
}

func (m *Model) renderHeader() string {
	title := titleStyle.Render("STOWMARK")
	if m.current.Name() != "" {
		title += " " + kindStyle.Render(m.current.Kind())
		if m.current.Encrypted() {
			title += " " + kindStyle.Render(lockBadge)
		}
		title += " " + highlightStyle.Render(m.current.Name())
	}
	location := m.current.Location()

	if current := m.top().location(); current != "" {
		location += "  ›  " + current
	}

	maxLocationWidth := max(1, m.width-lipgloss.Width(title)-2)
	locationText := repositoryStyle.Render(truncate(location, maxLocationWidth))
	spaces := max(1, m.width-lipgloss.Width(title)-lipgloss.Width(locationText)-1)

	return title + strings.Repeat(" ", spaces) + locationText
}

func (m *Model) renderError() string {
	return errorBlockStyle.Width(m.width).MaxHeight(maxErrorLines).Render(m.err.Error())
}

func (m *Model) renderFooter() string {
	var help string

	switch {
	case m.command != nil:
		help = m.command.view()
	case m.loading:
		help = m.progress + "   [esc] Cancel"
	case m.message != "":
		help = m.message
	default:
		help = m.top().help() + "   [:q] Exit"
	}

	return footerStyle.Width(m.width).Render(truncate(help, max(1, m.width-footerStyle.GetHorizontalFrameSize())))
}

func (m *Model) renderSidebar(availableHeight, width int) string {
	var builder strings.Builder

	if availableHeight >= minLogoHeight {
		builder.WriteString(logoStyle.Render(stowmarkLogo()))
		builder.WriteString("\n\n")
	}

	builder.WriteString(headerStyle.Render("Navigation"))
	builder.WriteString("\n\n")

	current := m.top().name()
	for _, item := range menuScreens {
		builder.WriteString(renderMenuItem(item, item == current))
	}

	if m.connected() && m.current.Encrypted() {
		writeEncryption(&builder, m.current, width)
	}

	if selected, ok := m.snapshots.selected(); ok {
		builder.WriteString("\n")
		builder.WriteString(headerStyle.Render("Snapshot"))
		builder.WriteString("\n\n")
		writeDetail(&builder, "ID", selected.ID(), width)
		writeDetail(&builder, "Created", selected.CreatedAt().Local().Format(dateTimeFormat), width)
		writeDetail(&builder, "Files", fmt.Sprintf("%d (%s)", selected.FileCount(), formatSize(selected.TotalSize())), width)
		writeDetail(&builder, "Source", selected.Source(), width)
	}

	return builder.String()
}

func writeEncryption(builder *strings.Builder, profile repository.Profile, width int) {
	builder.WriteString("\n")
	builder.WriteString(headerStyle.Render(lockBadge + " Encrypted"))
	builder.WriteString("\n\n")

	key := errorStyle.Render("not set")
	if profile.PrivateKey() != nil {
		key = truncateLeft(filepath.Base(*profile.PrivateKey()), width)
	}
	builder.WriteString(mutedStyle.Render("Private key") + "\n" + key + "\n")

	if fingerprint := profile.Fingerprint(); fingerprint != "" {
		writeDetail(builder, "Fingerprint", fingerprint, width)
	}
}

func writeDetail(builder *strings.Builder, label, value string, width int) {
	builder.WriteString(mutedStyle.Render(label))
	builder.WriteString("\n")
	builder.WriteString(truncate(value, width))
	builder.WriteString("\n")
}

func renderMenuItem(label string, selected bool) string {
	if selected {
		return selectedMenuStyle.Render("› "+label) + "\n"
	}

	return menuStyle.Render("  "+label) + "\n"
}

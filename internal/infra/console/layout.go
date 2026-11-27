package console

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
)

const (
	lockBadge             = "🔒"
	minDialogWidth        = 76
	panelLines            = 6
	panelColumnGap        = 4
	minColumnWidth        = 16
	maxColumnWidth        = 42
	minLogoTerminalHeight = 45
	detailLabelSize       = 13
	maxErrorLines         = 3
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
		return m.splash.view(m.version, m.width, m.height)
	}

	if m.dialog != nil {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, renderDialog(m.dialog.view(), m.width))
	}

	m.resizeScreens()

	sections := []string{m.renderHeader(), m.renderPanel(), m.renderBody()}
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
	width = max(1, m.width-contentStyle.GetHorizontalFrameSize())
	height = max(1, m.bodyHeight()-contentStyle.GetVerticalFrameSize())

	return width, height
}

func (m *Model) bodyHeight() int {
	errorHeight := 0
	if m.err != nil {
		errorHeight = lipgloss.Height(m.renderError())
	}

	return max(1, m.height-2-m.panelHeight()-panelStyle.GetVerticalFrameSize()-errorHeight)
}

func (m *Model) renderBody() string {
	height := m.bodyHeight()

	return contentStyle.
		Width(m.width).
		Height(height).
		MaxHeight(height).
		Render(m.top().view())
}

func (m *Model) renderHeader() string {
	title := titleStyle.Render("STOWMARK")
	if m.version != "" {
		title += " " + mutedStyle.Render(m.version)
	}
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
	case m.loading && m.transfer != nil && !m.cancelling:
		help = m.transfer.view(m.clock()) + "   [esc] Cancel"
	case m.loading:
		help = m.activity + "   [esc] Cancel"
	case m.message != "":
		help = m.message
	default:
		help = m.top().help() + "   [:] Commands   [:q] Exit"
	}

	return footerStyle.Width(m.width).Render(truncate(help, max(1, m.width-footerStyle.GetHorizontalFrameSize())))
}

func (m *Model) renderPanel() string {
	width := m.panelWidth()
	if !m.showLogo() {
		parts, _ := m.panelParts(width, panelLines)
		return panelStyle.Width(m.width).Render(lipgloss.JoinHorizontal(lipgloss.Top, parts...))
	}

	logo := stowmarkLogo()
	height := lipgloss.Height(logo)
	parts, _ := m.panelParts(width-lipgloss.Width(logo)-panelColumnGap, height)
	used := lipgloss.Width(lipgloss.JoinHorizontal(lipgloss.Top, parts...))
	parts = append(parts, panelGap(width-used-lipgloss.Width(logo), height), logo)

	return panelStyle.Width(m.width).Render(lipgloss.JoinHorizontal(lipgloss.Top, parts...))
}

func (m *Model) panelWidth() int {
	return m.width - panelStyle.GetHorizontalFrameSize()
}

func (m *Model) panelHeight() int {
	if m.showLogo() {
		return max(panelLines, lipgloss.Height(stowmarkLogo()))
	}

	return panelLines
}

func (m *Model) showLogo() bool {
	if m.height < minLogoTerminalHeight {
		return false
	}

	width := m.panelWidth()
	_, all := m.panelParts(width, panelLines)
	_, withLogo := m.panelParts(width-lipgloss.Width(stowmarkLogo())-panelColumnGap, panelLines)

	return withLogo > 0 && withLogo == all
}

func (m *Model) panelParts(width, height int) (parts []string, columns int) {
	used := 0

	for _, column := range m.panelColumns() {
		gap := 0
		if columns > 0 {
			gap = panelColumnGap
		}
		remaining := width - used - gap
		if remaining < minColumnWidth {
			break
		}
		if gap > 0 {
			parts = append(parts, panelGap(gap, height))
		}
		rendered := renderPanelColumn(column, min(remaining, maxColumnWidth), height)
		parts = append(parts, rendered)
		used += gap + lipgloss.Width(rendered)
		columns++
	}

	return parts, columns
}

func (m *Model) panelColumns() [][]string {
	encrypted := m.connected() && m.current.Encrypted()
	columns := append([][]string{navigationColumn(m.top().name())}, commandsColumns(encrypted)...)

	if encrypted {
		columns = append(columns, encryptionColumn(m.current))
	}

	if selected, ok := m.snapshots.selected(); ok {
		columns = append(columns, []string{
			headerStyle.Render("Snapshot"),
			"",
			detailLine("ID", selected.ID()),
			detailLine("Created", selected.CreatedAt().Local().Format(dateTimeFormat)),
			detailLine("Files", fmt.Sprintf("%d (%s)", selected.FileCount(), formatSize(selected.TotalSize()))),
			detailLine("Source", selected.Source()),
		})
	}

	return columns
}

func navigationColumn(current string) []string {
	lines := []string{headerStyle.Render("Navigation"), ""}
	for _, item := range menuScreens {
		lines = append(lines, renderMenuItem(item, item == current))
	}

	return lines
}

func commandsColumns(encrypted bool) [][]string {
	var columns [][]string
	header := headerStyle.Render("Commands")

	for chunk := range slices.Chunk(listedCommands(encrypted), panelLines-2) {
		lines := []string{header, ""}
		for _, command := range chunk {
			lines = append(lines, fmt.Sprintf(":%-7s ", command.name())+mutedStyle.Render(command.summary))
		}
		columns = append(columns, lines)
		header = ""
	}

	return columns
}

func encryptionColumn(profile repository.Profile) []string {
	key := errorStyle.Render("not set")
	if profile.PrivateKey() != nil {
		key = filepath.Base(*profile.PrivateKey())
	}
	lines := []string{headerStyle.Render(lockBadge + " Encrypted"), "", detailLine("Private key", key)}

	if fingerprint := profile.Fingerprint(); fingerprint != "" {
		lines = append(lines, detailLine("Fingerprint", fingerprint))
	}

	return lines
}

func detailLine(label, value string) string {
	return mutedStyle.Render(fmt.Sprintf("%-*s", detailLabelSize, label)) + value
}

func renderPanelColumn(lines []string, maxWidth, height int) string {
	for i, line := range lines {
		lines[i] = truncate(line, maxWidth)
	}

	return panelColumnStyle.Height(height).Render(strings.Join(lines, "\n"))
}

func panelGap(width, height int) string {
	return panelColumnStyle.Width(max(0, width)).Height(height).Render("")
}

func renderMenuItem(label string, selected bool) string {
	if selected {
		return selectedMenuStyle.Render("› " + label)
	}

	return menuStyle.Render("  " + label)
}

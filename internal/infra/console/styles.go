package console

import "github.com/charmbracelet/lipgloss"

const sidebarContentWidth = 24

var (
	colorPrimary   = lipgloss.Color("#7C5CFC")
	colorSecondary = lipgloss.Color("#5B9BD5")
	colorMuted     = lipgloss.Color("#6C7086")
	colorText      = lipgloss.Color("#CDD6F4")
	colorSurface   = lipgloss.Color("#181825")
	colorSelected  = lipgloss.Color("#313244")
	colorError     = lipgloss.Color("#F38BA8")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorText).
			Background(colorPrimary).
			Padding(0, 1)

	repositoryStyle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	logoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#21E6D7")).
			Background(colorSurface).
			Bold(true).
			Align(lipgloss.Center).
			Width(sidebarContentWidth)

	sidebarStyle = lipgloss.NewStyle().
			Background(colorSurface).
			Foreground(colorText).
			Padding(1, 2)

	selectedMenuStyle = lipgloss.NewStyle().
				Background(colorSelected).
				Foreground(colorText).
				Bold(true).
				Padding(0, 1).
				Width(sidebarContentWidth - 2)

	menuStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Background(colorSurface).
			Padding(0, 1).
			Width(sidebarContentWidth - 2)

	contentStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Padding(1, 2)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorSecondary)

	selectedRowStyle = lipgloss.NewStyle().
				Background(colorSelected).
				Foreground(colorText)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true)

	footerStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Background(colorSurface).
			Padding(0, 1)

	confirmRestoreStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorText).
				Background(colorPrimary).
				Padding(1, 2).
				MarginLeft(2)

	snapshotFormStyle = lipgloss.NewStyle().
				Foreground(colorText).
				Background(colorSurface).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorPrimary).
				Padding(1, 2).
				Width(56)

	snapshotInputStyle = lipgloss.NewStyle().
				Foreground(colorText).
				Background(colorSelected).
				Padding(0, 1).
				Width(48)
)

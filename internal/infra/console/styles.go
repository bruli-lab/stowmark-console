package console

import "charm.land/lipgloss/v2"

const menuWidth = 18

var (
	colorPrimary   = lipgloss.Color("#7C5CFC")
	colorSecondary = lipgloss.Color("#5B9BD5")
	colorMuted     = lipgloss.Color("#6C7086")
	colorText      = lipgloss.Color("#CDD6F4")
	colorSurface   = lipgloss.Color("#181825")
	colorSelected  = lipgloss.Color("#313244")
	colorError     = lipgloss.Color("#F38BA8")
	colorSuccess   = lipgloss.Color("#A6E3A1")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorText).
			Background(colorPrimary).
			Padding(0, 1)

	repositoryStyle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	surfaceStyle = lipgloss.NewStyle().Background(colorSurface)

	splashVersionStyle = lipgloss.NewStyle().
				Foreground(colorMuted).
				Background(colorSurface)

	panelStyle = lipgloss.NewStyle().
			Background(colorSurface).
			Foreground(colorText).
			Padding(1, 2)

	panelColumnStyle = lipgloss.NewStyle().
				Background(colorSurface).
				Foreground(colorText)

	selectedMenuStyle = lipgloss.NewStyle().
				Background(colorSelected).
				Foreground(colorText).
				Bold(true).
				Padding(0, 1).
				Width(menuWidth)

	menuStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Background(colorSurface).
			Padding(0, 1).
			Width(menuWidth)

	contentStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Padding(1, 2)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorSecondary)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true)

	footerStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Background(colorSurface).
			Padding(0, 1)

	kindStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Background(colorSelected).
			Padding(0, 1)

	errorBlockStyle = lipgloss.NewStyle().
			Foreground(colorError).
			Background(colorSurface).
			Bold(true).
			Padding(0, 1)

	tableHeaderStyle = lipgloss.NewStyle().
				Foreground(colorMuted).
				Bold(true).
				Padding(0, 1).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(colorSelected)

	tableCellStyle = lipgloss.NewStyle().
			Padding(0, 1)

	tableSelectedStyle = lipgloss.NewStyle().
				Background(colorSelected).
				Foreground(colorText).
				Bold(true)

	mutedStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	highlightStyle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	dialogStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Background(colorSurface).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(1, 2)

	inputStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Background(colorSelected).
			Padding(0, 1).
			Width(68)

	successStyle = lipgloss.NewStyle().
			Foreground(colorSuccess)
)

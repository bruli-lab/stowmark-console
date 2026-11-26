package console

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type splashFinishedMsg struct{}

type splash struct {
	version  string
	duration time.Duration
}

func WithSplash(version string, duration time.Duration) Option {
	return func(m *Model) {
		m.splash = &splash{version: version, duration: duration}
	}
}

func (s *splash) timer() tea.Cmd {
	return tea.Tick(s.duration, func(time.Time) tea.Msg { return splashFinishedMsg{} })
}

func (m *Model) skipSplash(msg tea.KeyPressMsg) tea.Cmd {
	if msg.String() == "ctrl+c" {
		return tea.Quit
	}

	m.splash = nil

	return nil
}

func (s *splash) view(width, height int) string {
	lines := []string{splashVersionStyle.Render(s.version)}

	for _, logo := range []string{splashLogo(), stowmarkLogo()} {
		if lipgloss.Height(logo)+2 <= height && lipgloss.Width(logo) <= width {
			lines = append(append(strings.Split(logo, "\n"), ""), lines...)
			break
		}
	}

	top := max(0, (height-len(lines))/2)
	screen := make([]string, 0, height)
	for row := range height {
		line := ""
		if index := row - top; index >= 0 && index < len(lines) {
			line = lines[index]
		}
		screen = append(screen, centerLine(line, width))
	}

	return strings.Join(screen, "\n")
}

func centerLine(line string, width int) string {
	line = truncate(line, width)
	free := width - ansi.StringWidth(line)
	left := free / 2

	return surfaceStyle.Render(strings.Repeat(" ", left)) + line + surfaceStyle.Render(strings.Repeat(" ", free-left))
}

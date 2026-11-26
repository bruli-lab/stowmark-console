package console

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

type previewScreen struct {
	path      string
	title     string
	truncated bool
	viewport  viewport.Model
	width     int
}

func newPreviewScreen(location, name string, content []byte, truncated bool) *previewScreen {
	text := strings.TrimSuffix(strings.ToValidUTF8(string(content), ""), "\n")
	lines := strings.Split(strings.ReplaceAll(text, "\t", "    "), "\n")

	model := viewport.New()
	model.SetContentLines(lines)
	model.LeftGutterFunc = lineNumberGutter(len(strconv.Itoa(len(lines))))

	return &previewScreen{path: location, title: name, truncated: truncated, viewport: model}
}

func lineNumberGutter(numberWidth int) viewport.GutterFunc {
	return func(info viewport.GutterContext) string {
		if info.Soft || info.Index >= info.TotalLines {
			return mutedStyle.Render(strings.Repeat(" ", numberWidth) + " │ ")
		}

		return mutedStyle.Render(fmt.Sprintf("%*d │ ", numberWidth, info.Index+1))
	}
}

func (s *previewScreen) name() string {
	return "Preview"
}

func (s *previewScreen) location() string {
	return s.path
}

func (s *previewScreen) help() string {
	return "↑↓ Scroll   ←→ Pan   [pgup/pgdown] Page   [g/G] Top/Bottom   [esc] Close"
}

func (s *previewScreen) resize(width, height int) {
	s.width = width
	s.viewport.SetWidth(width)
	s.viewport.SetHeight(max(1, height-screenTitleHeight))
}

func (s *previewScreen) update(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "backspace", "q":
		return emit(screenCloseRequested{})
	case "home", "g":
		s.viewport.GotoTop()
		return nil
	case "end", "G":
		s.viewport.GotoBottom()
		return nil
	}

	var cmd tea.Cmd
	s.viewport, cmd = s.viewport.Update(msg)

	return cmd
}

func (s *previewScreen) view() string {
	title := headerStyle.Render(s.title)
	if s.truncated {
		title += "  " + mutedStyle.Render(fmt.Sprintf("(showing the first %s)", formatSize(maxPreviewSize)))
	}

	return truncate(title, s.width) + "\n\n" + s.viewport.View()
}

package console

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

type previewScreen struct {
	request   filePreviewRequested
	content   previewContent
	truncated bool
	viewport  viewport.Model
	width     int
	height    int
}

func newPreviewScreen(request filePreviewRequested, content *previewContent, truncated bool) *previewScreen {
	model := viewport.New()

	switch content.kind {
	case textPreview:
		lines := textLines(content.data)
		model.SetContentLines(lines)
		model.LeftGutterFunc = lineNumberGutter(len(strconv.Itoa(len(lines))))
	case binaryPreview:
		model.SetContentLines(hexLines(content.data))
	case imagePreview:
	}

	return &previewScreen{request: request, content: *content, truncated: truncated, viewport: model}
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
	return s.request.location
}

func (s *previewScreen) help() string {
	if s.content.kind == imagePreview {
		return "[o] Open with the system application   [esc] Close"
	}

	return "↑↓ Scroll   ←→ Pan   [pgup/pgdown] Page   [g/G] Top/Bottom   [o] Open   [esc] Close"
}

func (s *previewScreen) resize(width, height int) {
	if width == s.width && height == s.height {
		return
	}

	s.width, s.height = width, height
	viewportHeight := max(1, height-screenTitleHeight)
	s.viewport.SetWidth(width)
	s.viewport.SetHeight(viewportHeight)

	if s.content.kind == imagePreview {
		s.viewport.SetContentLines(imageLines(s.content.image, width, viewportHeight))
	}
}

func (s *previewScreen) update(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "backspace", "q":
		return emit(screenCloseRequested{})
	case "o":
		return emit(fileOpenRequested{request: s.request})
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
	title := headerStyle.Render(s.request.name)
	if description := s.description(); description != "" {
		title += "  " + mutedStyle.Render(description)
	}

	return truncate(title, s.width) + "\n\n" + s.viewport.View()
}

func (s *previewScreen) description() string {
	var parts []string

	switch s.content.kind {
	case imagePreview:
		parts = append(parts, fmt.Sprintf("%s image, %d×%d", s.content.format, s.content.width, s.content.height))
	case binaryPreview:
		parts = append(parts, "binary file, hexadecimal view")
	case textPreview:
	}

	if s.truncated {
		parts = append(parts, "showing the first "+formatSize(previewLimit(s.request.name)))
	}

	if len(parts) == 0 {
		return ""
	}

	return "(" + strings.Join(parts, ", ") + ")"
}

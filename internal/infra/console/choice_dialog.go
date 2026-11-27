package console

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

type choice struct {
	key         string
	label       string
	description string
	cmd         tea.Cmd
}

type choiceDialog struct {
	title      string
	body       string
	note       string
	choices    []choice
	cancelHint string
	onCancel   tea.Cmd
}

func (d *choiceDialog) update(msg tea.Msg) (bool, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return false, nil
	}

	for _, option := range d.choices {
		if key.String() == option.key {
			return true, option.cmd
		}
	}

	switch key.String() {
	case "esc", "q", "n", "ctrl+c":
		return true, d.onCancel
	}

	return false, nil
}

func (d *choiceDialog) view() string {
	var builder strings.Builder

	builder.WriteString(headerStyle.Render(d.title))
	builder.WriteString("\n\n")
	if d.body != "" {
		builder.WriteString(d.body + "\n\n")
	}

	for _, option := range d.choices {
		builder.WriteString(highlightStyle.Render(fmt.Sprintf("[%s] %s", option.key, option.label)))
		builder.WriteString("\n    " + mutedStyle.Render(option.description) + "\n")
	}

	if d.note != "" {
		builder.WriteString("\n" + errorStyle.Render(d.note) + "\n")
	}

	cancelHint := d.cancelHint
	if cancelHint == "" {
		cancelHint = "Cancel"
	}
	builder.WriteString("\n[Esc] " + cancelHint)

	return builder.String()
}

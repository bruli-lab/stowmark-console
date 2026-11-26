package console

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type promptResult int

const (
	promptEditing promptResult = iota
	promptSubmitted
	promptCancelled
)

type textPrompt struct {
	title string
	label string
	value string
	err   string
}

func newTextPrompt(title, label, value string) *textPrompt {
	return &textPrompt{title: title, label: label, value: value}
}

func (p *textPrompt) handleKey(msg tea.KeyMsg) promptResult {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		return promptCancelled
	case tea.KeyEnter:
		return promptSubmitted
	case tea.KeyBackspace:
		runes := []rune(p.value)
		if len(runes) > 0 {
			p.value = string(runes[:len(runes)-1])
		}
	case tea.KeySpace:
		p.value += " "
	case tea.KeyRunes:
		p.value += string(msg.Runes)
	default:
		return promptEditing
	}

	p.err = ""

	return promptEditing
}

func (p *textPrompt) fail(reason string) {
	p.err = reason
}

func (p *textPrompt) trimmedValue() string {
	return strings.TrimSpace(p.value)
}

func (p *textPrompt) render(hint string) string {
	var builder strings.Builder

	builder.WriteString(headerStyle.Render(p.title))
	builder.WriteString("\n\n")
	builder.WriteString(p.label)
	builder.WriteString("\n")
	builder.WriteString(inputStyle.Render(p.value + "█"))

	if p.err != "" {
		builder.WriteString("\n\n")
		builder.WriteString(errorStyle.Render(p.err))
	}

	builder.WriteString("\n\n")
	builder.WriteString(hint)

	return builder.String()
}

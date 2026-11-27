package console

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

const commandInputWidth = 40

type commandLine struct {
	input textinput.Model
}

func newCommandLine() *commandLine {
	return &commandLine{input: newInput(":", "", commandInputWidth)}
}

func (c *commandLine) update(msg tea.Msg) (bool, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "esc", "ctrl+c":
			return true, nil
		case "enter":
			return true, runCommand(strings.TrimSpace(c.input.Value()))
		case "backspace":
			if c.input.Value() == "" {
				return true, nil
			}
		}
	}

	var cmd tea.Cmd
	c.input, cmd = c.input.Update(msg)

	return false, cmd
}

func (c *commandLine) view() string {
	return c.input.View()
}

func runCommand(command string) tea.Cmd {
	switch command {
	case "":
		return nil
	case "q", "q!", "quit":
		return tea.Quit
	case "s", "snap", "snapshots":
		return emit(snapshotsScreenRequested{})
	case "repo", "repos", "repositories":
		return emit(repositoriesScreenRequested{})
	case "keygen":
		return emit(keygenRequested{})
	case "rewrap":
		return emit(rewrapRequested{})
	case "rekey":
		return emit(rekeyRequested{})
	}

	return emit(statusMessage{text: "Unknown command: " + command})
}

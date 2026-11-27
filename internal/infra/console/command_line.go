package console

import (
	"slices"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

const commandInputWidth = 12

type consoleCommand struct {
	names     []string
	summary   string
	listed    bool
	encrypted bool
	run       func() tea.Cmd
}

func (c consoleCommand) name() string {
	return c.names[0]
}

var consoleCommands = []consoleCommand{
	{
		names:   []string{"repo", "repos", "repositories"},
		listed:  true,
		summary: "Repositories",
		run:     func() tea.Cmd { return emit(repositoriesScreenRequested{}) },
	},
	{
		names:   []string{"snap", "snapshots", "s"},
		summary: "Snapshots",
		run:     func() tea.Cmd { return emit(snapshotsScreenRequested{}) },
	},
	{
		names:   []string{"keygen"},
		listed:  true,
		summary: "New key pair",
		run:     func() tea.Cmd { return emit(keygenRequested{}) },
	},
	{
		names:     []string{"rewrap"},
		listed:    true,
		summary:   "Rewrap key",
		encrypted: true,
		run:       func() tea.Cmd { return emit(rewrapRequested{}) },
	},
	{
		names:     []string{"rekey"},
		listed:    true,
		summary:   "Rotate keys",
		encrypted: true,
		run:       func() tea.Cmd { return emit(rekeyRequested{}) },
	},
	{
		names:   []string{"unlock"},
		listed:  true,
		summary: "Remove stale locks",
		run:     func() tea.Cmd { return emit(unlockRequested{}) },
	},
	{
		names:   []string{"unlock!"},
		summary: "Remove every lock",
		run:     func() tea.Cmd { return emit(unlockRequested{all: true}) },
	},
	{
		names:   []string{"q", "q!", "quit"},
		summary: "Exit",
		run:     func() tea.Cmd { return tea.Quit },
	},
}

func listedCommands(encrypted bool) []consoleCommand {
	return slices.DeleteFunc(slices.Clone(consoleCommands), func(command consoleCommand) bool {
		return !command.listed || command.encrypted && !encrypted
	})
}

func commandNames() []string {
	names := make([]string, len(consoleCommands))
	for i, command := range consoleCommands {
		names[i] = command.name()
	}

	return names
}

type commandLine struct {
	input textinput.Model
}

func newCommandLine() *commandLine {
	input := newInput(":", "", commandInputWidth)
	input.ShowSuggestions = true
	input.SetSuggestions(commandNames())

	return &commandLine{input: input}
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
	matches := c.matches()
	if len(matches) == 0 {
		return c.input.View() + mutedStyle.Render("   no matching command")
	}

	hints := make([]string, len(matches))
	for i, command := range matches {
		hints[i] = ":" + command.name()
	}

	return c.input.View() + mutedStyle.Render("   "+strings.Join(hints, "  ")+"   [tab] Complete")
}

func (c *commandLine) matches() []consoleCommand {
	typed := strings.TrimSpace(c.input.Value())

	return slices.DeleteFunc(slices.Clone(consoleCommands), func(command consoleCommand) bool {
		return !slices.ContainsFunc(command.names, func(name string) bool {
			return strings.HasPrefix(name, typed)
		})
	})
}

func runCommand(command string) tea.Cmd {
	if command == "" {
		return nil
	}

	for _, candidate := range consoleCommands {
		if slices.Contains(candidate.names, command) {
			return candidate.run()
		}
	}

	return emit(statusMessage{text: "Unknown command: " + command})
}

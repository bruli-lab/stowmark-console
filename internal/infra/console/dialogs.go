package console

import (
	"errors"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

const promptInputWidth = 50

var errOriginRequired = errors.New("origin is required")

type dialog interface {
	update(msg tea.Msg) (closed bool, cmd tea.Cmd)
	view() string
}

type confirmDialog struct {
	title     string
	body      string
	warning   string
	hint      string
	onConfirm tea.Cmd
	onCancel  tea.Cmd
}

func (d *confirmDialog) update(msg tea.Msg) (bool, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return false, nil
	}

	switch key.String() {
	case "y", "Y":
		return true, d.onConfirm
	case "n", "N", "esc", "q", "ctrl+c":
		return true, d.onCancel
	}

	return false, nil
}

func (d *confirmDialog) view() string {
	var builder strings.Builder

	builder.WriteString(headerStyle.Render(d.title))
	builder.WriteString("\n\n")
	builder.WriteString(d.body)

	if d.warning != "" {
		builder.WriteString("\n\n")
		builder.WriteString(errorStyle.Render(d.warning))
	}

	builder.WriteString("\n\n")
	builder.WriteString(d.hint)

	return builder.String()
}

type promptDialog struct {
	title    string
	label    string
	hint     string
	input    textinput.Model
	err      string
	submit   func(value string) (tea.Msg, error)
	onCancel tea.Cmd
}

func newPromptDialog(title, label, value, hint string, submit func(string) (tea.Msg, error), onCancel tea.Cmd) *promptDialog {
	return &promptDialog{
		title:    title,
		label:    label,
		hint:     hint,
		input:    newInput("", value, promptInputWidth),
		submit:   submit,
		onCancel: onCancel,
	}
}

func newInput(prompt, value string, width int) textinput.Model {
	input := textinput.New()
	input.Prompt = prompt
	input.SetWidth(width)
	input.SetValue(value)

	styles := textinput.DefaultDarkStyles()
	styles.Cursor.Blink = false
	input.SetStyles(styles)
	input.Focus()

	return input
}

func (d *promptDialog) update(msg tea.Msg) (bool, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "esc", "ctrl+c":
			return true, d.onCancel
		case "enter":
			return d.confirm()
		}
	}

	d.err = ""

	var cmd tea.Cmd
	d.input, cmd = d.input.Update(msg)

	return false, cmd
}

func (d *promptDialog) confirm() (bool, tea.Cmd) {
	result, err := d.submit(strings.TrimSpace(d.input.Value()))
	if err != nil {
		d.err = err.Error()
		return false, nil
	}

	return true, emit(result)
}

func (d *promptDialog) view() string {
	var builder strings.Builder

	builder.WriteString(headerStyle.Render(d.title))
	builder.WriteString("\n\n")
	builder.WriteString(d.label)
	builder.WriteString("\n")
	builder.WriteString(inputStyle.Render(d.input.View()))

	if d.err != "" {
		builder.WriteString("\n\n")
		builder.WriteString(errorStyle.Render(d.err))
	}

	builder.WriteString("\n\n")
	builder.WriteString(d.hint)

	return builder.String()
}

func (m *Model) openInitConfirmation() {
	var body strings.Builder

	body.WriteString("No Stowmark repository found at:\n")
	body.WriteString(highlightStyle.Render(m.settings.RepositoryPath))
	body.WriteString("\n\nA new repository will be created with:")
	body.WriteString("\n  Encryption:  " + encryptionSummary(m.settings.Init))
	body.WriteString("\n  Compression: " + compressionSummary(m.settings.Init))

	m.dialog = &confirmDialog{
		title:     "Initialize repository",
		body:      body.String(),
		warning:   initRepositoryWarning(m.settings.RepositoryPath),
		hint:      "[y] Initialize  [n/Esc] Quit",
		onConfirm: emit(repositoryInitConfirmed{}),
		onCancel:  tea.Quit,
	}
}

func (m *Model) openCreateDialog() {
	m.dialog = newPromptDialog(
		"Create snapshot",
		"Origin",
		"",
		"[Enter] Create  [Esc] Cancel",
		func(value string) (tea.Msg, error) {
			if value == "" {
				return nil, errOriginRequired
			}
			source, err := resolveLocalPath(value)
			if err != nil {
				return nil, err
			}
			return snapshotSourceChosen{source: source}, nil
		},
		emit(statusMessage{text: "Snapshot creation cancelled"}),
	)
}

func (m *Model) openRestoreDialog(snapshotID string) {
	m.dialog = newPromptDialog(
		"Restore snapshot "+snapshotID,
		"Destination",
		defaultRestoreDestination(snapshotID),
		"[Enter] Continue  [Esc] Cancel",
		func(value string) (tea.Msg, error) {
			destination, err := resolveLocalPath(value)
			if err != nil {
				return nil, err
			}
			return restoreDestinationChosen{
				request: restoreRequest{snapshotID: snapshotID, destination: destination},
			}, nil
		},
		emit(statusMessage{text: "Restore cancelled"}),
	)
}

func (m *Model) openRestoreConfirmation(request restoreRequest) {
	m.dialog = &confirmDialog{
		title:     "Restore snapshot " + request.snapshotID + "?",
		body:      "Destination:\n" + highlightStyle.Render(request.destination),
		warning:   restoreDestinationWarning(request.destination),
		hint:      "[y] Confirm  [n/Esc] Cancel",
		onConfirm: emit(restoreConfirmed{request: request}),
		onCancel:  emit(statusMessage{text: "Restore cancelled"}),
	}
}

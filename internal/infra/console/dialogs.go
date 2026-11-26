package console

import (
	"errors"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

const (
	promptInputWidth   = 62
	visibleSuggestions = 5
)

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
	suggest  func(value string) []string
	check    func(value string) (string, error)
	status   string
	invalid  bool
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

func (d *promptDialog) withCompletion(suggest func(string) []string, check func(string) (string, error)) *promptDialog {
	d.suggest = suggest
	d.check = check
	d.input.ShowSuggestions = true
	d.refresh()

	return d
}

func (d *promptDialog) refresh() {
	value := strings.TrimSpace(d.input.Value())

	if d.suggest != nil {
		d.input.SetSuggestions(d.suggest(value))
	}

	if d.check == nil {
		return
	}

	d.status, d.invalid = "", false
	if value == "" {
		return
	}

	status, err := d.check(value)
	if err != nil {
		d.status, d.invalid = err.Error(), true
		return
	}
	d.status = status
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
	previous := d.input.Value()

	var cmd tea.Cmd
	d.input, cmd = d.input.Update(msg)

	if d.input.Value() != previous {
		d.refresh()
	}

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
	d.writeSuggestions(&builder)
	d.writeStatus(&builder)

	if d.err != "" {
		builder.WriteString("\n\n")
		builder.WriteString(errorStyle.Render(d.err))
	}

	builder.WriteString("\n\n")
	builder.WriteString(d.hint)

	return builder.String()
}

func (d *promptDialog) writeSuggestions(builder *strings.Builder) {
	matches := d.input.MatchedSuggestions()
	if len(matches) == 0 || (len(matches) == 1 && matches[0] == d.input.Value()) {
		return
	}

	current := d.input.CurrentSuggestionIndex()
	start := min(max(0, current-visibleSuggestions/2), max(0, len(matches)-visibleSuggestions))
	end := min(len(matches), start+visibleSuggestions)
	width := promptInputWidth

	for index := start; index < end; index++ {
		builder.WriteString("\n")
		if index == current {
			builder.WriteString(highlightStyle.Render("› " + truncateLeft(matches[index], width)))
			continue
		}
		builder.WriteString(mutedStyle.Render("  " + truncateLeft(matches[index], width)))
	}

	if len(matches) > visibleSuggestions {
		builder.WriteString("\n")
		builder.WriteString(mutedStyle.Render(fmt.Sprintf("  %d of %d", current+1, len(matches))))
	}
}

func (d *promptDialog) writeStatus(builder *strings.Builder) {
	if d.status == "" || (d.invalid && len(d.input.MatchedSuggestions()) > 0) {
		return
	}

	builder.WriteString("\n")
	if d.invalid {
		builder.WriteString(errorStyle.Render("✗ " + d.status))
		return
	}
	builder.WriteString(successStyle.Render("✓ " + d.status))
}

func (m *Model) openInitConfirmation() {
	declineHint := "Quit"
	if len(m.repositories) > 1 {
		declineHint = "Choose another repository"
	}

	var body strings.Builder

	body.WriteString("No Stowmark repository found at:\n")
	body.WriteString(highlightStyle.Render(m.current.Location()))
	body.WriteString("\n\nA new repository will be created with:")
	body.WriteString("\n  Encryption:  " + encryptionSummary(m.current.Init()))
	body.WriteString("\n  Compression: " + compressionSummary(m.current.Init()))

	m.dialog = &confirmDialog{
		title:     "Initialize repository " + m.current.Name(),
		body:      body.String(),
		warning:   initRepositoryWarning(m.current),
		hint:      "[y] Initialize  [n/Esc] " + declineHint,
		onConfirm: emit(repositoryInitConfirmed{}),
		onCancel:  emit(repositoryInitDeclined{}),
	}
}

func (m *Model) openCreateDialog() {
	origins := knownOrigins(m.snapshots.snapshots)
	origin := ""
	if selected, ok := m.snapshots.selected(); ok {
		origin = selected.Source()
	}

	m.dialog = newPromptDialog(
		"Create snapshot",
		"Origin directory",
		origin,
		"[Tab] Complete  [↑↓] Suggestions  [Enter] Create  [Esc] Cancel",
		func(value string) (tea.Msg, error) {
			if value == "" {
				return nil, errOriginRequired
			}
			if _, err := describeDirectory(value); err != nil {
				return nil, err
			}
			source, err := resolveLocalPath(value)
			if err != nil {
				return nil, err
			}
			return snapshotSourceChosen{source: source}, nil
		},
		emit(statusMessage{text: "Snapshot creation cancelled"}),
	).withCompletion(
		func(value string) []string { return pathSuggestions(value, origins) },
		describeDirectory,
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

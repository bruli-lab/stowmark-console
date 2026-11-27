package console

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/bruli-lab/stowmark-console/internal/domain/retention"
)

const (
	retentionInputWidth = 8
	retentionLabelWidth = 10
)

type retentionField struct {
	label string
	hint  string
	input textinput.Model
}

type retentionForm struct {
	title  string
	note   string
	fields []retentionField
	focus  int
	err    error
}

func newRetentionForm(repositoryName string, rules retention.Rules, configured bool) *retentionForm {
	count := func(value int) string {
		if value == 0 {
			return ""
		}
		return strconv.Itoa(value)
	}

	form := &retentionForm{
		title: "Forget snapshots of " + repositoryName,
		fields: []retentionField{
			{label: "Last", hint: "the newest snapshots", input: newInput("", count(rules.Last), retentionInputWidth)},
			{label: "Daily", hint: "the newest of each day", input: newInput("", count(rules.Daily), retentionInputWidth)},
			{label: "Weekly", hint: "the newest of each week", input: newInput("", count(rules.Weekly), retentionInputWidth)},
			{label: "Monthly", hint: "the newest of each month", input: newInput("", count(rules.Monthly), retentionInputWidth)},
			{label: "Yearly", hint: "the newest of each year", input: newInput("", count(rules.Yearly), retentionInputWidth)},
			{label: "Within", hint: "every snapshot newer than, e.g. 30d, 2w, 36h", input: newInput("", retention.FormatWithin(rules.Within), retentionInputWidth)},
		},
	}

	if configured {
		form.note = "Prefilled with the retention of the repositories file."
	}

	form.focusField(0)

	return form
}

func (f *retentionForm) focusField(index int) {
	f.focus = (index + len(f.fields)) % len(f.fields)
	for i := range f.fields {
		if i == f.focus {
			f.fields[i].input.Focus()
		} else {
			f.fields[i].input.Blur()
		}
	}
}

func (f *retentionForm) update(msg tea.Msg) (bool, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "esc", "ctrl+c":
			return true, emit(statusMessage{text: "Forget cancelled"})
		case "tab", "down":
			f.focusField(f.focus + 1)
			return false, nil
		case "shift+tab", "up":
			f.focusField(f.focus - 1)
			return false, nil
		case "enter":
			rules, err := f.rules()
			if err != nil {
				f.err = err
				return false, nil
			}
			return true, emit(forgetRulesChosen{rules: rules})
		}
	}

	var cmd tea.Cmd
	f.fields[f.focus].input, cmd = f.fields[f.focus].input.Update(msg)
	f.err = nil

	return false, cmd
}

func (f *retentionForm) rules() (retention.Rules, error) {
	counts := make([]int, len(f.fields)-1)
	for i := range counts {
		value := strings.TrimSpace(f.fields[i].input.Value())
		if value == "" {
			continue
		}
		count, err := strconv.Atoi(value)
		if err != nil || count < 0 {
			return retention.Rules{}, fmt.Errorf("%s must be a positive whole number", f.fields[i].label)
		}
		counts[i] = count
	}

	within, err := retention.ParseWithin(f.fields[len(f.fields)-1].input.Value())
	if err != nil {
		return retention.Rules{}, err
	}

	rules := retention.Rules{
		Last: counts[0], Daily: counts[1], Weekly: counts[2], Monthly: counts[3], Yearly: counts[4], Within: within,
	}

	return rules, rules.Validate()
}

func (f *retentionForm) view() string {
	var builder strings.Builder

	builder.WriteString(headerStyle.Render(f.title))
	builder.WriteString("\n\n")
	builder.WriteString("Keep the snapshots matched by any rule; leave a rule empty to skip it.\n")
	builder.WriteString("Stowmark applies the rules to the snapshots of each source on its own.\n")
	if f.note != "" {
		builder.WriteString(mutedStyle.Render(f.note) + "\n")
	}
	builder.WriteString("\n")

	for i := range f.fields {
		field := &f.fields[i]
		label := fmt.Sprintf("%-*s", retentionLabelWidth, field.label)
		if i == f.focus {
			label = highlightStyle.Render(label)
		}
		builder.WriteString(label + inputStyle.Width(retentionInputWidth+3).Render(field.input.View()) + "  " +
			mutedStyle.Render(field.hint) + "\n")
	}

	if f.err != nil {
		builder.WriteString("\n" + errorStyle.Render(f.err.Error()) + "\n")
	}

	builder.WriteString("\n[Tab/↓] Next  [↑] Previous  [Enter] Preview  [Esc] Cancel")

	return builder.String()
}

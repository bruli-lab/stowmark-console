package console

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

const (
	excludeInputWidth   = 52
	maxListedSkipped    = 12
	createOptionsFields = 2
)

type snapshotOriginChosen struct {
	source string
}

type createOptionsDialog struct {
	source          string
	note            string
	exclude         textinput.Model
	readIgnoreFiles bool
	hasIgnoreFile   bool
	focus           int
}

func newCreateOptionsDialog(source string, defaults snapshot.Exclusions) *createOptionsDialog {
	dialog := &createOptionsDialog{
		source:          source,
		exclude:         newInput("", strings.Join(defaults.Patterns(), ", "), excludeInputWidth),
		readIgnoreFiles: !defaults.SkipsIgnoreFiles(),
		hasIgnoreFile:   fileExists(filepath.Join(source, snapshot.IgnoreFileName)),
	}
	if !defaults.Empty() {
		dialog.note = "Prefilled with the exclusions of the repositories file."
	}
	dialog.exclude.Focus()

	return dialog
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func (d *createOptionsDialog) exclusions() snapshot.Exclusions {
	return snapshot.NewExclusions(snapshot.ParseExcludePatterns(d.exclude.Value()), !d.readIgnoreFiles)
}

func (d *createOptionsDialog) focusField(index int) {
	d.focus = (index + createOptionsFields) % createOptionsFields
	if d.focus == 0 {
		d.exclude.Focus()
	} else {
		d.exclude.Blur()
	}
}

func (d *createOptionsDialog) update(msg tea.Msg) (bool, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "esc", "ctrl+c":
			return true, emit(statusMessage{text: "Snapshot creation cancelled"})
		case "enter":
			return true, emit(snapshotSourceChosen{source: d.source, exclusions: d.exclusions()})
		case "tab", "down":
			d.focusField(d.focus + 1)
			return false, nil
		case "shift+tab", "up":
			d.focusField(d.focus - 1)
			return false, nil
		case "space":
			if d.focus == 1 {
				d.readIgnoreFiles = !d.readIgnoreFiles
				return false, nil
			}
		}
	}

	if d.focus != 0 {
		return false, nil
	}

	var cmd tea.Cmd
	d.exclude, cmd = d.exclude.Update(msg)

	return false, cmd
}

func (d *createOptionsDialog) view() string {
	var builder strings.Builder

	builder.WriteString(headerStyle.Render("Create snapshot of "+d.source) + "\n\n")
	if d.note != "" {
		builder.WriteString(mutedStyle.Render(d.note) + "\n\n")
	}

	label := "Exclude"
	if d.focus == 0 {
		label = highlightStyle.Render(label)
	}
	builder.WriteString(label + "\n" + inputStyle.Width(excludeInputWidth+3).Render(d.exclude.View()) + "\n")
	builder.WriteString(mutedStyle.Render("Patterns separated by commas, as in .gitignore: *.tmp, cache/, /build, !keep.log") + "\n\n")

	box := "[ ]"
	if d.readIgnoreFiles {
		box = "[x]"
	}
	toggle := box + " Read " + snapshot.IgnoreFileName + " files"
	if d.focus == 1 {
		toggle = highlightStyle.Render(toggle)
	}
	builder.WriteString(toggle + "\n")
	if d.hasIgnoreFile {
		builder.WriteString(mutedStyle.Render("    the origin has a "+snapshot.IgnoreFileName+" file") + "\n")
	}

	builder.WriteString("\n[Tab/↓] Next  [Space] Toggle  [Enter] Create  [Esc] Cancel")

	return builder.String()
}

func skippedSummary(skipped []snapshot.SkippedFile) string {
	kinds := snapshot.SkippedKinds(skipped)
	parts := make([]string, 0, len(kinds))
	for _, kind := range slices.Sorted(maps.Keys(kinds)) {
		parts = append(parts, plural(kinds[kind], kind))
	}

	return fmt.Sprintf("%d skipped (%s)", len(skipped), strings.Join(parts, ", "))
}

func skippedDialog(created *snapshot.Created) *infoDialog {
	var body strings.Builder

	body.WriteString("Stowmark stores regular files only (and, from format version 3, directories and\n")
	body.WriteString("symbolic links), so these entries of the origin are not in the snapshot:\n\n")

	for i, file := range created.Skipped {
		if i == maxListedSkipped {
			body.WriteString(mutedStyle.Render(fmt.Sprintf("  … and %d more", len(created.Skipped)-maxListedSkipped)) + "\n")
			break
		}
		body.WriteString(mutedStyle.Render(fmt.Sprintf("  %-15s", file.Kind)) + file.Path + "\n")
	}

	return &infoDialog{
		title: fmt.Sprintf("Snapshot %s: %s", created.Snapshot.ID(), skippedSummary(created.Skipped)),
		body:  strings.TrimRight(body.String(), "\n"),
	}
}

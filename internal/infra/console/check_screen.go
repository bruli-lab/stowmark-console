package console

import (
	"slices"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/bruli-lab/stowmark-console/internal/domain/integrity"
)

type checkScreen struct {
	report   integrity.Report
	viewport viewport.Model
	width    int
}

func newCheckScreen(report *integrity.Report) *checkScreen {
	model := viewport.New()
	model.SetContentLines(slices.Concat(checkSummary(report), checkProblems(report)))

	return &checkScreen{report: *report, viewport: model}
}

func (s *checkScreen) name() string {
	return "Check"
}

func (s *checkScreen) location() string {
	return "check " + s.report.Scope.String()
}

func (s *checkScreen) help() string {
	help := "↑↓ Scroll"
	if s.report.Prunable() {
		help += "   [p] Prune"
	}
	if s.report.StaleLocks > 0 {
		help += "   [u] Unlock"
	}

	return help + "   [r] Check again   [esc] Back"
}

func (s *checkScreen) resize(width, height int) {
	s.width = width
	s.viewport.SetWidth(width)
	s.viewport.SetHeight(max(1, height-screenTitleHeight))
}

func (s *checkScreen) update(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "backspace", "q":
		return emit(screenCloseRequested{})
	case "r":
		return emit(checkScopeChosen{scope: s.report.Scope})
	case "p":
		if s.report.Prunable() {
			return emit(pruneRequested{})
		}
		return nil
	case "u":
		if s.report.StaleLocks > 0 {
			return emit(unlockRequested{})
		}
		return nil
	}

	var cmd tea.Cmd
	s.viewport, cmd = s.viewport.Update(msg)

	return cmd
}

func (s *checkScreen) view() string {
	status := successStyle.Render("✓ Repository OK")
	switch {
	case s.report.HasErrors():
		status = errorStyle.Render("✗ Repository has errors")
	case s.report.HasWarnings():
		status = successStyle.Render("✓ Repository OK") + highlightStyle.Render(", with warnings")
	}

	title := status + "  " + mutedStyle.Render("checked the "+s.report.Scope.String())

	return truncate(title, s.width) + "\n\n" + s.viewport.View()
}

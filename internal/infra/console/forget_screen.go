package console

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"github.com/bruli-lab/stowmark-console/internal/domain/retention"
)

var forgetColumns = []columnSpec{
	{title: "ACTION", width: 8},
	{title: "CREATED", width: 19},
	{title: "ID", width: 26},
	{title: "KEPT BY", width: 22},
	{title: "SOURCE"},
}

type forgetScreen struct {
	rules retention.Rules
	plan  retention.Plan
	table table.Model
}

func newForgetScreen(rules retention.Rules, plan retention.Plan) *forgetScreen {
	sorted := slices.Clone(plan)
	slices.SortStableFunc(sorted, func(a, b retention.Decision) int {
		return cmp.Or(cmp.Compare(a.Source, b.Source), b.CreatedAt.Compare(a.CreatedAt))
	})

	screen := &forgetScreen{rules: rules, plan: sorted, table: newTable()}
	rows := make([]table.Row, len(sorted))
	for i := range sorted {
		decision := &sorted[i]
		action := "✗ forget"
		if decision.Keep() {
			action = "✓ keep"
		}
		rows[i] = tableRow(forgetColumns,
			action,
			decision.CreatedAt.Local().Format(dateTimeFormat),
			decision.SnapshotID,
			strings.Join(decision.Reasons, ", "),
			decision.Source,
		)
	}
	selectRow(&screen.table, rows, 0)

	return screen
}

func (s *forgetScreen) name() string {
	return "Forget"
}

func (s *forgetScreen) location() string {
	return "forget " + s.rules.String()
}

func (s *forgetScreen) help() string {
	return "↑↓ Navigate   [f] Forget   [p] Forget and prune   [esc] Back"
}

func (s *forgetScreen) resize(width, height int) {
	resizeTable(&s.table, forgetColumns, width, height)
}

func (s *forgetScreen) update(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "f", "enter":
		return emit(forgetRequested{request: forgetRequest{rules: s.rules, plan: s.plan}})
	case "p":
		return emit(forgetRequested{request: forgetRequest{rules: s.rules, plan: s.plan, prune: true}})
	case "esc", "backspace", "q":
		return emit(screenCloseRequested{})
	}

	var cmd tea.Cmd
	s.table, cmd = s.table.Update(msg)

	return cmd
}

func (s *forgetScreen) view() string {
	title := headerStyle.Render(fmt.Sprintf("Forget plan (%s): keep %d, forget %d",
		s.rules.String(), s.plan.Kept(), s.plan.Forgotten()))

	if len(s.plan) == 0 {
		return title + "\n\nThe repository has no snapshots."
	}

	return title + "\n\n" + s.table.View()
}

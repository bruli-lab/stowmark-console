package console

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/bruli-lab/stowmark-console/internal/app"
	"github.com/bruli-lab/stowmark-console/internal/domain/integrity"
)

const defaultDataSubset = "10%"

type checkRequested struct{}

type checkSubsetRequested struct{}

type checkScopeChosen struct {
	scope integrity.Scope
}

type repositoryCheckedMsg struct {
	report integrity.Report
	err    error
}

func (m *Model) handleCheckIntent(msg tea.Msg) (tea.Cmd, bool) {
	switch msg.(type) {
	case checkRequested:
		if m.requireRepository() {
			m.openCheckChoice()
		}
	case checkSubsetRequested:
		m.dialog = newPromptDialog(
			"Check part of the data of "+m.current.Name(),
			"Data subset",
			defaultDataSubset,
			"A percentage of the objects chosen at random (10%), or a part (2/7 checks\n"+
				"the second of seven parts: check 1/7 to 7/7 over time to cover everything).",
			func(value string) (tea.Msg, error) {
				scope, err := integrity.DataSubset(value)
				if err != nil {
					return nil, err
				}
				return checkScopeChosen{scope: scope}, nil
			},
			emit(statusMessage{text: "Check cancelled"}),
		)
	default:
		return nil, false
	}

	return nil, true
}

func (m *Model) openCheckChoice() {
	note := ""
	if m.current.MissingPrivateKey() {
		note = "Reading the data of this encrypted repository needs its private key,\nwhich is not configured."
	}

	m.dialog = &choiceDialog{
		title: "Check repository " + m.current.Name(),
		body:  "Every check reads the snapshots, the list of objects, the locks and the key generations.",
		note:  note,
		choices: []choice{
			{
				key:         "s",
				label:       "Structure only",
				description: "fast; finds corrupt manifests and missing objects",
				cmd:         emit(checkScopeChosen{scope: integrity.StructureOnly()}),
			},
			{
				key:         "d",
				label:       "All data",
				description: "also downloads every object and verifies its hash; slow on big repositories",
				cmd:         emit(checkScopeChosen{scope: integrity.AllData()}),
			},
			{
				key:         "p",
				label:       "Part of the data",
				description: "verifies a percentage or a part of the objects",
				cmd:         emit(checkSubsetRequested{}),
			},
		},
		onCancel: emit(statusMessage{text: "Check cancelled"}),
	}
}

func (m *Model) runCheck(scope integrity.Scope) tea.Cmd {
	bus := m.buses.Queries
	return m.startOperation("Checking the "+scope.String()+" of "+m.current.Name()+"...", func(ctx context.Context) tea.Msg {
		report, err := ask[integrity.Report](ctx, bus, app.CheckRepositoryQuery{Scope: scope})
		return repositoryCheckedMsg{report: report, err: err}
	})
}

func (m *Model) handleChecked(msg *repositoryCheckedMsg) {
	m.finishOperation()
	if m.fail("check repository "+m.current.Name(), msg.err) {
		return
	}

	if _, ok := m.top().(*checkScreen); ok {
		m.pop()
	}
	m.push(newCheckScreen(&msg.report))
}

func checkSummary(report *integrity.Report) []string {
	generations := "none"
	if len(report.OldKeyGenerations) > 0 {
		generations = formatGenerations(report.OldKeyGenerations)
	}

	rows := [][2]string{
		{"Snapshots", fmt.Sprint(report.Snapshots)},
		{"Corrupt manifests", fmt.Sprint(len(report.CorruptManifests))},
		{"Stored objects", fmt.Sprint(report.StoredObjects)},
		{"Referenced objects", fmt.Sprint(report.ReferencedObjects)},
		{"Missing objects", fmt.Sprint(len(report.MissingObjects))},
		{"Unreferenced objects", fmt.Sprint(report.UnreferencedObjects)},
		{"Stale locks", fmt.Sprint(report.StaleLocks)},
		{"Old key generations", generations},
	}
	if report.Scope.ReadsData() {
		rows = append(rows,
			[2]string{"Read objects", fmt.Sprint(report.ReadObjects)},
			[2]string{"Corrupt objects", fmt.Sprint(len(report.CorruptObjects))},
		)
	}

	lines := make([]string, len(rows))
	for i, row := range rows {
		lines[i] = mutedStyle.Render(fmt.Sprintf("%-22s", row[0])) + row[1]
	}

	return lines
}

func checkProblems(report *integrity.Report) []string {
	var lines []string

	if damaged := report.DamagedSnapshots(); len(damaged) > 0 {
		lines = append(lines, "", headerStyle.Render("Damaged snapshots"))
		for _, id := range damaged {
			lines = append(lines, "  "+id)
		}
	}

	var errs []string
	for _, problem := range report.CorruptManifests {
		errs = append(errs, fmt.Sprintf("manifest of snapshot %s cannot be read: %s", problem.ID, problem.Reason))
	}
	for _, missing := range report.MissingObjects {
		errs = append(errs, fmt.Sprintf("object %s is missing, used by %s", missing.Hash, strings.Join(missing.Snapshots, ", ")))
	}
	for _, problem := range report.CorruptObjects {
		errs = append(errs, fmt.Sprintf("object %s cannot be read: %s", problem.ID, problem.Reason))
	}
	if len(errs) > 0 {
		lines = append(lines, "", headerStyle.Render("Errors"))
		for _, line := range errs {
			lines = append(lines, errorStyle.Render("  ✗ ")+line)
		}
	}

	if warnings := checkWarnings(report); len(warnings) > 0 {
		lines = append(lines, "", headerStyle.Render("Warnings"))
		for _, line := range warnings {
			lines = append(lines, highlightStyle.Render("  ! ")+line)
		}
	}

	if len(lines) == 0 {
		lines = append(lines, "", successStyle.Render("No problems found."))
	}

	return lines
}

func checkWarnings(report *integrity.Report) []string {
	var warnings []string

	if report.UnreferencedObjects > 0 {
		warnings = append(warnings, fmt.Sprintf("%s not used by any snapshot: [p] prune frees them",
			plural(report.UnreferencedObjects, "object")))
	}
	if report.StaleLocks > 0 {
		warnings = append(warnings, plural(report.StaleLocks, "stale lock")+": [u] unlock removes them")
	}
	if len(report.OldKeyGenerations) > 0 {
		warnings = append(warnings, fmt.Sprintf("objects of old key generations %s remain, probably from an interrupted rekey: [p] prune removes them",
			formatGenerations(report.OldKeyGenerations)))
	}
	if report.UnprotectedItems > 0 {
		warnings = append(warnings, fmt.Sprintf("the object lock retention of %s in use has ended: extend it with Stowmark (retention extend)",
			plural(report.UnprotectedItems, "item")))
	}

	return warnings
}

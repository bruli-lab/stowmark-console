package console

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/bruli-lab/stowmark-console/internal/app"
	"github.com/bruli-lab/stowmark-console/internal/domain/retention"
)

const retainedDateFormat = "2006-01-02"

type forgetFormRequested struct{}

type forgetRulesChosen struct {
	rules retention.Rules
}

type forgetRequest struct {
	rules retention.Rules
	plan  retention.Plan
	prune bool
}

type forgetRequested struct {
	request forgetRequest
}

type forgetConfirmed struct {
	request forgetRequest
}

type pruneRequested struct{}

type pruneConfirmed struct{}

type forgetPlannedMsg struct {
	rules retention.Rules
	plan  retention.Plan
	err   error
}

type snapshotsForgottenMsg struct {
	event    retention.SnapshotsForgotten
	err      error
	pruned   *retention.RepositoryPruned
	pruneErr error
}

type prunePlannedMsg struct {
	plan retention.PrunePlan
	err  error
}

type repositoryPrunedMsg struct {
	event retention.RepositoryPruned
	err   error
}

func (m *Model) handleRetentionIntent(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case forgetFormRequested:
		if m.requireRepository() {
			rules, configured := m.current.Retention()
			m.dialog = newRetentionForm(m.current.Name(), rules, configured)
		}
	case forgetRequested:
		m.openForgetConfirmation(&msg.request)
	default:
		return nil, false
	}

	return nil, true
}

func (m *Model) openForgetConfirmation(request *forgetRequest) {
	forgotten := request.plan.Forgotten()
	if forgotten == 0 {
		m.message = "Nothing to forget: the policy keeps every snapshot"
		return
	}

	action := "Forget"
	body := fmt.Sprintf("Policy: %s\nKeep %s and forget %s.",
		request.rules.String(), plural(request.plan.Kept(), "snapshot"), plural(forgotten, "snapshot"))
	if request.prune {
		action = "Forget and prune"
		body += "\n\nThen prune frees the space of their contents. Prune takes an exclusive\n" +
			"lock: no other Stowmark process may use the repository meanwhile."
	}

	m.dialog = &confirmDialog{
		title:     fmt.Sprintf("Forget %s of %s?", plural(forgotten, "snapshot"), m.current.Name()),
		body:      body,
		warning:   "Forgotten snapshots cannot be restored.",
		hint:      "[y] " + action + "  [n/Esc] Cancel",
		onConfirm: emit(forgetConfirmed{request: *request}),
		onCancel:  emit(statusMessage{text: "Forget cancelled"}),
	}
}

func (m *Model) planForget(rules retention.Rules) tea.Cmd {
	bus := m.buses.Queries
	return m.startOperation("Planning forget...", func(ctx context.Context) tea.Msg {
		plan, err := ask[retention.Plan](ctx, bus, app.PlanForgetQuery{Rules: rules})
		return forgetPlannedMsg{rules: rules, plan: plan, err: err}
	})
}

func (m *Model) forgetSnapshots(request *forgetRequest) tea.Cmd {
	bus := m.buses.Commands
	activity := "Forgetting snapshots..."
	if request.prune {
		activity = "Forgetting snapshots and pruning..."
	}

	rules, prune := request.rules, request.prune

	return m.startOperation(activity, func(ctx context.Context) tea.Msg {
		forgotten, err := dispatch[retention.SnapshotsForgotten](ctx, bus, app.ForgetSnapshotsCommand{Rules: rules})
		msg := snapshotsForgottenMsg{event: forgotten, err: err}
		if err != nil || !prune {
			return msg
		}

		pruned, err := dispatch[retention.RepositoryPruned](ctx, bus, app.PruneRepositoryCommand{})
		msg.pruned = &pruned
		msg.pruneErr = err

		return msg
	})
}

func (m *Model) planPrune() tea.Cmd {
	bus := m.buses.Queries
	return m.startOperation("Planning prune (it lists every object, which is slow on S3 and GCS)...",
		func(ctx context.Context) tea.Msg {
			plan, err := ask[retention.PrunePlan](ctx, bus, app.PlanPruneQuery{})
			return prunePlannedMsg{plan: plan, err: err}
		})
}

func (m *Model) pruneRepository() tea.Cmd {
	bus := m.buses.Commands
	return m.startOperation("Pruning repository...", func(ctx context.Context) tea.Msg {
		pruned, err := dispatch[retention.RepositoryPruned](ctx, bus, app.PruneRepositoryCommand{})
		return repositoryPrunedMsg{event: pruned, err: err}
	})
}

func (m *Model) handleRetentionResult(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case forgetPlannedMsg:
		m.finishOperation()
		if m.fail("plan forget", msg.err) {
			return nil
		}
		m.push(newForgetScreen(msg.rules, msg.plan))

	case snapshotsForgottenMsg:
		return m.handleForgotten(&msg)

	case prunePlannedMsg:
		m.finishOperation()
		if m.fail("plan prune", msg.err) {
			return nil
		}
		m.openPruneConfirmation(&msg.plan)

	case repositoryPrunedMsg:
		m.finishOperation()
		if m.fail("prune repository "+m.current.Name(), msg.err) {
			return nil
		}
		m.message = pruneSummary(&msg.event.Result)
	}

	return nil
}

func (m *Model) handleForgotten(msg *snapshotsForgottenMsg) tea.Cmd {
	m.finishOperation()
	if m.fail("forget snapshots", msg.err) {
		return nil
	}

	m.screens = nil
	result := msg.event.Result
	m.message = "Forgot " + plural(len(result.Forgotten), "snapshot")
	if len(result.Postponed) > 0 {
		m.message += fmt.Sprintf("; %s kept by object lock until %s",
			plural(len(result.Postponed), "snapshot"), latestRetention(result.Postponed).Format(retainedDateFormat))
	}

	if msg.pruneErr != nil {
		m.fail("prune repository "+m.current.Name()+" after forgetting", msg.pruneErr)
	} else if msg.pruned != nil {
		m.message += ". " + pruneSummary(&msg.pruned.Result)
	}

	return m.loadSnapshots("")
}

func (m *Model) openPruneConfirmation(plan *retention.PrunePlan) {
	missing := ""
	if plan.MissingObjects > 0 {
		missing = fmt.Sprintf("Missing objects: %d (the snapshots that use them are damaged).", plan.MissingObjects)
	}

	if plan.Empty() {
		m.message = fmt.Sprintf("Nothing to prune: every stored object is referenced (%d)", plan.StoredObjects)
		if missing != "" {
			m.message += ". " + missing
		}
		return
	}

	body := fmt.Sprintf("%-22s%d\n%-22s%d\n%-22s%d (%s to free)",
		"Stored objects", plan.StoredObjects,
		"Referenced objects", plan.ReferencedObjects,
		"Unreferenced objects", plan.UnreferencedObjects, formatSize(plan.UnreferencedBytes))
	if len(plan.OldKeyGenerations) > 0 {
		body += fmt.Sprintf("\n%-22s%s", "Old key generations", generations(plan.OldKeyGenerations))
	}
	body += "\n\nPrune takes an exclusive lock: no other Stowmark process may use the\nrepository meanwhile."

	m.dialog = &confirmDialog{
		title:     "Prune repository " + m.current.Name() + "?",
		body:      body,
		warning:   missing,
		hint:      "[y] Prune  [n/Esc] Cancel",
		onConfirm: emit(pruneConfirmed{}),
		onCancel:  emit(statusMessage{text: "Prune cancelled"}),
	}
}

func pruneSummary(result *retention.PruneResult) string {
	parts := []string{fmt.Sprintf("Pruned: removed %s and freed %s",
		plural(result.RemovedObjects, "object"), formatSize(result.FreedBytes))}

	if len(result.RemovedKeyGenerations) > 0 {
		parts = append(parts, "removed key generations "+generations(result.RemovedKeyGenerations))
	}
	if result.RemovedTemporaryFiles > 0 {
		parts = append(parts, "removed "+plural(result.RemovedTemporaryFiles, "temporary file"))
	}
	if result.PostponedObjects > 0 {
		parts = append(parts, fmt.Sprintf("%s (%s) kept by object lock until %s",
			plural(result.PostponedObjects, "object"), formatSize(result.PostponedBytes),
			result.RetainedUntil.Format(retainedDateFormat)))
	}
	if result.MissingObjects > 0 {
		parts = append(parts, plural(result.MissingObjects, "object")+" missing")
	}

	return strings.Join(parts, "; ")
}

func generations(values []uint64) string {
	names := make([]string, len(values))
	for i, value := range values {
		names[i] = fmt.Sprintf("%d", value)
	}

	return strings.Join(names, ", ")
}

func latestRetention(postponed []retention.PostponedSnapshot) (latest time.Time) {
	for _, item := range postponed {
		if item.RetainedUntil.After(latest) {
			latest = item.RetainedUntil
		}
	}

	return latest
}

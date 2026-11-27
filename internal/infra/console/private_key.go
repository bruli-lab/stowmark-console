package console

import (
	"context"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/bruli-lab/stowmark-console/internal/app"
	"github.com/bruli-lab/stowmark-console/internal/domain/encryption"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
)

type privateKeyChosen struct {
	path string
}

type privateKeyDeclined struct{}

type privateKeyUsedMsg struct {
	path    string
	profile repository.Profile
	status  repository.Status
	err     error
}

func (m *Model) openPrivateKeyPrompt() {
	prompt := newKeyFileDialog(
		"Private key of "+m.current.Name(),
		"The repository is encrypted. Private key file for this session",
		suggestedPrivateKey(defaultKeysFolder()),
		func(path string) tea.Msg { return privateKeyChosen{path: path} },
		emit(privateKeyDeclined{}),
	)
	if m.err != nil {
		prompt.err = m.err.Error()
	}
	m.dialog = prompt
}

func (m *Model) usePrivateKey(path string) tea.Cmd {
	commands, queries := m.buses.Commands, m.buses.Queries
	profile := m.current
	return m.startOperation("Opening repository "+profile.Name()+" with its private key...", func(ctx context.Context) tea.Msg {
		opened, err := dispatch[repository.Opened](ctx, commands, app.UsePrivateKeyCommand{Profile: profile, PrivateKey: path})
		if err != nil {
			return privateKeyUsedMsg{path: path, profile: profile, err: err}
		}
		status, err := ask[repository.Status](ctx, queries, app.RepositoryStatusQuery{})
		return privateKeyUsedMsg{path: path, profile: opened.Profile, status: status, err: err}
	})
}

func (m *Model) handlePrivateKeyUsed(msg *privateKeyUsedMsg) tea.Cmd {
	m.finishOperation()
	m.current = msg.profile
	if m.failWithKey("open repository "+m.current.Name(), msg.err, &msg.path) {
		m.openPrivateKeyPrompt()
		return nil
	}

	if msg.status != repository.StatusReady {
		return m.applyStatus(msg.status)
	}

	m.message = "Repository " + m.current.Name() + " opened with the private key " + msg.path
	m.dialog = &infoDialog{title: "Private key of " + m.current.Name(), body: privateKeySummary(m.current.Name(), msg.path)}

	return m.loadSnapshots("")
}

func privateKeySummary(name, path string) string {
	var body strings.Builder

	body.WriteString("This session uses the private key\n" + highlightStyle.Render(path) + "\n\n")
	body.WriteString("To open the repository next time without being asked, ")
	if name == environmentRepositoryName {
		body.WriteString("set:\n\n" + highlightStyle.Render("    STOWMARK_PRIVATE_KEY="+path))
	} else {
		body.WriteString("add to " + highlightStyle.Render(name) + "\nin repositories.yaml:\n\n" + highlightStyle.Render("    private_key: "+path))
	}

	return body.String()
}

func (m *Model) explainMissingKey() {
	m.declinePrivateKey()
	m.openPrivateKeyPrompt()
}

func (m *Model) declinePrivateKey() {
	m.err = explainKeyError(m.current, encryption.ErrPrivateKeyRequired)
	if len(m.repositories) > 1 {
		m.showRepositories()
	}
}

func suggestedPrivateKey(folder string) string {
	matches, err := filepath.Glob(filepath.Join(folder, "*-private.pem"))
	if err != nil || len(matches) != 1 {
		return folder + string(filepath.Separator)
	}

	return matches[0]
}

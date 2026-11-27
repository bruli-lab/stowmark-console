package console

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/bruli-lab/stowmark-console/internal/app"
	"github.com/bruli-lab/stowmark-console/internal/domain/encryption"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
)

type encryptedInitFolderChosen struct {
	folder  string
	options repository.InitOptions
}

type encryptedRepositoryInitializedMsg struct {
	keys    encryption.KeyPair
	profile repository.Profile
	err     error
}

func (m *Model) initEncryptedRepository(folder string, options repository.InitOptions) tea.Cmd {
	bus := m.buses.Commands
	profile := m.current.WithInit(options)
	return m.startOperation("Generating a key pair and initializing "+profile.Name()+" encrypted...", func(ctx context.Context) tea.Msg {
		events, err := bus.Handle(ctx, app.InitEncryptedRepositoryCommand{Profile: profile, KeysFolder: folder})
		msg := encryptedRepositoryInitializedMsg{err: err}
		for _, recorded := range events {
			switch typed := recorded.(type) {
			case encryption.KeysGenerated:
				msg.keys = typed.Keys
			case repository.Opened:
				msg.profile = typed.Profile
			}
		}
		return msg
	})
}

func (m *Model) handleEncryptedInitialized(msg *encryptedRepositoryInitializedMsg) tea.Cmd {
	m.finishOperation()
	if m.fail("initialize repository "+m.current.Name()+" encrypted", msg.err) {
		return nil
	}

	m.current = msg.profile
	m.message = fmt.Sprintf("Repository %s initialized encrypted (format version %d)", m.current.Name(), m.current.Init().FormatVersion())
	m.dialog = &infoDialog{title: "Repository " + m.current.Name() + " encrypted", body: encryptedInitSummary(m.current.Name(), msg.keys)}

	return m.loadSnapshots("")
}

func encryptedInitSummary(name string, keys encryption.KeyPair) string {
	var body strings.Builder

	body.WriteString(mutedStyle.Render("Private key") + "\n" + highlightStyle.Render(keys.PrivateKey()) + "\n")
	body.WriteString(mutedStyle.Render("Public key") + "\n" + highlightStyle.Render(keys.PublicKey()) + "\n\n")
	body.WriteString("This session already uses them. To open the repository next time, ")
	if name == environmentRepositoryName {
		body.WriteString("set:\n\n")
		body.WriteString(highlightStyle.Render("    STOWMARK_PUBLIC_KEY="+keys.PublicKey()) + "\n")
		body.WriteString(highlightStyle.Render("    STOWMARK_PRIVATE_KEY=" + keys.PrivateKey()))
	} else {
		body.WriteString("add to " + highlightStyle.Render(name) + "\nin repositories.yaml:\n\n")
		body.WriteString(configurationLines(keys.PublicKey(), keys.PrivateKey()))
	}
	body.WriteString("\n\n" + errorStyle.Render("Keep the private key safe: without it the repository cannot be read."))

	return body.String()
}

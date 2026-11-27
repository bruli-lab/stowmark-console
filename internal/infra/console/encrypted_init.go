package console

import (
	"context"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/bruli-lab/stowmark-console/internal/app"
	"github.com/bruli-lab/stowmark-console/internal/domain/encryption"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
)

type encryptedInitRequested struct{}

type initConfirmationRequested struct{}

type encryptedInitFolderChosen struct {
	folder string
}

type encryptedRepositoryInitializedMsg struct {
	keys    encryption.KeyPair
	profile repository.Profile
	err     error
}

func (m *Model) openEncryptedInitDialog() {
	m.dialog = newPromptDialog(
		"Initialize "+m.current.Name()+" encrypted",
		"Folder for the new key pair",
		defaultKeysFolder(),
		"[Tab] Complete  [↑↓] Suggestions  [Enter] Initialize  [Esc] Back",
		func(value string) (tea.Msg, error) {
			if _, err := describeKeysFolder(value); err != nil {
				return nil, err
			}
			folder, err := resolveLocalPath(value)
			if err != nil {
				return nil, err
			}
			return encryptedInitFolderChosen{folder: folder}, nil
		},
		emit(initConfirmationRequested{}),
	).withCompletion(
		func(value string) []string { return pathSuggestions(value, nil) },
		describeKeysFolder,
	)
}

func (m *Model) initEncryptedRepository(folder string) tea.Cmd {
	bus := m.buses.Commands
	profile := m.current
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
	m.message = "Repository " + m.current.Name() + " initialized encrypted (format version 3)"
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
		body.WriteString(highlightStyle.Render("    STOWMARK_PRIVATE_KEY="+keys.PrivateKey()) + "\n")
		body.WriteString(highlightStyle.Render("    STOWMARK_FORMAT_VERSION=3"))
	} else {
		body.WriteString("add to " + highlightStyle.Render(name) + "\nin repositories.yaml:\n\n")
		body.WriteString(configurationLines(keys.PublicKey(), keys.PrivateKey()) + "\n")
		body.WriteString(highlightStyle.Render("    format_version: 3"))
	}
	body.WriteString("\n\n" + errorStyle.Render("Keep the private key safe: without it the repository cannot be read."))

	return body.String()
}

package console

import (
	"context"
	"errors"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/bruli-lab/stowmark-console/internal/app"
	"github.com/bruli-lab/stowmark-console/internal/domain/encryption"
)

var errConfirmationMismatch = errors.New("the name does not match")

type keygenRequested struct{}

type rewrapRequested struct{}

type rekeyRequested struct{}

type keysFolderChosen struct {
	folder string
}

type rewrapRequest struct {
	oldPrivateKey string
	newPublicKey  string
}

type rewrapOldKeyChosen struct {
	oldPrivateKey string
}

type rewrapNewKeyChosen struct {
	request rewrapRequest
}

type rewrapConfirmed struct {
	request rewrapRequest
}

type rotationRequest struct {
	privateKey string
	publicKey  string
}

type rotationPrivateKeyChosen struct {
	privateKey string
}

type rotationPublicKeyChosen struct {
	request rotationRequest
}

type rotationConfirmed struct {
	request rotationRequest
}

func (m *Model) handleKeyIntent(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case keygenRequested:
		if m.requireRepository() {
			m.openKeygenDialog()
		}
	case rewrapRequested:
		if m.requireEncryptedRepository() {
			m.openKeyDialog("Rewrap the key of "+m.current.Name(), "Current private key", m.configuredPrivateKey(),
				func(path string) tea.Msg { return rewrapOldKeyChosen{oldPrivateKey: path} })
		}
	case rewrapOldKeyChosen:
		m.openKeyDialog("Rewrap the key of "+m.current.Name(), "New public key", "",
			func(path string) tea.Msg {
				return rewrapNewKeyChosen{request: rewrapRequest{oldPrivateKey: msg.oldPrivateKey, newPublicKey: path}}
			})
	case rewrapNewKeyChosen:
		m.openRewrapConfirmation(msg.request)
	case rekeyRequested:
		if m.requireEncryptedRepository() {
			m.openKeyDialog("Rotate the key of "+m.current.Name(), "Current private key", m.configuredPrivateKey(),
				func(path string) tea.Msg { return rotationPrivateKeyChosen{privateKey: path} })
		}
	case rotationPrivateKeyChosen:
		m.openKeyDialog("Rotate the key of "+m.current.Name(), "Public key for the new key generation",
			m.rotationPublicKey(msg.privateKey),
			func(path string) tea.Msg {
				return rotationPublicKeyChosen{request: rotationRequest{privateKey: msg.privateKey, publicKey: path}}
			})
	case rotationPublicKeyChosen:
		m.openRotationConfirmation(msg.request)
	default:
		return nil, false
	}

	return nil, true
}

func (m *Model) requireRepository() bool {
	if !m.connected() {
		m.message = "Choose a repository first"
		return false
	}

	return true
}

func (m *Model) requireEncryptedRepository() bool {
	if !m.requireRepository() {
		return false
	}

	if !m.current.Encrypted() {
		m.message = "Repository " + m.current.Name() + " is not encrypted"
		return false
	}

	return true
}

func (m *Model) configuredPrivateKey() string {
	if key := m.current.PrivateKey(); key != nil {
		return *key
	}

	return ""
}

func (m *Model) configuredPublicKey() string {
	if key := m.current.Init().PublicKey(); key != nil {
		return *key
	}

	return ""
}

func (m *Model) rotationPublicKey(privateKey string) string {
	if public := m.configuredPublicKey(); public != "" {
		return public
	}

	return pairedKey(privateKey, privateKeySuffix, publicKeySuffix)
}

func (m *Model) openKeygenDialog() {
	m.dialog = newPromptDialog(
		"Generate a key pair",
		"Folder for the new keys",
		defaultKeysFolder(),
		"[Tab] Complete  [↑↓] Suggestions  [Enter] Generate  [Esc] Cancel",
		func(value string) (tea.Msg, error) {
			if _, err := describeKeysFolder(value); err != nil {
				return nil, err
			}
			folder, err := resolveLocalPath(value)
			if err != nil {
				return nil, err
			}
			return keysFolderChosen{folder: folder}, nil
		},
		emit(statusMessage{text: "Key generation cancelled"}),
	).withCompletion(
		func(value string) []string { return pathSuggestions(value, nil) },
		describeKeysFolder,
	)
}

func (m *Model) openKeyDialog(title, label, value string, chosen func(path string) tea.Msg) {
	m.dialog = newKeyFileDialog(title, label, value, chosen, emit(statusMessage{text: "Key change cancelled"}))
}

func newKeyFileDialog(title, label, value string, chosen func(path string) tea.Msg, onCancel tea.Cmd) *promptDialog {
	return newPromptDialog(
		title,
		label,
		value,
		"[Tab] Complete  [↑↓] Suggestions  [Enter] Continue  [Esc] Cancel",
		func(value string) (tea.Msg, error) {
			if _, err := describeKeyFile(value); err != nil {
				return nil, err
			}
			path, err := resolveLocalPath(value)
			if err != nil {
				return nil, err
			}
			return chosen(path), nil
		},
		onCancel,
	).withCompletion(keyFileSuggestions, describeKeyFile)
}

func (m *Model) openRewrapConfirmation(request rewrapRequest) {
	var body strings.Builder

	body.WriteString("The repository key is wrapped again with the new public key.\n")
	body.WriteString("From then on only its private key opens the repository.\n\n")
	body.WriteString(mutedStyle.Render("Current private key") + "\n" + highlightStyle.Render(request.oldPrivateKey) + "\n")
	body.WriteString(mutedStyle.Render("New public key") + "\n" + highlightStyle.Render(request.newPublicKey))

	m.dialog = &confirmDialog{
		title:     "Rewrap the key of " + m.current.Name() + "?",
		body:      body.String(),
		warning:   "It takes an exclusive lock: no other Stowmark process may use the repository meanwhile.",
		hint:      "[y] Rewrap  [n/Esc] Cancel",
		onConfirm: emit(rewrapConfirmed{request: request}),
		onCancel:  emit(statusMessage{text: "Key rewrap cancelled"}),
	}
}

func (m *Model) openRotationConfirmation(request rotationRequest) {
	name := m.current.Name()

	var label strings.Builder

	label.WriteString("Every object is encrypted again with a new key generation, which can take long\n")
	label.WriteString("on big repositories. Esc cancels it and Stowmark discards the incomplete generation.\n\n")
	label.WriteString(mutedStyle.Render("Current private key") + "\n" + highlightStyle.Render(request.privateKey) + "\n")
	label.WriteString(mutedStyle.Render("Public key") + "\n" + highlightStyle.Render(request.publicKey) + "\n\n")
	label.WriteString(errorStyle.Render("It takes an exclusive lock: no other Stowmark process may use the repository meanwhile."))
	label.WriteString("\n\nType " + highlightStyle.Render(name) + " to confirm")

	m.dialog = newPromptDialog(
		"Rotate the key of "+name+"?",
		label.String(),
		"",
		"[Enter] Rotate  [Esc] Cancel",
		func(value string) (tea.Msg, error) {
			if value != name {
				return nil, fmt.Errorf("%w: type %s", errConfirmationMismatch, name)
			}
			return rotationConfirmed{request: request}, nil
		},
		emit(statusMessage{text: "Key rotation cancelled"}),
	)
}

func (m *Model) generateKeys(folder string) tea.Cmd {
	bus := m.buses.Commands
	return m.startOperation("Generating a key pair in "+folder+"...", func(ctx context.Context) tea.Msg {
		event, err := dispatch[encryption.KeysGenerated](ctx, bus, app.GenerateKeysCommand{Folder: folder})
		return keysGeneratedMsg{event: event, err: err}
	})
}

func (m *Model) rewrapKey(request rewrapRequest) tea.Cmd {
	bus := m.buses.Commands
	return m.startOperation("Rewrapping the key of "+m.current.Name()+"...", func(ctx context.Context) tea.Msg {
		event, err := dispatch[encryption.KeyRewrapped](ctx, bus, app.RewrapKeyCommand{
			OldPrivateKey: request.oldPrivateKey,
			NewPublicKey:  request.newPublicKey,
		})
		return keyRewrappedMsg{request: request, event: event, err: err}
	})
}

func (m *Model) rotateKey(request rotationRequest) tea.Cmd {
	bus := m.buses.Commands
	return m.startOperation("Rotating the key of "+m.current.Name()+" (encrypting every object again)...", func(ctx context.Context) tea.Msg {
		event, err := dispatch[encryption.KeyRotated](ctx, bus, app.RotateKeyCommand{
			PrivateKey: request.privateKey,
			PublicKey:  request.publicKey,
		})
		return keyRotatedMsg{request: request, event: event, err: err}
	})
}

func (m *Model) handleKeyResult(msg tea.Msg) tea.Cmd {
	m.finishOperation()

	switch msg := msg.(type) {
	case keysGeneratedMsg:
		if m.fail("generate the key pair", msg.err) {
			return nil
		}
		m.showGeneratedKeys(msg.event.Keys)

	case keyRewrappedMsg:
		if m.failWithKey("rewrap the key of "+m.current.Name(), msg.err, &msg.request.oldPrivateKey) {
			return nil
		}
		private := pairedKey(msg.request.newPublicKey, publicKeySuffix, privateKeySuffix)
		if m.matchesConfiguration(msg.request.newPublicKey, private) {
			m.message = "Key of " + m.current.Name() + " rewrapped to the configured key pair"
			return nil
		}
		m.showConfigurationChange("Key of "+m.current.Name()+" rewrapped", msg.request.newPublicKey, private)

	case keyRotatedMsg:
		if m.failWithKey("rotate the key of "+m.current.Name(), msg.err, &msg.request.privateKey) {
			return nil
		}
		if m.matchesConfiguration(msg.request.publicKey, msg.request.privateKey) {
			m.message = "Key of " + m.current.Name() + " rotated: the repository uses a new key generation"
			return nil
		}
		m.showConfigurationChange("Key of "+m.current.Name()+" rotated", msg.request.publicKey,
			pairedKey(msg.request.publicKey, publicKeySuffix, privateKeySuffix))
	}

	return nil
}

func (m *Model) matchesConfiguration(publicKey, privateKey string) bool {
	return publicKey == m.configuredPublicKey() && privateKey == m.configuredPrivateKey()
}

func (m *Model) showGeneratedKeys(keys encryption.KeyPair) {
	var body strings.Builder

	body.WriteString(mutedStyle.Render("Private key") + "\n" + highlightStyle.Render(keys.PrivateKey()) + "\n")
	body.WriteString(mutedStyle.Render("Public key") + "\n" + highlightStyle.Render(keys.PublicKey()) + "\n\n")
	body.WriteString("To encrypt a repository with them, add to its entry in repositories.yaml:\n\n")
	body.WriteString(configurationLines(keys.PublicKey(), keys.PrivateKey()))
	body.WriteString("\n\n" + errorStyle.Render("Keep the private key safe: without it the repository cannot be read."))

	m.dialog = &infoDialog{title: "Key pair generated", body: body.String()}
}

func (m *Model) showConfigurationChange(title, publicKey, privateKey string) {
	if privateKey == "" {
		privateKey = "<the private key of " + publicKey + ">"
	}

	var body strings.Builder

	body.WriteString("Update repository " + highlightStyle.Render(m.current.Name()) + " in repositories.yaml:\n\n")
	body.WriteString(configurationLines(publicKey, privateKey))
	body.WriteString("\n\nThen restart the console: this session still uses the previous private key.")

	m.dialog = &infoDialog{title: title, body: body.String()}
}

func configurationLines(publicKey, privateKey string) string {
	return highlightStyle.Render("    public_key: "+publicKey) + "\n" + highlightStyle.Render("    private_key: "+privateKey)
}

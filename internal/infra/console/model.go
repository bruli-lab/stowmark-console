package console

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/bruli-lab/stowmark-console/internal/app"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

type Model struct {
	ctx             context.Context
	handlers        app.Handlers
	settings        Settings
	width           int
	height          int
	snapshots       *snapshotsScreen
	screens         []screen
	dialog          dialog
	command         *commandLine
	loading         bool
	progress        string
	cancelOperation context.CancelFunc
	message         string
	err             error
}

func New(ctx context.Context, settings Settings, handlers app.Handlers) Model {
	return Model{
		ctx:       ctx,
		handlers:  handlers,
		settings:  settings,
		snapshots: newSnapshotsScreen(),
		loading:   true,
	}
}

func (m *Model) Init() tea.Cmd {
	return m.checkRepository()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeScreens()
		return m, nil

	case tea.KeyPressMsg:
		cmd := m.handleKey(msg)
		return m, cmd

	case tea.PasteMsg:
		cmd := m.handleInput(msg)
		return m, cmd
	}

	if cmd, handled := m.handleIntent(msg); handled {
		return m, cmd
	}

	cmd := m.handleResult(msg)

	return m, cmd
}

func (m *Model) handleKey(msg tea.KeyPressMsg) tea.Cmd {
	switch {
	case m.dialog != nil || m.command != nil:
		return m.handleInput(msg)
	case msg.String() == "ctrl+c":
		return tea.Quit
	case m.loading:
		m.handleLoadingKey(msg)
		return nil
	case msg.String() == ":":
		m.command = newCommandLine()
		return nil
	}

	m.err = nil
	m.message = ""

	return m.top().update(msg)
}

func (m *Model) handleInput(msg tea.Msg) tea.Cmd {
	switch {
	case m.dialog != nil:
		closed, cmd := m.dialog.update(msg)
		if closed {
			m.dialog = nil
		}
		return cmd

	case m.command != nil:
		closed, cmd := m.command.update(msg)
		if closed {
			m.command = nil
		}
		return cmd
	}

	return nil
}

func (m *Model) handleLoadingKey(msg tea.KeyPressMsg) {
	if msg.String() == "esc" && m.cancelOperation != nil {
		m.cancelOperation()
		m.progress = "Cancelling..."
	}
}

func (m *Model) top() screen {
	if len(m.screens) == 0 {
		return m.snapshots
	}

	return m.screens[len(m.screens)-1]
}

func (m *Model) push(next screen) {
	m.screens = append(m.screens, next)
	m.resizeScreens()
}

func (m *Model) pop() {
	if len(m.screens) > 0 {
		m.screens = m.screens[:len(m.screens)-1]
	}
}

func (m *Model) showManifest(manifest snapshot.Manifest) {
	if files, ok := m.top().(*filesScreen); ok && files.snapshotID() == manifest.SnapshotID() {
		files.setManifest(manifest)
		return
	}

	m.push(newFilesScreen(manifest))
}

func (m *Model) resizeScreens() {
	width, height := m.contentSize()

	m.snapshots.resize(width, height)
	for _, item := range m.screens {
		item.resize(width, height)
	}
}

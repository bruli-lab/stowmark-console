package console

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/bruli-lab/stowmark-console/internal/app"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/domain/snapshot"
)

type Model struct {
	ctx             context.Context
	buses           app.Buses
	repositories    []repository.Profile
	current         repository.Profile
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
	openDirectory   string
	opener          FileOpener
	splash          *splash
}

type FileOpener func(path string) error

type Option func(*Model)

func WithFileOpener(directory string, opener FileOpener) Option {
	return func(m *Model) {
		m.openDirectory = directory
		m.opener = opener
	}
}

func New(ctx context.Context, buses app.Buses, options ...Option) Model {
	model := Model{
		ctx:       ctx,
		buses:     buses,
		snapshots: newSnapshotsScreen(),
		loading:   true,
	}

	for _, option := range options {
		option(&model)
	}

	return model
}

func (m *Model) Init() tea.Cmd {
	if m.splash != nil {
		return tea.Batch(m.loadRepositories(), m.splash.timer())
	}

	return m.loadRepositories()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeScreens()
		return m, nil

	case splashFinishedMsg:
		m.splash = nil
		return m, nil

	case tea.KeyPressMsg:
		if m.splash != nil {
			cmd := m.skipSplash(msg)
			return m, cmd
		}
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

func (m *Model) connected() bool {
	return m.current.Name() != ""
}

func (m *Model) showRepositories() {
	m.screens = []screen{newRepositoriesScreen(m.repositories, m.current.Name())}
	m.resizeScreens()
}

func (m *Model) resizeScreens() {
	width, height := m.contentSize()

	m.snapshots.resize(width, height)
	for _, item := range m.screens {
		item.resize(width, height)
	}
}

package console

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bruli-lab/stowmark-console.git/internal/config"
	"github.com/bruli-lab/stowmark-console.git/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console.git/internal/infra/disk"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	ctx               context.Context
	conf              *config.Config
	folderRepo        repository.FolderRepository
	entries           []repository.Entry
	cursor            int
	currentPath       string
	previewPath       string
	previewLines      []string
	previewOffset     int
	previewing        bool
	width             int
	height            int
	loading           bool
	err               error
	commandMode       bool
	commandInput      string
	message           string
	repoPath          string
	confirmingRestore bool
	pendingSnapshotID string
	creatingSnapshot  bool
	snapshotOrigin    string
	snapshotFormError string
	preserveMessage   bool
}

func New(ctx context.Context, conf *config.Config) Model {
	return Model{
		ctx:         ctx,
		conf:        conf,
		loading:     true,
		folderRepo:  disk.NewFolderRepository(),
		currentPath: filepath.Clean(conf.Repository()),
		repoPath:    conf.Repository(),
	}
}

func (m *Model) Init() tea.Cmd {
	configPath := filepath.Join(m.repoPath, "config.json")

	_, err := os.Stat(configPath)
	switch {
	case err == nil:
		return loadRepositoryContent(
			m.ctx,
			m.folderRepo,
			m.currentPath,
		)

	case os.IsNotExist(err):
		m.loading = true
		m.message = "Initializing repository..."

		return initRepositoryCmd(
			m.ctx,
			m.repoPath,
		)

	default:
		return func() tea.Msg {
			return repositoryLoadedMsg{
				path: m.repoPath,
				err:  err,
			}
		}
	}
}

func (m *Model) openSelectedFile() tea.Cmd {
	if len(m.entries) == 0 ||
		m.cursor < 0 ||
		m.cursor >= len(m.entries) {
		return nil
	}

	entry := m.entries[m.cursor]
	if entry.IsDir() {
		return nil
	}

	filePath := filepath.Join(
		m.currentPath,
		entry.Name(),
	)

	isText, err := m.folderRepo.IsTextFile(m.ctx, filePath)
	if err != nil {
		return func() tea.Msg {
			return editorFinishedMsg{err: err}
		}
	}

	if !isText {
		return func() tea.Msg {
			return editorFinishedMsg{
				err: fmt.Errorf("%q no és un fitxer de text", entry.Name()),
			}
		}
	}

	command, err := editorCommand(filePath)
	if err != nil {
		return func() tea.Msg {
			return editorFinishedMsg{err: err}
		}
	}

	return tea.ExecProcess(command, func(err error) tea.Msg {
		return editorFinishedMsg{err: err}
	})
}

func (m *Model) requestRestore(snapshotID string) {
	m.confirmingRestore = true
	m.pendingSnapshotID = snapshotID
	m.err = nil
	m.message = ""
}

func (m *Model) activeSection() string {
	rootPath := filepath.Clean(m.conf.Repository())
	currentPath := filepath.Clean(m.currentPath)

	relativePath, err := filepath.Rel(rootPath, currentPath)
	if err != nil || relativePath == "." {
		return "repository"
	}

	firstDirectory := strings.Split(relativePath, string(filepath.Separator))[0]

	switch strings.ToLower(firstDirectory) {
	case "snapshots":
		return "snapshots"
	case "objects":
		return "objects"
	default:
		return "repository"
	}
}

func (m *Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case filePreviewLoadedMsg:
		m.loading = false
		m.err = msg.err

		if msg.err != nil {
			return m, nil
		}

		m.previewPath = msg.path
		m.previewLines = strings.Split(msg.content, "\n")
		m.previewOffset = 0
		m.previewing = true

	case snapshotVerifiedMsg:
		m.loading = false

		if msg.err != nil {
			m.err = fmt.Errorf(
				"failed to verify snapshot %s: %w",
				msg.snapshotID,
				msg.err,
			)
			m.message = ""
			return m, nil
		}

		m.err = nil
		m.message = fmt.Sprintf(
			"Snapshot %s verified successfully",
			msg.snapshotID,
		)

		return m, nil

	case snapshotRestoredMsg:
		m.loading = false

		if msg.err != nil {
			m.err = fmt.Errorf(
				"failed to restore snapshot %s: %w",
				msg.snapshotID,
				msg.err,
			)
			m.message = ""
			return m, nil
		}

		m.err = nil
		m.message = fmt.Sprintf(
			"Snapshot %s restored successfully",
			msg.snapshotID,
		)

		return m, nil

	case initRepositoryMsg:
		m.loading = false
		m.err = msg.err

		if msg.err != nil {
			return m, nil
		}

		m.message = "Repository initialized successfully"
		m.loading = true

		return m, loadRepositoryContent(
			m.ctx,
			m.folderRepo,
			m.repoPath,
		)
	case snapshotCreatedMsg:
		m.loading = false

		if msg.err != nil {
			m.err = fmt.Errorf("failed to create snapshot: %w", msg.err)
			m.message = ""
			return m, nil
		}

		m.err = nil
		m.message = fmt.Sprintf(
			"Snapshot %s created: %d files, %s",
			msg.snapshotID,
			msg.fileCount,
			formatSize(msg.totalSize),
		)
		m.preserveMessage = true
		m.loading = true

		return m, loadRepositoryContent(
			m.ctx,
			m.folderRepo,
			filepath.Join(filepath.Clean(m.repoPath), "snapshots"),
		)

	case repositoryLoadedMsg:
		m.loading = false
		m.err = msg.err
		if !m.preserveMessage {
			m.message = ""
		}
		m.preserveMessage = false
		m.commandMode = false
		m.commandInput = ""

		if msg.err != nil {
			return m, nil
		}

		m.currentPath = msg.path
		m.entries = msg.entries
		m.cursor = 0

	case editorFinishedMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("opening file with vim: %w", msg.err)
			return m, nil
		}

		m.loading = true
		m.err = nil

		return m, loadRepositoryContent(
			m.ctx,
			m.folderRepo,
			m.currentPath,
		)

	case tea.KeyMsg:
		if m.creatingSnapshot {
			switch msg.String() {
			case "ctrl+c", "esc":
				m.creatingSnapshot = false
				m.snapshotOrigin = ""
				m.snapshotFormError = ""
				m.message = "Snapshot creation cancelled"

				return m, nil

			case "enter":
				origin := strings.TrimSpace(m.snapshotOrigin)
				if origin == "" {
					m.snapshotFormError = "Origin is required"
					return m, nil
				}

				m.creatingSnapshot = false
				m.snapshotOrigin = ""
				m.snapshotFormError = ""
				m.loading = true
				m.err = nil
				m.message = "Creating snapshot..."

				return m, createSnapshotCmd(
					m.ctx,
					m.repoPath,
					origin,
				)

			case "backspace":
				runes := []rune(m.snapshotOrigin)
				if len(runes) > 0 {
					m.snapshotOrigin = string(runes[:len(runes)-1])
				}
				m.snapshotFormError = ""

			default:
				if msg.Type == tea.KeyRunes {
					m.snapshotOrigin += string(msg.Runes)
					m.snapshotFormError = ""
				}
			}

			return m, nil
		}

		if m.confirmingRestore {
			switch msg.String() {
			case "y", "Y", "enter":
				snapshotID := m.pendingSnapshotID

				m.confirmingRestore = false
				m.pendingSnapshotID = ""
				m.loading = true
				m.err = nil
				m.message = "Restoring snapshot..."

				return m, restoreSnapshotCmd(
					m.ctx,
					m.repoPath,
					snapshotID,
				)

			case "n", "N", "esc", "q":
				m.confirmingRestore = false
				m.pendingSnapshotID = ""
				m.message = "Restore cancelled"
				m.err = nil

				return m, nil
			}

			return m, nil
		}
		if m.commandMode {
			switch msg.String() {
			case "esc":
				m.commandMode = false
				m.commandInput = ""

			case "enter":
				command := m.commandInput
				m.commandMode = false
				m.commandInput = ""

				if command == "q" {
					return m, tea.Quit
				}

			case "backspace":
				runes := []rune(m.commandInput)

				if len(runes) > 0 {
					m.commandInput = string(runes[:len(runes)-1])
				} else {
					m.commandMode = false
				}

			default:
				if msg.Type == tea.KeyRunes {
					m.commandInput += string(msg.Runes)
				}
			}

			return m, nil
		}

		if m.previewing {
			switch msg.String() {
			case "esc", "backspace":
				m.previewing = false
				m.previewPath = ""
				m.previewLines = nil
				m.previewOffset = 0
				m.err = nil

			case ":":
				m.commandMode = true
				m.commandInput = ""

			case "up", "k":
				if m.previewOffset > 0 {
					m.previewOffset--
				}

			case "down", "j":
				maxOffset := max(
					0,
					len(m.previewLines)-m.previewVisibleLines(),
				)

				if m.previewOffset < maxOffset {
					m.previewOffset++
				}

			case "home", "g":
				m.previewOffset = 0

			case "end", "G":
				m.previewOffset = max(
					0,
					len(m.previewLines)-m.previewVisibleLines(),
				)

			case "ctrl+c":
				return m, tea.Quit
			}

			return m, nil
		}

		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case ":":
			m.commandMode = true
			m.commandInput = ""

		case "c":
			if !m.isSnapshotsDirectory() {
				return m, nil
			}

			m.creatingSnapshot = true
			m.snapshotOrigin = ""
			m.snapshotFormError = ""
			m.message = ""
			m.err = nil

			return m, nil

		case "v":
			snapshotPath, ok := m.selectedSnapshotPath()
			if !ok {
				return m, nil
			}
			snapshotFile := filepath.Base(snapshotPath)
			snapshotID := strings.TrimSuffix(snapshotFile, filepath.Ext(snapshotFile))

			m.loading = true
			m.message = "Verifying snapshot..."
			return m, verifySnapshotCmd(m.ctx, m.repoPath, snapshotID)

		case "x":
			snapshotPath, ok := m.selectedSnapshotPath()
			if !ok {
				return m, nil
			}

			snapshotFile := filepath.Base(snapshotPath)
			snapshotID := strings.TrimSuffix(
				snapshotFile,
				filepath.Ext(snapshotFile),
			)

			m.requestRestore(snapshotID)

			return m, nil

		case "esc", "backspace", "left", "h":
			rootPath := filepath.Clean(m.conf.Repository())
			currentPath := filepath.Clean(m.currentPath)

			if currentPath != rootPath {
				m.loading = true
				m.err = nil
				m.message = ""

				cmd := m.openParentDirectory()
				return m, cmd
			}

		case "enter":
			if len(m.entries) == 0 ||
				m.cursor < 0 ||
				m.cursor >= len(m.entries) {
				return m, nil
			}

			m.loading = true
			m.err = nil
			m.message = ""

			if m.entries[m.cursor].IsDir() {
				cmd := m.openSelectedDirectory()
				return m, cmd
			}

			cmd := m.previewSelectedFile()
			return m, cmd

		case "e":
			m.message = ""
			cmd := m.openSelectedFile()
			return m, cmd

		case "r":
			m.loading = true
			m.err = nil
			m.message = ""

			return m, loadRepositoryContent(
				m.ctx,
				m.folderRepo,
				m.currentPath,
			)

		case "up", "k":
			m.message = ""

			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			m.message = ""

			if m.cursor < len(m.entries)-1 {
				m.cursor++
			}

		case "home", "g":
			m.message = ""
			m.cursor = 0

		case "end", "G":
			m.message = ""

			if len(m.entries) > 0 {
				m.cursor = len(m.entries) - 1
			}
		}
	}

	return m, nil
}

func (m *Model) previewVisibleLines() int {
	return max(1, m.height-8)
}

func (m *Model) isSnapshotsDirectory() bool {
	snapshotsPath := filepath.Join(
		filepath.Clean(m.conf.Repository()),
		"snapshots",
	)

	return filepath.Clean(m.currentPath) == snapshotsPath
}

func (m *Model) selectedSnapshotPath() (string, bool) {
	if !m.isSnapshotsDirectory() ||
		len(m.entries) == 0 ||
		m.cursor < 0 ||
		m.cursor >= len(m.entries) ||
		m.entries[m.cursor].IsDir() {
		return "", false
	}

	return filepath.Join(
		m.currentPath,
		m.entries[m.cursor].Name(),
	), true
}

func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Starting Stowmark Console..."
	}

	header := m.renderHeader()
	footer := m.renderFooter()

	bodyHeight := max(
		1,
		m.height-lipgloss.Height(header)-lipgloss.Height(footer),
	)

	const sidebarTotalWidth = 28

	sidebarFrameWidth := sidebarStyle.GetHorizontalFrameSize()
	sidebarFrameHeight := sidebarStyle.GetVerticalFrameSize()
	contentFrameWidth := contentStyle.GetHorizontalFrameSize()
	contentFrameHeight := contentStyle.GetVerticalFrameSize()

	sidebarWidth := max(
		1,
		sidebarTotalWidth-sidebarFrameWidth,
	)
	contentWidth := max(
		1,
		m.width-sidebarTotalWidth-contentFrameWidth,
	)

	sidebarHeight := max(
		1,
		bodyHeight-sidebarFrameHeight,
	)
	contentHeight := max(
		1,
		bodyHeight-contentFrameHeight,
	)

	sidebar := sidebarStyle.
		Width(sidebarWidth).
		Height(sidebarHeight).
		Render(m.renderSidebar(sidebarHeight))

	content := contentStyle.
		Width(contentWidth).
		Height(contentHeight).
		Render(m.renderContent(contentHeight))

	body := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sidebar,
		content,
	)

	view := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		body,
		footer,
	)

	if m.confirmingRestore {
		confirmation := confirmRestoreStyle.Render(
			fmt.Sprintf(
				"Restore snapshot %s?\n\n[y/Enter] Confirm  [n/Esc] Cancel",
				m.pendingSnapshotID,
			),
		)

		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			confirmation,
		)
	}

	if m.creatingSnapshot {
		origin := m.snapshotOrigin
		if origin == "" {
			origin = " "
		}

		var form strings.Builder
		form.WriteString(headerStyle.Render("Create snapshot"))
		form.WriteString("\n\nOrigin\n")
		form.WriteString(snapshotInputStyle.Render(origin + "█"))

		if m.snapshotFormError != "" {
			form.WriteString("\n\n")
			form.WriteString(errorStyle.Render(m.snapshotFormError))
		}

		form.WriteString("\n\n[Enter] Create  [Esc] Cancel")

		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			snapshotFormStyle.Render(form.String()),
		)
	}

	return view
}

func (m *Model) renderHeader() string {
	title := titleStyle.Render("STOWMARK")

	repositoryPath := repositoryStyle.Render(m.currentPath)

	spaces := max(
		1,
		m.width-lipgloss.Width(title)-lipgloss.Width(repositoryPath)-2,
	)

	return title + strings.Repeat(" ", spaces) + repositoryPath
}

func (m *Model) renderSidebar(availableHeight int) string {
	activeSection := m.activeSection()

	items := []struct {
		label    string
		selected bool
	}{
		{
			label:    "Repository",
			selected: activeSection == "repository",
		},
		{
			label:    "Snapshots",
			selected: activeSection == "snapshots",
		},
		{
			label:    "Objects",
			selected: activeSection == "objects",
		},
	}

	var builder strings.Builder

	if availableHeight >= 16 {
		builder.WriteString(logoStyle.Render(stowmarkLogo))
		builder.WriteString("\n\n")
	}

	builder.WriteString(headerStyle.Render("Navigation"))
	builder.WriteString("\n\n")

	for _, item := range items {
		style := menuStyle
		prefix := "  "

		if item.selected {
			style = selectedMenuStyle
			prefix = "› "
		}

		builder.WriteString(style.Render(prefix + item.label))
		builder.WriteString("\n")
	}

	return builder.String()
}

func (m *Model) previewSelectedFile() tea.Cmd {
	if len(m.entries) == 0 ||
		m.cursor < 0 ||
		m.cursor >= len(m.entries) {
		return nil
	}

	entry := m.entries[m.cursor]
	if entry.IsDir() {
		return nil
	}

	filePath := filepath.Join(m.currentPath, entry.Name())

	return func() tea.Msg {
		isText, err := m.folderRepo.IsTextFile(m.ctx, filePath)
		if err != nil {
			return filePreviewLoadedMsg{
				path: filePath,
				err:  err,
			}
		}

		if !isText {
			return filePreviewLoadedMsg{
				path: filePath,
				err: fmt.Errorf(
					"%q no és un fitxer de text",
					entry.Name(),
				),
			}
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			return filePreviewLoadedMsg{
				path: filePath,
				err: fmt.Errorf(
					"read file %q: %w",
					filePath,
					err,
				),
			}
		}

		return filePreviewLoadedMsg{
			path:    filePath,
			content: string(content),
		}
	}
}

func (m *Model) renderFilePreview(availableHeight int) string {
	var builder strings.Builder

	builder.WriteString(
		headerStyle.Render(filepath.Base(m.previewPath)),
	)
	builder.WriteString("\n\n")

	visibleLines := max(1, availableHeight-3)
	start := min(m.previewOffset, len(m.previewLines))
	end := min(len(m.previewLines), start+visibleLines)

	lineNumberWidth := len(fmt.Sprintf("%d", len(m.previewLines)))
	contentWidth := max(1, m.width-38-lineNumberWidth)

	for index := start; index < end; index++ {
		line := strings.ReplaceAll(m.previewLines[index], "\t", "    ")
		line = truncate(line, contentWidth)

		_, _ = fmt.Fprintf(
			&builder,
			"%*d │ %s\n",
			lineNumberWidth,
			index+1,
			line,
		)
	}

	return builder.String()
}

func (m *Model) renderContent(availableHeight int) string {
	if m.previewing {
		return m.renderFilePreview(availableHeight)
	}
	var builder strings.Builder

	builder.WriteString(headerStyle.Render("Repository contents"))
	builder.WriteString("\n\n")

	if m.loading {
		builder.WriteString("Loading repository...")
		return builder.String()
	}

	if m.err != nil {
		builder.WriteString(errorStyle.Render(m.err.Error()))
		return builder.String()
	}

	if len(m.entries) == 0 {
		builder.WriteString("The repository is empty.")
		return builder.String()
	}

	_, _ = fmt.Fprintf(
		&builder,
		"  %-40s %-12s %12s\n",
		"NAME",
		"TYPE",
		"SIZE",
	)

	builder.WriteString(
		strings.Repeat("─", min(68, max(1, m.width-32))),
	)
	builder.WriteString("\n")

	maxRows := max(1, availableHeight-6)
	start := 0

	if m.cursor >= maxRows {
		start = m.cursor - maxRows + 1
	}

	end := min(len(m.entries), start+maxRows)

	for index := start; index < end; index++ {
		entry := m.entries[index]

		entryType := "file"
		size := formatSize(entry.Size())

		if entry.IsDir() {
			entryType = "directory"
			size = "-"
		}

		row := fmt.Sprintf(
			"  %-40s %-12s %12s",
			truncate(entry.Name(), 40),
			entryType,
			size,
		)

		if index == m.cursor {
			row = selectedRowStyle.Render("›" + row[1:])
		}

		builder.WriteString(row)
		builder.WriteString("\n")
	}

	return builder.String()
}

func (m *Model) renderFooter() string {
	var help string

	if m.isSnapshotsDirectory() {
		help = "↑/↓ navigate • enter open • c create • v verify • x restore • esc back • :q exit"
	} else {
		help = "↑/↓ navigate • enter open • e edit • r refresh • :q exit"
	}

	if m.commandMode {
		help = ":" + m.commandInput
	} else if m.message != "" {
		help = m.message
	}

	return footerStyle.
		Width(m.width).
		Render(help)
}

func (m *Model) openSelectedDirectory() tea.Cmd {
	if len(m.entries) == 0 ||
		m.cursor < 0 ||
		m.cursor >= len(m.entries) {
		return nil
	}

	entry := m.entries[m.cursor]
	if !entry.IsDir() {
		return nil
	}

	nextPath := filepath.Join(m.currentPath, entry.Name())

	return loadRepositoryContent(
		m.ctx,
		m.folderRepo,
		nextPath,
	)
}

func (m *Model) openParentDirectory() tea.Cmd {
	rootPath := filepath.Clean(m.conf.Repository())
	currentPath := filepath.Clean(m.currentPath)

	if currentPath == rootPath {
		return nil
	}

	parentPath := filepath.Dir(currentPath)

	relativePath, err := filepath.Rel(rootPath, parentPath)
	if err != nil ||
		relativePath == ".." ||
		strings.HasPrefix(
			relativePath,
			".."+string(filepath.Separator),
		) {
		return nil
	}

	return loadRepositoryContent(
		m.ctx,
		m.folderRepo,
		parentPath,
	)
}

func formatSize(size int64) string {
	const unit = 1024

	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	divisor := int64(unit)
	exponent := 0

	for value := size / unit; value >= unit; value /= unit {
		divisor *= unit
		exponent++
	}

	units := "KMGTPE"

	return fmt.Sprintf(
		"%.1f %ciB",
		float64(size)/float64(divisor),
		units[exponent],
	)
}

func truncate(value string, width int) string {
	if len(value) <= width {
		return value
	}

	if width <= 1 {
		return value[:width]
	}

	return value[:width-1] + "…"
}

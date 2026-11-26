package console

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/infra/disk"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	ctx             context.Context
	folderRepo      repository.FolderRepository
	repoPath        string
	entries         []repository.Entry
	cursor          int
	currentPath     string
	previewPath     string
	previewLines    []string
	previewOffset   int
	previewing      bool
	width           int
	height          int
	loading         bool
	err             error
	commandMode     bool
	commandInput    string
	message         string
	preserveMessage bool
	confirmingInit  bool
	initWarning     string
	snapshotForm    *textPrompt
	restoreForm     *textPrompt
	restore         *restoreRequest
	privateKey      *string
	initSettings    InitSettings
}

type restoreRequest struct {
	snapshotID  string
	destination string
	warning     string
}

func New(ctx context.Context, settings Settings) Model {
	return Model{
		ctx:          ctx,
		loading:      true,
		folderRepo:   disk.NewFolderRepository(),
		currentPath:  filepath.Clean(settings.RepositoryPath),
		repoPath:     settings.RepositoryPath,
		privateKey:   settings.PrivateKey,
		initSettings: settings.Init,
	}
}

func (m *Model) rootPath() string {
	return filepath.Clean(m.repoPath)
}

func (m *Model) confirmingRestore() bool {
	return m.restore != nil && m.restoreForm == nil
}

func (m *Model) selectedEntry() (repository.Entry, bool) {
	if m.cursor < 0 || m.cursor >= len(m.entries) {
		return repository.Entry{}, false
	}

	return m.entries[m.cursor], true
}

func (m *Model) selectedSnapshotID() (string, bool) {
	entry, ok := m.selectedEntry()
	if !ok || !m.isSnapshotsDirectory() || entry.IsDir() {
		return "", false
	}

	return strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())), true
}

func (m *Model) activeSection() string {
	rootPath := m.rootPath()
	currentPath := filepath.Clean(m.currentPath)

	relativePath, err := filepath.Rel(rootPath, currentPath)
	if err != nil || relativePath == "." {
		return "repository"
	}

	firstDirectory, _, _ := strings.Cut(relativePath, string(filepath.Separator))

	switch strings.ToLower(firstDirectory) {
	case "snapshots":
		return "snapshots"
	case "objects":
		return "objects"
	default:
		return "repository"
	}
}

func (m *Model) previewVisibleLines() int {
	return max(1, m.height-8)
}

func (m *Model) isSnapshotsDirectory() bool {
	snapshotsPath := filepath.Join(m.rootPath(), "snapshots")

	return filepath.Clean(m.currentPath) == snapshotsPath
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
	entry, ok := m.selectedEntry()
	if !ok || entry.IsDir() {
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

func (m *Model) openSelectedDirectory() tea.Cmd {
	entry, ok := m.selectedEntry()
	if !ok || !entry.IsDir() {
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
	rootPath := m.rootPath()
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

package console

import (
	"fmt"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
)

const currentMarker = "● "

var repositoryColumns = []columnSpec{
	{title: "NAME", width: 22},
	{title: "TYPE", width: 8},
	{title: "LOCATION"},
}

type repositoriesScreen struct {
	profiles []repository.Profile
	current  string
	table    table.Model
}

func newRepositoriesScreen(profiles []repository.Profile, current string) *repositoriesScreen {
	screen := &repositoriesScreen{profiles: profiles, current: current, table: newTable()}

	cursor := 0
	rows := make([]table.Row, len(profiles))

	for i, profile := range profiles {
		marker := "  "
		if profile.Name() == current {
			marker = currentMarker
			cursor = i
		}
		rows[i] = tableRow(repositoryColumns, marker+profile.Name(), profile.Kind(), profile.Location())
	}

	selectRow(&screen.table, rows, cursor)

	return screen
}

func (s *repositoriesScreen) name() string {
	return "Repositories"
}

func (s *repositoriesScreen) location() string {
	return ""
}

func (s *repositoriesScreen) help() string {
	if s.current == "" {
		return "↑↓ Navigate   ↵ Open"
	}

	return "↑↓ Navigate   ↵ Open   [esc] Back"
}

func (s *repositoriesScreen) resize(width, height int) {
	resizeTable(&s.table, repositoryColumns, width, height)
}

func (s *repositoriesScreen) selected() (repository.Profile, bool) {
	cursor := s.table.Cursor()
	if cursor < 0 || cursor >= len(s.profiles) {
		return repository.Profile{}, false
	}

	return s.profiles[cursor], true
}

func (s *repositoriesScreen) update(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "enter", "right", "l":
		selected, ok := s.selected()
		switch {
		case !ok:
			return nil
		case selected.Name() == s.current:
			return emit(screenCloseRequested{})
		default:
			return emit(repositoryOpenRequested{name: selected.Name()})
		}

	case "esc", "backspace", "left", "h", "q":
		if s.current == "" {
			return nil
		}
		return emit(screenCloseRequested{})
	}

	var cmd tea.Cmd
	s.table, cmd = s.table.Update(msg)

	return cmd
}

func (s *repositoriesScreen) view() string {
	title := headerStyle.Render(fmt.Sprintf("Repositories (%d)", len(s.profiles)))
	if s.current == "" {
		title += "  " + mutedStyle.Render("Choose the repository to open")
	}

	if len(s.profiles) == 0 {
		return title + "\n\nNo repositories configured."
	}

	return title + "\n\n" + s.table.View()
}

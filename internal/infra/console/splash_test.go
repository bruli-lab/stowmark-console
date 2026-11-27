package console

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
)

func newSplashHarness(t *testing.T) *harness {
	t.Helper()

	h := newHarness(t, true)
	WithVersion("v1.2.3")(h.model)
	WithSplash(time.Hour)(h.model)

	return h
}

func TestConsoleSplash(t *testing.T) {
	t.Parallel()

	t.Run(`Given a console with a splash screen
	When it starts
	Then only the logo and the version are shown while the repositories load underneath`, func(t *testing.T) {
		t.Parallel()
		h := newSplashHarness(t)

		h.run(h.model.loadRepositories())

		view := h.view()
		require.Contains(t, view, "▀")
		require.Contains(t, ansi.Strip(view), "v1.2.3")
		require.NotContains(t, ansi.Strip(view), "Navigation")
		require.Equal(t, 40, lipgloss.Height(view))
		require.Equal(t, 120, lipgloss.Width(view))
		require.Equal(t, "Snapshots", h.model.top().name())
	})

	t.Run(`Given the splash screen
	When its time is over
	Then the console is shown`, func(t *testing.T) {
		t.Parallel()
		h := newSplashHarness(t)
		h.run(h.model.loadRepositories())

		h.model.Update(splashFinishedMsg{})

		require.Contains(t, ansi.Strip(h.view()), "Navigation")
	})

	t.Run(`Given the splash screen
	When a key is pressed
	Then the splash is skipped without passing the key to the screens`, func(t *testing.T) {
		t.Parallel()
		h := newSplashHarness(t)
		h.run(h.model.loadRepositories())

		cmd := h.press(runes("c"))

		require.Nil(t, cmd)
		require.Nil(t, h.model.dialog)
		require.Contains(t, ansi.Strip(h.view()), "Navigation")
	})

	t.Run(`Given the splash screen
	When ctrl+c is pressed
	Then the console quits`, func(t *testing.T) {
		t.Parallel()
		h := newSplashHarness(t)

		require.True(t, isQuit(h.press(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})))
	})

	tests := []struct {
		name      string
		width     int
		height    int
		logoLines int
	}{
		{name: "a tall terminal, then the large logo is used", width: 120, height: 40, logoLines: 19},
		{name: "a medium terminal, then the small logo is used", width: 80, height: 16, logoLines: 12},
		{name: "a tiny terminal, then only the version is shown", width: 40, height: 6},
	}

	for _, tt := range tests {
		t.Run(`Given the splash screen in `+tt.name, func(t *testing.T) {
			t.Parallel()
			s := &splash{}

			view := s.view("v1.2.3", tt.width, tt.height)

			require.Equal(t, tt.logoLines, logoRows(view))
			require.Equal(t, tt.height, lipgloss.Height(view))
			for line := range strings.SplitSeq(view, "\n") {
				require.Equal(t, tt.width, ansi.StringWidth(line))
			}
			require.Contains(t, ansi.Strip(view), "v1.2.3")
		})
	}
}

func logoRows(view string) int {
	rows := 0
	for line := range strings.SplitSeq(view, "\n") {
		if strings.Contains(line, "▀") {
			rows++
		}
	}

	return rows
}

func TestConsoleHeaderVersion(t *testing.T) {
	t.Parallel()

	t.Run(`Given a console with a version
	When the header is rendered
	Then the version follows the name`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true)
		WithVersion("v1.2.3")(h.model)
		h.started()

		require.Contains(t, ansi.Strip(h.model.renderHeader()), " STOWMARK  v1.2.3  local  default ")
	})

	t.Run(`Given a console without a version
	When the header is rendered
	Then only the name is shown`, func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, true).started()

		require.Contains(t, ansi.Strip(h.model.renderHeader()), " STOWMARK   local  default ")
	})
}

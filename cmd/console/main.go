package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/bruli-lab/stowmark-console/internal/config"
	"github.com/bruli-lab/stowmark-console/internal/infra/console"
	"github.com/bruli-lab/stowmark-console/internal/infra/stderr"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	conf, err := config.New()
	if err != nil {
		return err
	}

	redirection, err := stderr.RedirectTo(conf.LogFile())
	if err != nil {
		return err
	}

	model := console.New(context.Background(), conf.Repository())
	_, runErr := tea.NewProgram(&model, tea.WithAltScreen()).Run()

	if err := redirection.Restore(); err != nil {
		return errors.Join(runErr, err)
	}

	if runErr != nil {
		return fmt.Errorf("running console (logs at %s): %w", conf.LogFile(), runErr)
	}

	return nil
}

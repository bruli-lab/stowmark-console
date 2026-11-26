package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/bruli-lab/stowmark-console/internal/app"
	"github.com/bruli-lab/stowmark-console/internal/config"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/infra/console"
	"github.com/bruli-lab/stowmark-console/internal/infra/stderr"
	"github.com/bruli-lab/stowmark-console/internal/infra/store"
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

	redirection, err := stderr.RedirectTo(conf.LogFile)
	if err != nil {
		return err
	}

	runErr := runConsole(conf)

	if err := redirection.Restore(); err != nil {
		return errors.Join(runErr, err)
	}

	if runErr != nil {
		return fmt.Errorf("%w (logs at %s)", runErr, conf.LogFile)
	}

	return nil
}

func runConsole(conf *config.Config) (err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session, err := store.Open(ctx, conf.Repository, conf.PrivateKey)
	if err != nil {
		return fmt.Errorf("opening repository: %w", err)
	}
	defer func() { err = errors.Join(err, session.Close()) }()

	handlers := app.NewHandlers(store.NewRepositories(session), store.NewSnapshots(session))
	model := console.New(ctx, console.Settings{
		RepositoryPath: conf.Repository,
		Init:           repository.NewInitOptions(conf.Compression, conf.CompressionLevel, conf.PublicKey),
	}, handlers)

	if _, err := tea.NewProgram(&model, tea.WithAltScreen()).Run(); err != nil {
		return fmt.Errorf("running console: %w", err)
	}

	return nil
}

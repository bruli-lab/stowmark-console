package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/bruli-lab/stowmark-console/internal/app"
	"github.com/bruli-lab/stowmark-console/internal/config"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/infra/console"
	"github.com/bruli-lab/stowmark-console/internal/infra/stderr"
	"github.com/bruli-lab/stowmark-console/internal/infra/store"
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

	session := store.NewSession()
	defer func() { err = errors.Join(err, session.Close()) }()

	catalog := store.NewCatalog(session, catalogEntries(conf.Repositories))
	handlers := app.NewHandlers(catalog, store.NewRepositories(session), store.NewSnapshots(session))
	model := console.New(ctx, &handlers)

	if _, err := tea.NewProgram(&model).Run(); err != nil {
		return fmt.Errorf("running console: %w", err)
	}

	return nil
}

func catalogEntries(repositories []config.Repository) []store.Entry {
	entries := make([]store.Entry, len(repositories))

	for i := range repositories {
		repo := &repositories[i]
		entries[i] = store.Entry{
			Profile: repository.NewProfile(
				repo.Name,
				repo.URL,
				repository.NewInitOptions(repo.Compression, repo.CompressionLevel, repo.PublicKey),
			),
			PrivateKey:  repo.PrivateKey,
			Environment: repo.Env,
		}
	}

	return entries
}

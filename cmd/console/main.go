package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/bruli-lab/stowmark-console/internal/app"
	"github.com/bruli-lab/stowmark-console/internal/buildinfo"
	"github.com/bruli-lab/stowmark-console/internal/config"
	"github.com/bruli-lab/stowmark-console/internal/domain/repository"
	"github.com/bruli-lab/stowmark-console/internal/domain/retention"
	"github.com/bruli-lab/stowmark-console/internal/infra/console"
	"github.com/bruli-lab/stowmark-console/internal/infra/desktop"
	"github.com/bruli-lab/stowmark-console/internal/infra/stderr"
	"github.com/bruli-lab/stowmark-console/internal/infra/store"
)

const splashDuration = 1500 * time.Millisecond

var version string

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

	feed := console.NewProgressFeed()
	session := store.NewSession(store.WithProgress(feed))
	defer func() { err = errors.Join(err, session.Close()) }()

	catalog := store.NewCatalog(session, catalogEntries(conf.Repositories))
	buses := app.NewBuses(&app.Ports{
		Catalog:      catalog,
		Repositories: store.NewRepositories(session),
		Snapshots:    store.NewSnapshots(session),
		Keys:         store.NewKeys(session),
		Retention:    store.NewRetention(session),
		Checker:      store.NewChecker(session),
	})
	openDirectory, err := os.MkdirTemp("", "stowmark-console-open-*")
	if err != nil {
		return fmt.Errorf("creating the directory for opened files: %w", err)
	}
	defer func() { err = errors.Join(err, os.RemoveAll(openDirectory)) }()

	model := console.New(ctx, buses,
		console.WithFileOpener(openDirectory, desktop.Open),
		console.WithSplash(buildinfo.Version(version), splashDuration),
		console.WithProgressFeed(feed),
	)

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
				repository.NewInitOptions(repo.Compression, repo.CompressionLevel, repo.PublicKey).
					WithFormatVersion(repo.FormatVersion),
			).WithPrivateKey(repo.PrivateKey).WithRetention(retentionRules(repo.Retention)),
			Environment: repo.Env,
		}
	}

	return entries
}

func retentionRules(configured *config.Retention) *retention.Rules {
	if configured == nil {
		return nil
	}

	return &retention.Rules{
		Last:    configured.KeepLast,
		Daily:   configured.KeepDaily,
		Weekly:  configured.KeepWeekly,
		Monthly: configured.KeepMonthly,
		Yearly:  configured.KeepYearly,
		Within:  configured.Within,
	}
}

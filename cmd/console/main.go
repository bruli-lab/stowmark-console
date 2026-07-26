package main

import (
	"context"
	"fmt"
	"os"

	"github.com/bruli-lab/stowmark-console.git/internal/config"
	"github.com/bruli-lab/stowmark-console.git/internal/infra/console"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	conf, err := config.New()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	model := console.New(context.Background(), conf)

	program := tea.NewProgram(
		&model,
		tea.WithoutCatchPanics(),
	)

	if _, err := program.Run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error executant la consola: %v\n", err)
		os.Exit(1)
	}
}

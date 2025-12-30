package main

import (
	"fmt"
	"os"

	"git.15b.it/eno/critic/internal/app"
	"git.15b.it/eno/critic/internal/cli"
	"git.15b.it/eno/critic/internal/git"
	"git.15b.it/eno/critic/internal/logger"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Initialize logger
	logger.Init()
	logger.Info("=== Critic starting ===")

	// Check if we're in a git repository
	if !git.IsGitRepo() {
		fmt.Fprintln(os.Stderr, "Error: Not a git repository")
		logger.Error("Not a git repository")
		os.Exit(1)
	}

	// Parse command-line arguments
	args, err := cli.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Create and run the application
	m := app.NewModel(args)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

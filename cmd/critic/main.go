package main

import (
	"flag"
	"fmt"
	"os"

	"git.15b.it/eno/critic/internal/app"
	"git.15b.it/eno/critic/internal/git"
	"git.15b.it/eno/critic/internal/logger"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Initialize logger
	if err := logger.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize logger: %v\n", err)
	}
	logger.Info("=== Critic starting ===")

	// Parse flags
	noHighlight := flag.Bool("no-highlight", false, "Disable syntax highlighting")
	flag.Parse()

	// Check if we're in a git repository
	if !git.IsGitRepo() {
		fmt.Fprintln(os.Stderr, "Error: Not a git repository")
		logger.Error("Not a git repository")
		os.Exit(1)
	}

	// Get remaining arguments as paths
	paths := flag.Args()

	// Create and run the application
	m := app.NewModel(paths, !*noHighlight)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

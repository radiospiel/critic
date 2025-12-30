package main

import (
	"os"

	"git.15b.it/eno/critic/internal/app"
	"git.15b.it/eno/critic/internal/cli"
	"git.15b.it/eno/critic/internal/git"
	"git.15b.it/eno/critic/internal/logger"
	"git.15b.it/eno/critic/internal/preconditions"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	logger.Info("=== Critic starting ===")

	// Check if we're in a git repository
	preconditions.Check(git.IsGitRepo(), "Not a git repository")

	// Parse command-line arguments
	args, err := cli.Parse(os.Args[1:])
	preconditions.Check(err == nil, "Failed to parse arguments: %v", err)

	// Create and run the application
	m := app.NewModel(args)
	p := tea.NewProgram(m, tea.WithAltScreen())

	_, err = p.Run()
	preconditions.Check(err == nil, "Application error: %v", err)
}

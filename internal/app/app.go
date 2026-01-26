package app

import (
	"fmt"

	"git.15b.it/eno/critic/internal/config"
	"git.15b.it/eno/critic/internal/git"
	"git.15b.it/eno/critic/simple-go/logger"
	"git.15b.it/eno/critic/simple-go/must"
	"git.15b.it/eno/critic/teapot"
	"github.com/samber/lo"
)

// Args represents parsed command-line arguments
type Args struct {
	Bases      []string // List of base points (e.g., ["main", "origin/main", "HEAD"])
	Paths      []string // Paths to diff
	Extensions []string // File extensions to include
	Debug      bool     // Enable debug mode
}

// GetDefaultBases returns the default base points based on git state
func GetDefaultBases() []string {
	candidates := []string{
		"main", "master", "origin/" + must.Must2(git.GetCurrentBranch()), "HEAD",
	}

	return lo.Filter(candidates, func(ref string, _ int) bool {
		return git.HasRef(ref)
	})
}

// Run runs the application with the given arguments
func Run(args *Args) error {
	logger.Info("=== Critic starting ===")

	// Check if we're in a git repository
	if !git.IsGitRepo() {
		return fmt.Errorf("not a git repository")
	}

	// Set default bases if none were specified
	if len(args.Bases) == 0 {
		args.Bases = GetDefaultBases()
	}

	// Set default extensions if none were specified
	if len(args.Extensions) == 0 {
		args.Extensions = config.DefaultFileExtensions
	}

	// Create delegate (critic-specific logic)
	delegate := NewDelegate(args)

	// Create and run the application using teapot.App
	app := teapot.NewApp(delegate.mainLayout, delegate)
	delegate.app = app // Give delegate access to app for focus manager

	return app.Run()
}

package app

import (
	"git.15b.it/eno/critic/internal/git"
	"git.15b.it/eno/critic/simple-go/must"
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

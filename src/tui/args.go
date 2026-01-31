package tui

// Args represents parsed command-line arguments
type Args struct {
	Bases      []string // List of base points (e.g., ["main", "origin/main", "HEAD"])
	Paths      []string // Paths to diff
	Extensions []string // File extensions to include
	Debug      bool     // Enable debug mode
}

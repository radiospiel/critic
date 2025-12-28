package cli

import (
	"fmt"
	"strings"

	"git.15b.it/eno/critic/internal/config"
	"git.15b.it/eno/critic/internal/git"
)

// Args represents parsed command-line arguments
type Args struct {
	Bases      []string // List of base points (e.g., ["merge-base", "origin/main", "HEAD"])
	Current    string   // Current target (e.g., "current" or a git ref)
	Paths      []string // Paths to diff
	Extensions []string // File extensions to include
}

// Parse parses command-line arguments into structured Args
// Supports: critic --extensions=c,rb,go base1,base2,base3..current -- path1 path2 path3
func Parse(args []string) (*Args, error) {
	result := &Args{
		Extensions: config.DefaultFileExtensions,
		Paths:      []string{"."},
		Current:    "current", // Default to working directory
	}

	// Separate args into flags and positional args
	var positional []string
	var pathArgs []string
	foundSeparator := false

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if foundSeparator {
			// After --, everything is a path
			pathArgs = append(pathArgs, arg)
			continue
		}

		if arg == "--" {
			foundSeparator = true
			continue
		}

		if strings.HasPrefix(arg, "--extensions=") {
			// Parse extensions flag
			exts := strings.TrimPrefix(arg, "--extensions=")
			if exts != "" {
				result.Extensions = strings.Split(exts, ",")
			}
			continue
		}

		if strings.HasPrefix(arg, "--") {
			return nil, fmt.Errorf("unknown flag: %s", arg)
		}

		// Positional argument
		positional = append(positional, arg)
	}

	// If paths were explicitly provided, use them
	if len(pathArgs) > 0 {
		result.Paths = pathArgs
	}

	// Parse base..current syntax from positional args
	if len(positional) > 0 {
		if err := parseBasesCurrent(positional[0], result); err != nil {
			return nil, err
		}
	}

	// Set default bases if none were specified
	if len(result.Bases) == 0 {
		bases, err := getDefaultBases()
		if err != nil {
			return nil, fmt.Errorf("failed to determine default bases: %w", err)
		}
		result.Bases = bases
	}

	return result, nil
}

// parseBasesCurrent parses the "base1,base2,base3..current" syntax
func parseBasesCurrent(arg string, result *Args) error {
	parts := strings.Split(arg, "..")

	if len(parts) > 2 {
		return fmt.Errorf("invalid base..current syntax: too many '..' separators")
	}

	// Parse bases (before ..)
	if parts[0] != "" {
		result.Bases = strings.Split(parts[0], ",")
	}

	// Parse current (after ..)
	if len(parts) == 2 && parts[1] != "" {
		result.Current = parts[1]
	}

	return nil
}

// getDefaultBases returns the default base points based on git state
func getDefaultBases() ([]string, error) {
	bases := []string{}

	// 1. Add merge base against main/master (if exists)
	mergeBase, err := git.GetMergeBase()
	if err == nil && mergeBase != "" {
		bases = append(bases, "merge-base")
	}

	// 2. Add origin/<current-branch> if it exists
	branch, err := git.GetCurrentBranch()
	if err == nil && branch != "" {
		originBranch := "origin/" + branch
		// Check if origin branch exists
		if branchExists(originBranch) {
			bases = append(bases, originBranch)
		}
	}

	// 3. Add HEAD (last committed version)
	bases = append(bases, "HEAD")

	return bases, nil
}

// branchExists checks if a git ref exists
func branchExists(ref string) bool {
	// Try to resolve the ref
	_, err := git.ResolveRef(ref)
	return err == nil
}

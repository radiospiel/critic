package cli

import (
	"fmt"
	"strings"

	"git.15b.it/eno/critic/internal/config"
	"git.15b.it/eno/critic/internal/git"
	"github.com/spf13/cobra"
)

// Args represents parsed command-line arguments
type Args struct {
	Bases      []string // List of base points (e.g., ["main", "origin/main", "HEAD"])
	Current    string   // Current target (e.g., "current" or a git ref)
	Paths      []string // Paths to diff
	Extensions []string // File extensions to include
}

// Parse parses command-line arguments into structured Args
// Supports: critic --extensions=c,rb,go base1,base2,base3..current -- path1 path2 path3
// Now implemented using Cobra
func Parse(args []string) (*Args, error) {
	var result *Args
	var parseErr error
	executed := false

	// Create root command with a custom RunE that captures args
	rootCmd := newRootCmd(func(args *Args, err error) {
		result = args
		parseErr = err
		executed = true
	})

	// Set args
	rootCmd.SetArgs(args)

	// Execute
	if err := rootCmd.Execute(); err != nil {
		return nil, err
	}

	if parseErr != nil {
		return nil, parseErr
	}

	// If command wasn't executed (e.g., --help), result will be nil
	if !executed {
		// Help or version was displayed, exit gracefully
		// Cobra already printed the output, so we just need to exit
		return nil, fmt.Errorf("help displayed")
	}

	return result, nil
}

// newRootCmd creates the root cobra command
func newRootCmd(callback func(*Args, error)) *cobra.Command {
	var extensionsFlag []string

	cmd := &cobra.Command{
		Use:   "critic [flags] [base1,base2..current] [-- path1 path2 ...]",
		Short: "Critic - Git diff viewer with side-by-side comparison",
		Long: `Critic is a terminal-based git diff viewer that shows changes side-by-side.

Syntax:
  critic [base1,base2,base3..current] [-- path1 path2 path3]

Examples:
  critic                           # Compare against default bases (main/master, origin/<branch>, HEAD)
  critic main..current             # Compare main branch to working directory
  critic main,develop..current     # Compare against multiple bases
  critic HEAD~1..HEAD              # Compare two commits
  critic -- src tests              # Only show changes in src and tests directories
  critic --extensions=go,rs        # Only show .go and .rs files
`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args: func(cmd *cobra.Command, args []string) error {
			// Custom validator: allow at most 1 arg before --, any number after --
			argsLenAtDash := cmd.ArgsLenAtDash()
			if argsLenAtDash >= 0 {
				// There was a -- separator
				// args before -- should be at most 1
				if argsLenAtDash > 1 {
					return fmt.Errorf("accepts at most 1 arg before --, received %d", argsLenAtDash)
				}
				return nil
			}
			// No -- separator, so all args are positional
			if len(args) > 1 {
				return fmt.Errorf("accepts at most 1 arg, received %d", len(args))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			result := &Args{
				Extensions: extensionsFlag,
				Paths:      []string{"."},
				Current:    "current", // Default to working directory
			}

			// Get paths after -- separator
			argsLenAtDash := cmd.ArgsLenAtDash()
			var baseArg string
			if argsLenAtDash >= 0 {
				// There was a -- separator
				// The base..current arg is before --
				if argsLenAtDash > 0 {
					baseArg = args[0]
				}
				// Paths are after --
				pathArgs := args[argsLenAtDash:]
				if len(pathArgs) > 0 {
					result.Paths = pathArgs
				}
			} else {
				// No -- separator
				if len(args) > 0 {
					baseArg = args[0]
				}
			}

			// Parse base..current syntax
			if baseArg != "" {
				if err := parseBasesCurrent(baseArg, result); err != nil {
					callback(nil, err)
					return err
				}
			}

			// Set default bases if none were specified
			if len(result.Bases) == 0 {
				bases, err := getDefaultBases()
				if err != nil {
					callback(nil, fmt.Errorf("failed to determine default bases: %w", err))
					return err
				}
				result.Bases = bases
			}

			callback(result, nil)
			return nil
		},
	}

	// Define flags
	cmd.Flags().StringSliceVar(&extensionsFlag, "extensions", config.DefaultFileExtensions, "Comma-separated list of file extensions to include")

	return cmd
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

	// 1. Add main/master if it exists (will use merge-base automatically)
	if branchExists("main") {
		bases = append(bases, "main")
	} else if branchExists("master") {
		bases = append(bases, "master")
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

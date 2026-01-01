package cli

import (
	"fmt"
	"strings"

	"git.15b.it/eno/critic/internal/app"
	"git.15b.it/eno/critic/internal/config"
	"git.15b.it/eno/critic/internal/git"
	"git.15b.it/eno/critic/internal/logger"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// Execute runs the CLI application
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		// Cobra already printed the error
		// Just exit with error code
		logger.Error("Command failed: %v", err)
	}
}

// ParseArgsForTesting parses command-line arguments without running the app
// This is exported for testing purposes only
func ParseArgsForTesting(args []string) (*app.Args, error) {
	var result *app.Args
	var parseErr error

	cmd := newTestCmd(func(a *app.Args, err error) {
		result = a
		parseErr = err
	})

	cmd.SetArgs(args)

	if err := cmd.Execute(); err != nil {
		return nil, err
	}

	if parseErr != nil {
		return nil, parseErr
	}

	return result, nil
}

// newTestCmd creates a command for testing that doesn't run the app
func newTestCmd(callback func(*app.Args, error)) *cobra.Command {
	var extensionsFlag []string

	cmd := &cobra.Command{
		Use:           "critic [flags] [base1,base2..current] [-- path1 path2 ...]",
		Short:         "Critic - Git diff viewer",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args: func(cmd *cobra.Command, args []string) error {
			argsLenAtDash := cmd.ArgsLenAtDash()
			if argsLenAtDash >= 0 {
				if argsLenAtDash > 1 {
					return fmt.Errorf("accepts at most 1 arg before --, received %d", argsLenAtDash)
				}
				return nil
			}
			if len(args) > 1 {
				return fmt.Errorf("accepts at most 1 arg, received %d", len(args))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			parsedArgs := &app.Args{
				Extensions: extensionsFlag,
				Paths:      []string{"."},
				Current:    "current",
			}

			argsLenAtDash := cmd.ArgsLenAtDash()
			var baseArg string
			if argsLenAtDash >= 0 {
				if argsLenAtDash > 0 {
					baseArg = args[0]
				}
				pathArgs := args[argsLenAtDash:]
				if len(pathArgs) > 0 {
					parsedArgs.Paths = pathArgs
				}
			} else {
				if len(args) > 0 {
					baseArg = args[0]
				}
			}

			if baseArg != "" {
				if err := parseBasesCurrent(baseArg, parsedArgs); err != nil {
					callback(nil, err)
					return err
				}
			}

			if len(parsedArgs.Bases) == 0 {
				bases, err := getDefaultBases()
				if err != nil {
					callback(nil, fmt.Errorf("failed to determine default bases: %w", err))
					return err
				}
				parsedArgs.Bases = bases
			}

			callback(parsedArgs, nil)
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&extensionsFlag, "extensions", config.DefaultFileExtensions, "Comma-separated list of file extensions to include")

	return cmd
}

// NewRootCmd creates the root cobra command
func NewRootCmd() *cobra.Command {
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
		PreRunE: func(cmd *cobra.Command, args []string) error {
			logger.Info("=== Critic starting ===")

			// Check if we're in a git repository
			if !git.IsGitRepo() {
				return fmt.Errorf("not a git repository")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse arguments
			parsedArgs := &app.Args{
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
					parsedArgs.Paths = pathArgs
				}
			} else {
				// No -- separator
				if len(args) > 0 {
					baseArg = args[0]
				}
			}

			// Parse base..current syntax
			if baseArg != "" {
				if err := parseBasesCurrent(baseArg, parsedArgs); err != nil {
					return err
				}
			}

			// Set default bases if none were specified
			if len(parsedArgs.Bases) == 0 {
				bases, err := getDefaultBases()
				if err != nil {
					return fmt.Errorf("failed to determine default bases: %w", err)
				}
				parsedArgs.Bases = bases
			}

			// Create and run the application
			m := app.NewModel(parsedArgs)
			p := tea.NewProgram(m, tea.WithAltScreen())

			if _, err := p.Run(); err != nil {
				return fmt.Errorf("application error: %w", err)
			}

			return nil
		},
	}

	// Define flags
	cmd.Flags().StringSliceVar(&extensionsFlag, "extensions", config.DefaultFileExtensions, "Comma-separated list of file extensions to include")

	return cmd
}

// parseBasesCurrent parses the "base1,base2,base3..current" syntax
func parseBasesCurrent(arg string, result *app.Args) error {
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

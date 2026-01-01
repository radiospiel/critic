package cli

import (
	"fmt"
	"strings"

	"git.15b.it/eno/critic/internal/app"
	"git.15b.it/eno/critic/internal/config"
	"github.com/spf13/cobra"
)

var commandHandler func(*app.Args) error

// OnCommand sets the callback to run when the command is executed
func OnCommand(handler func(*app.Args) error) {
	commandHandler = handler
}

// Execute runs the CLI application
func Execute() error {
	return NewRootCmd().Execute()
}

// ParseArgsForTesting parses command-line arguments without running the app
// This is exported for testing purposes only
func ParseArgsForTesting(args []string) (*app.Args, error) {
	var result *app.Args
	var parseErr error

	// Save the original handler
	originalHandler := commandHandler

	// Set a test handler that captures the args
	OnCommand(func(a *app.Args) error {
		result = a
		return nil
	})

	// Restore the original handler when done
	defer func() {
		commandHandler = originalHandler
	}()

	// Create command and execute
	cmd := NewRootCmd()
	cmd.SetArgs(args)

	if err := cmd.Execute(); err != nil {
		return nil, err
	}

	if parseErr != nil {
		return nil, parseErr
	}

	return result, nil
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

			// Call the command handler (set via OnCommand)
			if commandHandler != nil {
				return commandHandler(parsedArgs)
			}

			return fmt.Errorf("no command handler set")
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


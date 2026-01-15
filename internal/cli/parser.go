package cli

import (
	"fmt"
	"strings"

	"git.15b.it/eno/critic/internal/app"
	"github.com/spf13/cobra"
)

// Execute runs the CLI application with the given handler
func Execute(handler func(*app.Args) error) error {
	rootCmd := newRootCmd(handler)

	// Add subcommands
	rootCmd.AddCommand(newMCPCmd())
	rootCmd.AddCommand(newConvoCmd())
	rootCmd.AddCommand(newLogCmd())

	return rootCmd.Execute()
}

// ParseArgsForTesting parses command-line arguments without running the app
// This is exported for testing purposes only
func ParseArgsForTesting(args []string) (*app.Args, error) {
	var result *app.Args

	// Create command with a test handler that captures the args
	cmd := newRootCmd(func(a *app.Args) error {
		result = a
		return nil
	})

	cmd.SetArgs(args)

	if err := cmd.Execute(); err != nil {
		return nil, err
	}

	return result, nil
}

// newRootCmd creates the root cobra command with the given handler
func newRootCmd(handler func(*app.Args) error) *cobra.Command {
	var extensionsFlag []string
	var noAnimationFlag bool

	cmd := &cobra.Command{
		Use:   "critic [flags] [base1,base2,...] [-- path1 path2 ...]",
		Short: "Critic - Git diff viewer with side-by-side comparison",
		Long: `Critic is a terminal-based git diff viewer that shows changes side-by-side.

Syntax:
  critic [base1,base2,base3] [-- path1 path2 path3]

Examples:
  critic                           # Compare against default bases (main/master, origin/<branch>, HEAD)
  critic main                      # Compare main branch to HEAD
  critic main,develop              # Compare against multiple bases
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
				Extensions:  ensureSlice(extensionsFlag),
				Paths:       []string{"."},
				NoAnimation: noAnimationFlag,
			}

			// Get paths after -- separator
			argsLenAtDash := cmd.ArgsLenAtDash()
			var baseArg string
			if argsLenAtDash >= 0 {
				// There was a -- separator
				// The bases arg is before --
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

			// Parse bases
			if baseArg != "" {
				parsedArgs.Bases = strings.Split(baseArg, ",")
			}

			// Call the command handler
			return handler(parsedArgs)
		},
	}

	// Define flags
	cmd.Flags().StringSliceVar(&extensionsFlag, "extensions", nil, "Comma-separated list of file extensions to include")
	cmd.Flags().BoolVar(&noAnimationFlag, "no-animation", false, "Disable animations")

	return cmd
}

// ensureSlice converts nil to an empty slice
func ensureSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

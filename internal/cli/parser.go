package cli

import (
	"git.15b.it/eno/critic/internal/app"
	"github.com/spf13/cobra"
)

// Execute runs the CLI application with the given handler
func Execute(handler func(*app.Args) error) error {
	rootCmd := newRootCmd()

	// Add subcommands
	rootCmd.AddCommand(newTUICmd(handler))
	rootCmd.AddCommand(newWebUICmd())
	rootCmd.AddCommand(newMCPCmd())
	rootCmd.AddCommand(newConvoCmd())
	rootCmd.AddCommand(newLogCmd())

	return rootCmd.Execute()
}

// ParseArgsForTesting parses command-line arguments without running the app
// This is exported for testing purposes only
func ParseArgsForTesting(args []string) (*app.Args, error) {
	var result *app.Args

	// Create tui command with a test handler that captures the args
	tuiCmd := newTUICmd(func(a *app.Args) error {
		result = a
		return nil
	})

	// Prepend "tui" to args for testing
	tuiCmd.SetArgs(args)

	if err := tuiCmd.Execute(); err != nil {
		return nil, err
	}

	return result, nil
}

// newRootCmd creates the root cobra command
func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "critic",
		Short: "Critic - Git diff viewer and code review tool",
		Long: `Critic is a git diff viewer and code review tool.

Available commands:
  tui     Start the terminal user interface
  webui   Start the web user interface
  mcp     Start the MCP server
  convo   Manage conversations

Examples:
  critic tui                    # Start TUI with default bases
  critic tui main               # Compare against main branch
  critic webui                  # Start web interface on localhost:8080
  critic webui --port=3000      # Start web interface on port 3000
`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	return cmd
}

// ensureSlice converts nil to an empty slice
func ensureSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

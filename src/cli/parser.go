package cli

import (
	"github.com/spf13/cobra"
)

// Execute runs the CLI application with the given handler
func Execute() error {
	rootCmd := newRootCmd()

	// Add subcommands
	rootCmd.AddCommand(newHTTPDCmd())
	rootCmd.AddCommand(newMCPCmd())
	rootCmd.AddCommand(newConvoCmd())
	rootCmd.AddCommand(newTestCmd())
	rootCmd.AddCommand(newREPLCmd())
	rootCmd.AddCommand(newAgentCmd())

	return rootCmd.Execute()
}

// newRootCmd creates the root cobra command
func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "critic",
		Short: "Critic - Git diff viewer and code review tool",
		Long: `Critic is a git diff viewer and code review tool.

Examples:
  critic httpd                    # Start API server on localhost:65432
  critic httpd --port=8000        # Start API server on custom port
`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().String("project", "", "Path to project.critic config file (default: auto-detect in git root)")

	return cmd
}

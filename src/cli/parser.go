package cli

import (
	"github.com/radiospiel/critic/src/git"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

// getDefaultBases returns the default base points based on git state
func getDefaultBases() []string {
	candidates := []string{
		"main", "master", "origin/" + git.GetCurrentBranch(), "HEAD",
	}

	return lo.Filter(candidates, func(ref string, _ int) bool {
		return git.HasRef(ref)
	})
}

// Execute runs the CLI application with the given handler
func Execute() error {
	rootCmd := newRootCmd()

	// Add subcommands
	rootCmd.AddCommand(newAPICmd())
	rootCmd.AddCommand(newMCPCmd())
	rootCmd.AddCommand(newConvoCmd())

	return rootCmd.Execute()
}

// newRootCmd creates the root cobra command
func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "critic",
		Short: "Critic - Git diff viewer and code review tool",
		Long: `Critic is a git diff viewer and code review tool.

Available commands:
  api     Start the HTTP/connect API server (includes web UI)
  mcp     Start the MCP server
  convo   Manage conversations

Examples:
  critic api                    # Start API server on localhost:65432
  critic api --port=8000        # Start API server on custom port
`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	return cmd
}

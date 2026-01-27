package cli

import (
	"github.com/spf13/cobra"
	"github.org/radiospiel/critic/simple-go/must"
	"github.com/samber/lo"
	"github.org/radiospiel/critic/src/git"
)

// getDefaultBases returns the default base points based on git state
func getDefaultBases() []string {
	candidates := []string{
		"main", "master", "origin/" + must.Must2(git.GetCurrentBranch()), "HEAD",
	}

	return lo.Filter(candidates, func(ref string, _ int) bool {
		return git.HasRef(ref)
	})
}

// Execute runs the CLI application with the given handler
func Execute() error {
	rootCmd := newRootCmd()

	// Add subcommands
	rootCmd.AddCommand(newTUICmd())
	rootCmd.AddCommand(newWebUICmd())
	rootCmd.AddCommand(newMCPCmd())
	rootCmd.AddCommand(newConvoCmd())
	rootCmd.AddCommand(newLogCmd())

	return rootCmd.Execute()
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

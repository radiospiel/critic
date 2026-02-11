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

// mergeDefaultBases merges explicitly added bases with the defaults (master, main, HEAD),
// ensuring all valid refs are included without duplicates.
func mergeDefaultBases(explicit []string) []string {
	defaults := getDefaultBases()

	// Start with defaults, then append explicit ones that aren't already present
	seen := make(map[string]bool, len(defaults)+len(explicit))
	merged := make([]string, 0, len(defaults)+len(explicit))

	for _, ref := range defaults {
		if !seen[ref] {
			seen[ref] = true
			merged = append(merged, ref)
		}
	}
	for _, ref := range explicit {
		if !seen[ref] && git.HasRef(ref) {
			seen[ref] = true
			merged = append(merged, ref)
		}
	}

	return merged
}

// Execute runs the CLI application with the given handler
func Execute() error {
	rootCmd := newRootCmd()

	// Add subcommands
	rootCmd.AddCommand(newHTTPDCmd())
	rootCmd.AddCommand(newMCPCmd())
	rootCmd.AddCommand(newConvoCmd())
	rootCmd.AddCommand(newTestCmd())

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

	return cmd
}

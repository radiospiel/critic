package cli

import (
	"github.com/radiospiel/critic/src/mcp"
	"github.com/spf13/cobra"
)

// newMCPCmd creates the mcp subcommand
func newMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Run the HITL MCP server for Claude Code integration",
		Long: `Run the Human-in-the-Loop MCP server that enables Claude Code to request
feedback from a human reviewer during code changes.

The server communicates with Claude Code via stdio (JSON-RPC) and retrieves
feedback from the message database.

Tools provided:
  - get_review_feedback: Retrieve unresolved feedback from the database

Example Claude Code config (.claude/settings.json):
  {
    "mcpServers": {
      "critic": {
        "command": "critic",
        "args": ["mcp"]
      }
    }
  }

Reviewers can add comments using the TUI:
  critic diff
`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			server := mcp.NewServer()
			return server.Run()
		},
	}

	return cmd
}

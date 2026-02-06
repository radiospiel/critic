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

	- get_critic_conversations: Get a list of conversation UUIDs. Optionally filter by status ('unresolved' or 'resolved'). Use this to check for reviewer feedback.
	- get_full_critic_conversation: Get the complete conversation including all messages and replies. Returns conversation metadata and all messages ordered chronologically.
	- reply_to_critic_conversation: Add a reply to an existing conversation. Use this to respond to reviewer feedback.
	- critic_announce: Post an announcement visible in the Critic UI. Creates a message on the root conversation and marks it as unresolved.

Claude Code registration via "claude mcp add critic -- /path/to/critic mcp"
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

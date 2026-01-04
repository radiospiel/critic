package cli

import (
	"time"

	"git.15b.it/eno/critic/internal/mcp"
	"github.com/spf13/cobra"
)

// MCPArgs holds the arguments for the MCP server command
type MCPArgs struct {
	SocketPath string
	Timeout    time.Duration
}

// newMCPCmd creates the mcp subcommand
func newMCPCmd() *cobra.Command {
	args := &MCPArgs{}

	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Run the HITL MCP server for Claude Code integration",
		Long: `Run the Human-in-the-Loop MCP server that enables Claude Code to request
feedback from a human reviewer during code changes.

The server communicates with Claude Code via stdio (JSON-RPC) and with the
reviewer via a Unix socket at the specified path.

Tools provided:
  - get_review_feedback: Wait for and retrieve feedback from the reviewer
  - notify_reviewer: Send a notification without waiting for response

Example Claude Code config (.claude/settings.json):
  {
    "mcpServers": {
      "critic": {
        "command": "critic",
        "args": ["mcp"]
      }
    }
  }

To send feedback as a reviewer:
  echo "Looks good, approved!" | nc -U /tmp/critic-hitl.sock

Or use the review subcommand:
  critic review "Your feedback here"
  critic review --approve "LGTM"
  critic review --reject "Please add error handling"
`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			server := mcp.NewServer(args.SocketPath)
			if args.Timeout > 0 {
				server.SetFeedbackTimeout(args.Timeout)
			}
			return server.Run()
		},
	}

	cmd.Flags().StringVarP(&args.SocketPath, "socket", "s", mcp.DefaultSocketPath, "Unix socket path for reviewer communication")
	cmd.Flags().DurationVarP(&args.Timeout, "timeout", "t", mcp.DefaultFeedbackTimeout, "Timeout for waiting for reviewer feedback")

	return cmd
}

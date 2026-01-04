package cli

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"git.15b.it/eno/critic/internal/mcp"
	"github.com/spf13/cobra"
)

// ReviewArgs holds the arguments for the review command
type ReviewArgs struct {
	SocketPath string
	Approve    bool
	Reject     bool
}

// newReviewCmd creates the review subcommand
func newReviewCmd() *cobra.Command {
	args := &ReviewArgs{}

	cmd := &cobra.Command{
		Use:   "review [feedback message]",
		Short: "Send feedback to the HITL MCP server",
		Long: `Send feedback to a running HITL MCP server. This command is used by the
human reviewer to provide feedback on Claude Code's work.

The feedback is sent via Unix socket to the MCP server, which will then
return it to Claude Code as a tool result.

Examples:
  # Send plain feedback
  critic review "Looks good, but add error handling to the API call"

  # Approve the changes
  critic review --approve "LGTM, ship it!"

  # Reject the changes
  critic review --reject "Please refactor this to use the existing helper"

  # Interactive mode (reads from stdin)
  echo "Your feedback" | critic review
`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			var feedback string

			if len(cmdArgs) > 0 {
				feedback = strings.Join(cmdArgs, " ")
			} else {
				// Read from stdin if no args provided
				var lines []string
				scanner := cmd.InOrStdin()
				buf := make([]byte, 4096)
				for {
					n, err := scanner.Read(buf)
					if n > 0 {
						lines = append(lines, string(buf[:n]))
					}
					if err != nil {
						break
					}
				}
				feedback = strings.TrimSpace(strings.Join(lines, ""))
			}

			if feedback == "" {
				return fmt.Errorf("feedback message is required")
			}

			// Determine message type
			msgType := "feedback"
			if args.Approve {
				msgType = "approved"
			} else if args.Reject {
				msgType = "rejected"
			}

			// Create the message
			msg := mcp.ReviewerMessage{
				Type:     msgType,
				Feedback: feedback,
			}

			// Connect to the socket
			conn, err := net.Dial("unix", args.SocketPath)
			if err != nil {
				return fmt.Errorf("failed to connect to HITL server at %s: %w\n(Is the MCP server running?)", args.SocketPath, err)
			}
			defer conn.Close()

			// Send the message
			data, err := json.Marshal(msg)
			if err != nil {
				return fmt.Errorf("failed to marshal message: %w", err)
			}

			_, err = conn.Write(append(data, '\n'))
			if err != nil {
				return fmt.Errorf("failed to send message: %w", err)
			}

			fmt.Printf("Feedback sent: [%s] %s\n", msgType, feedback)
			return nil
		},
	}

	cmd.Flags().StringVarP(&args.SocketPath, "socket", "s", mcp.DefaultSocketPath, "Unix socket path for HITL server")
	cmd.Flags().BoolVarP(&args.Approve, "approve", "a", false, "Mark feedback as approval")
	cmd.Flags().BoolVarP(&args.Reject, "reject", "r", false, "Mark feedback as rejection")

	return cmd
}

package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/radiospiel/critic/simple-go/json"
	"github.com/radiospiel/critic/src/git"
	"github.com/radiospiel/critic/src/messagedb"
	"github.com/radiospiel/critic/src/pkg/critic"
	"github.com/spf13/cobra"
)

// newTestCmd creates the test parent command
func newTestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Testing utilities for critic",
		Long: `Commands for testing critic functionality.

Available subcommands:
  conversation   Create or reply to a conversation at file:line
`,
	}

	cmd.AddCommand(newTestConversationCmd())

	return cmd
}

// newTestConversationCmd creates the test conversation command
func newTestConversationCmd() *cobra.Command {
	var author string

	cmd := &cobra.Command{
		Use:   "conversation <file:line> <message>",
		Short: "Create or reply to a conversation at file:line",
		Long: `Create a conversation at the given file:line with text.
If a conversation already exists at that file:line, reply to it instead.

Examples:
  critic test conversation "src/main.go:42" "This looks wrong"
  critic test conversation --author ai "src/main.go:42" "I've fixed the issue"
`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			fileLine := args[0]
			message := args[1]

			// Parse "file:line" format
			lastColon := strings.LastIndex(fileLine, ":")
			if lastColon == -1 {
				return fmt.Errorf("invalid format: expected 'file:line', got '%s'", fileLine)
			}

			filePath := fileLine[:lastColon]
			lineStr := fileLine[lastColon+1:]

			lineNumber, err := strconv.Atoi(lineStr)
			if err != nil {
				return fmt.Errorf("invalid line number: %s", lineStr)
			}
			if lineNumber <= 0 {
				return fmt.Errorf("line number must be positive: %d", lineNumber)
			}

			// Validate author
			var msgAuthor critic.Author
			switch author {
			case "human":
				msgAuthor = critic.AuthorHuman
			case "ai":
				msgAuthor = critic.AuthorAI
			default:
				return fmt.Errorf("invalid author: %s (must be 'human' or 'ai')", author)
			}

			gitRoot := git.GetGitRoot()

			mdb, err := messagedb.New(gitRoot)
			if err != nil {
				return fmt.Errorf("failed to initialize message database: %w", err)
			}
			defer mdb.Close()

			// Check if a conversation already exists at this file:line
			conversations, err := mdb.GetConversationsByFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to get conversations for file: %w", err)
			}

			var existingConvID string
			for _, conv := range conversations {
				if conv.Lineno == lineNumber {
					existingConvID = conv.ID
					break
				}
			}

			if existingConvID != "" {
				// Reply to existing conversation
				reply, err := mdb.ReplyToConversation(existingConvID, message, msgAuthor)
				if err != nil {
					return fmt.Errorf("failed to create reply: %w", err)
				}

				response := ReplyResponse{
					UUID:      reply.UUID,
					Author:    string(reply.Author),
					Message:   reply.Message,
					CreatedAt: reply.CreatedAt.Format("2006-01-02 15:04:05"),
				}

				fmt.Println(json.ToPrettyJson(response))
			} else {
				// Create new conversation
				codeVersion := git.ResolveRef("HEAD")
				conversation, err := mdb.CreateConversation(
					msgAuthor,
					message,
					filePath,
					lineNumber,
					codeVersion,
					"", // no context for CLI-created conversations
					critic.TypeConversation,
				)
				if err != nil {
					return fmt.Errorf("failed to create conversation: %w", err)
				}

				// Build response
				messages := make([]MessageResponse, 0, len(conversation.Messages))
				for _, msg := range conversation.Messages {
					messages = append(messages, MessageResponse{
						UUID:      msg.UUID,
						Author:    string(msg.Author),
						Message:   msg.Message,
						CreatedAt: msg.CreatedAt.Format("2006-01-02 15:04:05"),
						UpdatedAt: msg.UpdatedAt.Format("2006-01-02 15:04:05"),
						IsUnread:  msg.IsUnread,
					})
				}

				response := ConversationResponse{
					UUID:        conversation.UUID,
					Status:      string(conversation.Status),
					FilePath:    conversation.FilePath,
					LineNumber:  conversation.LineNumber,
					CodeVersion: conversation.CodeVersion,
					Context:     conversation.Context,
					CreatedAt:   conversation.CreatedAt.Format("2006-01-02 15:04:05"),
					UpdatedAt:   conversation.UpdatedAt.Format("2006-01-02 15:04:05"),
					Messages:    messages,
				}

				fmt.Println(json.ToPrettyJson(response))
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&author, "author", "human", "Author of the message: 'human' or 'ai'")

	return cmd
}

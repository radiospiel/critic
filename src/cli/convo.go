package cli

import (
	"fmt"

	"github.com/radiospiel/critic/simple-go/json"
	"github.com/radiospiel/critic/src/git"
	"github.com/radiospiel/critic/src/messagedb"
	"github.com/radiospiel/critic/src/pkg/critic"
	"github.com/spf13/cobra"
)

// ConversationSummary represents a summary of a conversation for listing
type ConversationSummary struct {
	UUID           string `json:"uuid"`
	MessagePreview string `json:"message_preview"`
	Status         string `json:"status"`
	Author         string `json:"author"`
	FilePath       string `json:"file_path"`
	LineNumber     int    `json:"line_number"`
	Context        string `json:"context,omitempty"`
}

// ReplyResponse represents the response from creating a reply
type ReplyResponse struct {
	UUID      string `json:"uuid"`
	Author    string `json:"author"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}

// MessageResponse represents a message in a conversation for JSON output
type MessageResponse struct {
	UUID      string `json:"uuid"`
	Author    string `json:"author"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	IsUnread  bool   `json:"is_unread"`
}

// ConversationResponse represents a full conversation for JSON output
type ConversationResponse struct {
	UUID        string            `json:"uuid"`
	Status      string            `json:"status"`
	FilePath    string            `json:"file_path"`
	LineNumber  int               `json:"line_number"`
	CodeVersion string            `json:"code_version"`
	Context     string            `json:"context"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
	Messages    []MessageResponse `json:"messages"`
}

// newConvoCmd creates the convo parent command
func newConvoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "convo",
		Short: "Manage and view conversations",
		Long: `Commands for managing and viewing critic conversations.

Available subcommands:
  list      List conversations with details
  show      Show a complete conversation
  reply     Reply to a conversation
  announce  Post an announcement
`,
	}

	cmd.AddCommand(newConvoListCmd())
	cmd.AddCommand(newConvoShowCmd())
	cmd.AddCommand(newConvoReplyCmd())
	cmd.AddCommand(newConvoAnnounceCmd())

	return cmd
}

// newConvoListCmd creates the convo list command
func newConvoListCmd() *cobra.Command {
	var status string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List conversations with details",
		Long: `List all conversations with UUID, message preview, status, and author.

Examples:
  critic convo list                    # List all conversations
  critic convo list --status unresolved # List unresolved conversations
  critic convo list --status resolved   # List resolved conversations
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			gitRoot := git.GetGitRoot()

			mdb, err := messagedb.New(gitRoot)
			if err != nil {
				return fmt.Errorf("failed to initialize message database: %w", err)
			}
			defer mdb.Close()

			roots, err := mdb.GetConversations(status, nil)
			if err != nil {
				return fmt.Errorf("failed to get conversations: %w", err)
			}

			if len(roots) == 0 {
				fmt.Println("[]")
				return nil
			}

			// Batch-fetch full conversations
			uuids := make([]string, len(roots))
			for i, r := range roots {
				uuids[i] = r.UUID
			}
			conversations, err := mdb.GetFullConversations(uuids)
			if err != nil {
				return fmt.Errorf("failed to get full conversations: %w", err)
			}

			summaries := make([]ConversationSummary, 0, len(conversations))
			for _, conv := range conversations {
				if len(conv.Messages) == 0 {
					continue
				}

				firstMsg := conv.Messages[0]

				messagePreview := firstMsg.Message
				if len(messagePreview) > 100 {
					messagePreview = messagePreview[:100] + "..."
				}

				summaries = append(summaries, ConversationSummary{
					UUID:           conv.UUID,
					MessagePreview: messagePreview,
					Status:         string(conv.Status),
					Author:         string(firstMsg.Author),
					FilePath:       conv.FilePath,
					LineNumber:     conv.LineNumber,
					Context:        conv.Context,
				})
			}

			// Output as JSON array of objects
			fmt.Println(json.ToPrettyJson(summaries))
			return nil
		},
	}

	cmd.Flags().StringVar(&status, "status", "", "Filter by status: 'unresolved' or 'resolved'")

	return cmd
}

// newConvoShowCmd creates the convo show command
func newConvoShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <uuid>",
		Short: "Show a complete conversation",
		Long: `Display a complete conversation including all messages and replies as JSON.

Example:
  critic convo show a1b2c3d4-e5f6-7890-abcd-ef1234567890
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			uuid := args[0]

			gitRoot := git.GetGitRoot()

			mdb, err := messagedb.New(gitRoot)
			if err != nil {
				return fmt.Errorf("failed to initialize message database: %w", err)
			}
			defer mdb.Close()

			conversation, err := mdb.GetFullConversation(uuid)
			if err != nil {
				return fmt.Errorf("failed to get conversation: %w", err)
			}

			// Build message responses
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
			return nil
		},
	}

	return cmd
}

// newConvoReplyCmd creates the convo reply command
func newConvoReplyCmd() *cobra.Command {
	var author string

	cmd := &cobra.Command{
		Use:   "reply <uuid> <message>",
		Short: "Reply to a conversation",
		Long: `Add a reply to an existing conversation.

Examples:
  critic convo reply a1b2c3d4-e5f6-7890-abcd-ef1234567890 "This looks good"
  critic convo reply --author ai a1b2c3d4-e5f6-7890-abcd-ef1234567890 "I've fixed the issue"
`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			uuid := args[0]
			message := args[1]

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

			reply, err := mdb.ReplyToConversation(uuid, message, msgAuthor)
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
			return nil
		},
	}

	cmd.Flags().StringVar(&author, "author", "human", "Author of the reply: 'human' or 'ai'")

	return cmd
}

// newConvoAnnounceCmd creates the convo announce command
func newConvoAnnounceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "announce <message>",
		Short: "Post an announcement visible in the Critic UI",
		Long: `Post an announcement that appears as a banner in the Critic UI.
Creates a message on the root conversation and marks it as unresolved.

Examples:
  critic convo announce "Please review the auth changes before merging"
  critic convo announce "Build is broken, do not merge"
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			message := args[0]

			gitRoot := git.GetGitRoot()

			mdb, err := messagedb.New(gitRoot)
			if err != nil {
				return fmt.Errorf("failed to initialize message database: %w", err)
			}
			defer mdb.Close()

			// Get or create the root conversation
			rootConv, err := mdb.LoadRootConversation()
			if err != nil {
				return fmt.Errorf("failed to load root conversation: %w", err)
			}

			// Add the announcement as a reply
			reply, err := mdb.ReplyToConversation(rootConv.UUID, message, critic.AuthorAI)
			if err != nil {
				return fmt.Errorf("failed to create announcement: %w", err)
			}

			// Mark as unresolved so it shows in the UI
			if err := mdb.MarkConversationAs(rootConv.UUID, critic.ConversationUnresolved); err != nil {
				return fmt.Errorf("failed to mark announcement as unresolved: %w", err)
			}

			response := ReplyResponse{
				UUID:      reply.UUID,
				Author:    string(reply.Author),
				Message:   reply.Message,
				CreatedAt: reply.CreatedAt.Format("2006-01-02 15:04:05"),
			}

			fmt.Println(json.ToPrettyJson(response))
			return nil
		},
	}

	return cmd
}

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

// AgentConversationEntry represents a conversation in the agent conversations list output
type AgentConversationEntry struct {
	UUID       string `json:"uuid"`
	Author     string `json:"author"`
	Status     string `json:"status"`
}

// newAgentCmd creates the agent parent command
func newAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Commands for AI agent interaction",
		Long: `Commands designed for programmatic AI agent interaction with critic.

Available subcommands:
  conversations   List conversations with optional filters
  conversation    Show a complete conversation
  reply           Reply to a conversation
  announce        Post an announcement
  explain         Post an explanation on a code line
`,
	}

	cmd.AddCommand(newAgentConversationsCmd())
	cmd.AddCommand(newAgentConversationCmd())
	cmd.AddCommand(newAgentReplyCmd())
	cmd.AddCommand(newAgentAnnounceCmd())
	cmd.AddCommand(newAgentExplainCmd())

	return cmd
}

// newAgentConversationsCmd creates the agent conversations command
func newAgentConversationsCmd() *cobra.Command {
	var statusFilter string
	var lastAuthorFilter string

	cmd := &cobra.Command{
		Use:   "conversations",
		Short: "List conversations with details",
		Long: `List conversations with UUID, last message author, and status.

Supports comma-separated multi-value filters:
  --status=unresolved,resolved
  --status=actionable          (unresolved with last message from human)
  --last-author=human,ai

All output is JSON.

Examples:
  critic agent conversations
  critic agent conversations --status=actionable
  critic agent conversations --status=unresolved
  critic agent conversations --last-author=human
  critic agent conversations --status=unresolved --last-author=human,ai
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			gitRoot := git.GetGitRoot()

			mdb, err := messagedb.New(gitRoot)
			if err != nil {
				return fmt.Errorf("failed to initialize message database: %w", err)
			}
			defer mdb.Close()

			return runAgentConversations(cmd, mdb, statusFilter, lastAuthorFilter)
		},
	}

	cmd.Flags().StringVar(&statusFilter, "status", "", "Filter by status (comma-separated): unresolved, resolved, actionable (unresolved + last msg from human)")
	cmd.Flags().StringVar(&lastAuthorFilter, "last-author", "", "Filter by last message author (comma-separated): human, ai")

	return cmd
}

// runAgentConversations implements the conversations listing logic
func runAgentConversations(cmd *cobra.Command, messaging critic.Messaging, statusFilter, lastAuthorFilter string) error {
	statusSet := parseCommaSeparated(statusFilter)
	lastAuthorSet := parseCommaSeparated(lastAuthorFilter)

	// Delegate status filtering to the DB layer, which supports virtual
	// statuses like "actionable" (unresolved + last message from human).
	// For multi-value status filters, make separate calls and merge.
	var roots []*critic.Conversation
	if len(statusSet) == 0 {
		var err error
		roots, err = messaging.GetConversations("", nil)
		if err != nil {
			return fmt.Errorf("failed to get conversations: %w", err)
		}
	} else {
		seen := make(map[string]bool)
		for status := range statusSet {
			convs, err := messaging.GetConversations(status, nil)
			if err != nil {
				return fmt.Errorf("failed to get conversations: %w", err)
			}
			for _, conv := range convs {
				if !seen[conv.UUID] {
					seen[conv.UUID] = true
					roots = append(roots, conv)
				}
			}
		}
	}

	if len(roots) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "[]")
		return nil
	}

	// Batch-fetch full conversations to determine last author
	uuids := make([]string, len(roots))
	for i, r := range roots {
		uuids[i] = r.UUID
	}
	conversations, err := messaging.GetFullConversations(uuids)
	if err != nil {
		return fmt.Errorf("failed to get full conversations: %w", err)
	}

	entries := make([]AgentConversationEntry, 0, len(conversations))
	for _, conv := range conversations {
		if len(conv.Messages) == 0 {
			continue
		}

		lastMsg := conv.Messages[len(conv.Messages)-1]
		lastAuthor := string(lastMsg.Author)

		// Apply last-author filter
		if len(lastAuthorSet) > 0 && !lastAuthorSet[lastAuthor] {
			continue
		}

		entries = append(entries, AgentConversationEntry{
			UUID:   conv.UUID,
			Author: lastAuthor,
			Status: string(conv.Status),
		})
	}

	fmt.Fprintln(cmd.OutOrStdout(), json.ToPrettyJson(entries))
	return nil
}

// newAgentConversationCmd creates the agent conversation (singular) command
func newAgentConversationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "conversation <uuid>",
		Short: "Show a complete conversation",
		Long: `Get the complete conversation including all messages and replies.
Returns conversation metadata and all messages ordered chronologically.

Example:
  critic agent conversation a1b2c3d4-e5f6-7890-abcd-ef1234567890
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

			return runAgentConversation(cmd, mdb, uuid)
		},
	}

	return cmd
}

// runAgentConversation implements the conversation show logic
func runAgentConversation(cmd *cobra.Command, messaging critic.Messaging, uuid string) error {
	conversation, err := messaging.GetFullConversation(uuid)
	if err != nil {
		return fmt.Errorf("failed to get conversation: %w", err)
	}

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

	fmt.Fprintln(cmd.OutOrStdout(), json.ToPrettyJson(response))
	return nil
}

// newAgentReplyCmd creates the agent reply command
func newAgentReplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reply <uuid> <message>",
		Short: "Reply to a conversation",
		Long: `Post a reply to an existing conversation as the AI agent.

Examples:
  critic agent reply a1b2c3d4-e5f6-7890-abcd-ef1234567890 "I've fixed the issue"
`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			uuid := args[0]
			message := args[1]

			gitRoot := git.GetGitRoot()

			mdb, err := messagedb.New(gitRoot)
			if err != nil {
				return fmt.Errorf("failed to initialize message database: %w", err)
			}
			defer mdb.Close()

			return runAgentReply(cmd, mdb, uuid, message)
		},
	}

	return cmd
}

// runAgentReply implements the reply logic
func runAgentReply(cmd *cobra.Command, messaging critic.Messaging, uuid, message string) error {
	reply, err := messaging.ReplyToConversation(uuid, message, critic.AuthorAI)
	if err != nil {
		return fmt.Errorf("failed to create reply: %w", err)
	}

	response := ReplyResponse{
		UUID:      reply.UUID,
		Author:    string(reply.Author),
		Message:   reply.Message,
		CreatedAt: reply.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	fmt.Fprintln(cmd.OutOrStdout(), json.ToPrettyJson(response))
	return nil
}

// newAgentAnnounceCmd creates the agent announce command
func newAgentAnnounceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "announce <message>",
		Short: "Post an announcement visible in the Critic UI",
		Long: `Post an announcement that appears as a banner in the Critic UI.
Creates a message on the root conversation and marks it as unresolved.

Examples:
  critic agent announce "Please review the auth changes before merging"
  critic agent announce "Build is broken, do not merge"
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

			return runAgentAnnounce(cmd, mdb, message)
		},
	}

	return cmd
}

// runAgentAnnounce implements the announce logic
func runAgentAnnounce(cmd *cobra.Command, messaging critic.Messaging, message string) error {
	rootConv, err := messaging.LoadRootConversation()
	if err != nil {
		return fmt.Errorf("failed to load root conversation: %w", err)
	}

	reply, err := messaging.ReplyToConversation(rootConv.UUID, message, critic.AuthorAI)
	if err != nil {
		return fmt.Errorf("failed to create announcement: %w", err)
	}

	if err := messaging.MarkConversationAs(rootConv.UUID, critic.ConversationUnresolved); err != nil {
		return fmt.Errorf("failed to mark announcement as unresolved: %w", err)
	}

	response := ReplyResponse{
		UUID:      reply.UUID,
		Author:    string(reply.Author),
		Message:   reply.Message,
		CreatedAt: reply.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	fmt.Fprintln(cmd.OutOrStdout(), json.ToPrettyJson(response))
	return nil
}

// newAgentExplainCmd creates the agent explain command
func newAgentExplainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "explain <file> <line> <comment>",
		Short: "Post an explanation on a specific code line",
		Long: `Post an explanation on a specific code line.
Explanations are informal annotations shown with a lightbulb icon in the Critic UI.

Examples:
  critic agent explain src/main.go 42 "This function handles the auth flow"
  critic agent explain src/util.go 10 "Using a mutex here to prevent race conditions"
`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			file := args[0]
			lineStr := args[1]
			comment := args[2]

			line, err := strconv.Atoi(lineStr)
			if err != nil {
				return fmt.Errorf("invalid line number: %s", lineStr)
			}
			if line <= 0 {
				return fmt.Errorf("line number must be positive: %d", line)
			}

			gitRoot := git.GetGitRoot()

			mdb, err := messagedb.New(gitRoot)
			if err != nil {
				return fmt.Errorf("failed to initialize message database: %w", err)
			}
			defer mdb.Close()

			codeVersion := git.ResolveRef("HEAD")

			return runAgentExplain(cmd, mdb, file, line, comment, codeVersion)
		},
	}

	return cmd
}

// runAgentExplain implements the explain logic
func runAgentExplain(cmd *cobra.Command, messaging critic.Messaging, file string, line int, comment, codeVersion string) error {
	conversation, err := messaging.CreateConversation(critic.AuthorAI, comment, file, line, codeVersion, "", critic.TypeExplanation)
	if err != nil {
		return fmt.Errorf("failed to create explanation: %w", err)
	}

	response := ReplyResponse{
		UUID:      conversation.UUID,
		Author:    string(critic.AuthorAI),
		Message:   comment,
		CreatedAt: conversation.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	fmt.Fprintln(cmd.OutOrStdout(), json.ToPrettyJson(response))
	return nil
}

// parseCommaSeparated splits a comma-separated string into a set of trimmed values.
// Returns an empty map if the input is empty.
func parseCommaSeparated(s string) map[string]bool {
	if s == "" {
		return nil
	}
	result := make(map[string]bool)
	for _, v := range strings.Split(s, ",") {
		v = strings.TrimSpace(v)
		if v != "" {
			result[v] = true
		}
	}
	return result
}

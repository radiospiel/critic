package critic

import (
	"time"

	"github.com/samber/lo"
)

// Author represents who authored a message
type Author string

const (
	AuthorHuman Author = "human"
	AuthorAI    Author = "ai"
)

// ConversationStatus represents the status of a conversation
type ConversationStatus string

const (
	StatusUnresolved ConversationStatus = "unresolved"
	StatusResolved   ConversationStatus = "resolved"
)

// Message represents a single message in a conversation
type Message struct {
	UUID      string
	Author    Author
	Message   string
	CreatedAt time.Time
	UpdatedAt time.Time
	IsUnread  bool // Only relevant for AI messages
}

// Conversation represents a conversation with its location and all messages
type Conversation struct {
	UUID        string // UUID of the root message
	Status      ConversationStatus
	FilePath    string    // Git-relative path to the file
	LineNumber  int       // Line number in the file
	CodeVersion string    // Git commit or version identifier
	Context     string    // Code context around the commented line
	Messages    []Message // Root message + all replies, ordered by created_at
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// FileConversationSummary contains information about conversations for a specific file
type FileConversationSummary struct {
	FilePath              string
	HasUnresolvedComments bool
	HasResolvedComments   bool
	HasUnreadAIMessages   bool
}

// Messaging defines the interface for managing critic conversations
type Messaging interface {
	// GetConversations returns a list of root-level conversations
	// If status is provided, filters by that status (e.g., "unresolved")
	// If status is empty, returns all conversations
	// Only returns the root message info, not the full thread
	GetConversations(status string) ([]Conversation, error)

	// GetConversationsByFile returns all conversations for a specific file
	// Returns root-level conversations ordered by line number
	GetConversationsByFile(filePath string) ([]Conversation, error)

	// GetFullConversation returns the complete conversation including all replies
	// Messages are ordered by created_at (root message first, then replies in chronological order)
	GetFullConversation(uuid string) (*Conversation, error)

	// GetConversationsForFile returns all conversations for a specific file
	GetConversationsForFile(filePath string) ([]*Conversation, error)

	// GetFileConversationSummary returns a summary of conversations for a file
	// This is used for efficient file list rendering
	GetFileConversationSummary(filePath string) (*FileConversationSummary, error)

	// ReplyToConversation adds a reply to an existing conversation
	ReplyToConversation(conversationUUID string, message string, author Author) (*Message, error)

	// CreateConversation creates a new conversation (root message)
	CreateConversation(author Author, message, filePath string, lineNumber int, codeVersion string, context string) (*Conversation, error)

	// MarkAsResolved marks a conversation as resolved
	MarkAsResolved(conversationUUID string) error

	// MarkAsUnresolved marks a conversation as unresolved
	MarkAsUnresolved(conversationUUID string) error

	// MarkAsRead marks an AI message as read
	MarkAsRead(messageUUID string) error

	// Close closes the messaging system and releases resources
	Close() error
}

// GetFilesWithComments returns a list of unique file paths that have conversations
func GetFilesWithComments(m Messaging) ([]string, error) {
	conversations, err := m.GetConversations("")
	if err != nil {
		return nil, err
	}

	filePaths := lo.Map(conversations, func(conv Conversation, _ int) string {
		return conv.FilePath
	})
	return lo.Uniq(filePaths), nil
}

// GetFilesWithUnreadAIMessages returns a list of unique file paths that have unread AI messages
func GetFilesWithUnreadAIMessages(m Messaging) ([]string, error) {
	conversations, err := m.GetConversations("")
	if err != nil {
		return nil, err
	}

	// TODO: future improvement: this can be optimized with SQL
	// Get full conversations (skipping any that fail to load)
	fullConvs := lo.FilterMap(conversations, func(conv Conversation, _ int) (*Conversation, bool) {
		fullConv, err := m.GetFullConversation(conv.UUID)
		return fullConv, err == nil
	})

	// Filter to those with unread AI messages
	convsWithUnread := lo.Filter(fullConvs, func(conv *Conversation, _ int) bool {
		return lo.ContainsBy(conv.Messages, func(msg Message) bool {
			return msg.Author == AuthorAI && msg.IsUnread
		})
	})

	filePaths := lo.Map(convsWithUnread, func(conv *Conversation, _ int) string {
		return conv.FilePath
	})
	return lo.Uniq(filePaths), nil
}

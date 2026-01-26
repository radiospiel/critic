package critic

import (
	"time"
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
	ReadByAI    bool // Whether the AI has read this conversation via MCP
}

// FileConversationSummary contains information about Conversations for a specific file
type FileConversationSummary struct {
	FilePath              string
	HasUnresolvedComments bool
	HasResolvedComments   bool
	HasUnreadAIMessages   bool
}

// Messaging defines the interface for managing critic Conversations
type Messaging interface {
	// GetConversations returns a list of root-level Conversations
	// If status is provided, filters by that status (e.g., "unresolved")
	// If status is empty, returns all Conversations
	// Only returns the root message info, not the full thread
	GetConversations(status string) ([]Conversation, error)

	// GetFullConversation returns the complete conversation including all replies
	// Messages are ordered by created_at (root message first, then replies in chronological order)
	GetFullConversation(uuid string) (*Conversation, error)

	// GetConversationsForFile returns all Conversations for a specific file
	GetConversationsForFile(filePath string) ([]*Conversation, error)

	// GetFileConversationSummary returns a summary of Conversations for a file
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

	// MarkAsReadByAI marks a conversation as having been read by the AI
	MarkAsReadByAI(conversationUUID string) error

	// Close closes the messaging system and releases resources
	Close() error
}

// DummyMessaging implements critic.Messaging for testing
type DummyMessaging struct {
	Conversations map[string][]*Conversation
	Summaries     map[string]*FileConversationSummary
}

func NewDummyMessaging() *DummyMessaging {
	return &DummyMessaging{
		Conversations: make(map[string][]*Conversation),
		Summaries:     make(map[string]*FileConversationSummary),
	}
}

func (m *DummyMessaging) GetConversations(status string) ([]Conversation, error) {
	var all []Conversation
	for _, convs := range m.Conversations {
		for _, c := range convs {
			all = append(all, *c)
		}
	}
	return all, nil
}

func (m *DummyMessaging) GetFullConversation(uuid string) (*Conversation, error) {
	for _, convs := range m.Conversations {
		for _, c := range convs {
			if c.UUID == uuid {
				return c, nil
			}
		}
	}
	return nil, nil
}

func (m *DummyMessaging) GetConversationsForFile(filePath string) ([]*Conversation, error) {
	return m.Conversations[filePath], nil
}

func (m *DummyMessaging) GetFileConversationSummary(filePath string) (*FileConversationSummary, error) {
	return m.Summaries[filePath], nil
}

func (m *DummyMessaging) ReplyToConversation(conversationUUID string, message string, author Author) (*Message, error) {
	return &Message{UUID: "reply-1"}, nil
}

func (m *DummyMessaging) CreateConversation(author Author, message, filePath string, lineNumber int, codeVersion string, context string) (*Conversation, error) {
	return &Conversation{UUID: "conv-1"}, nil
}

func (m *DummyMessaging) MarkAsResolved(conversationUUID string) error   { return nil }
func (m *DummyMessaging) MarkAsUnresolved(conversationUUID string) error { return nil }
func (m *DummyMessaging) MarkAsRead(messageUUID string) error            { return nil }
func (m *DummyMessaging) MarkAsReadByAI(conversationUUID string) error   { return nil }
func (m *DummyMessaging) Close() error                                   { return nil }

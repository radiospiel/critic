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
	StatusUnresolved         ConversationStatus = "unresolved"
	StatusResolved           ConversationStatus = "resolved"
	StatusActive             ConversationStatus = "active"
	StatusWaitingForResponse ConversationStatus = "waiting_for_response"
	StatusInformal           ConversationStatus = "informal"
	StatusArchived           ConversationStatus = "archived"
)

// ConversationType represents the type of a conversation
type ConversationType string

const (
	TypeConversation ConversationType = "conversation"
	TypeExplanation  ConversationType = "explanation"
)

// ConversationUpdate represents an update to apply to a conversation
type ConversationUpdate string

const (
	ConversationResolved   ConversationUpdate = "resolved"
	ConversationUnresolved ConversationUpdate = "unresolved"
	ConversationArchived   ConversationUpdate = "archived"
	ConversationReadByAI   ConversationUpdate = "read_by_ai"
)

// MessageReadStatus represents the read status of a message
type MessageReadStatus string

const (
	MessageRead   MessageReadStatus = "read"
	MessageUnread MessageReadStatus = "unread"
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
	UUID             string // UUID of the root message
	Status           ConversationStatus
	ConversationType ConversationType
	FilePath         string    // Git-relative path to the file
	LineNumber       int       // Line number in the file
	CodeVersion      string    // Git commit or version identifier
	Context          string    // Code context around the commented line
	Messages         []Message // Root message + all replies, ordered by created_at
	CreatedAt        time.Time
	UpdatedAt        time.Time
	ReadByAI         bool // Whether the AI has read this conversation via MCP
}

// FileConversationSummary contains information about Conversations for a specific file
type FileConversationSummary struct {
	FilePath              string
	TotalCount            int
	UnresolvedCount       int
	ResolvedCount         int
	ExplanationCount      int
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

	// GetConversationsSummary returns summaries for all files that have conversations
	// Only includes files with at least one conversation
	GetConversationsSummary() ([]*FileConversationSummary, error)

	// ReplyToConversation adds a reply to an existing conversation
	ReplyToConversation(conversationUUID string, message string, author Author) (*Message, error)

	// CreateConversation creates a new conversation (root message)
	CreateConversation(author Author, message, filePath string, lineNumber int, codeVersion string, context string) (*Conversation, error)

	// CreateExplanation creates a new explanation (informal annotation on a code line)
	CreateExplanation(author Author, comment, filePath string, lineNumber int, codeVersion string, context string) (*Conversation, error)

	// MarkConversationAs applies an update to a conversation (resolved, unresolved, read_by_ai)
	MarkConversationAs(conversationUUID string, update ConversationUpdate) error

	// MarkMessageAs marks a message with a given read status
	MarkMessageAs(messageUUID string, status MessageReadStatus) error

	// LoadRootConversation returns the root conversation (filePath="", lineNumber=0).
	// If it doesn't exist, it creates one.
	LoadRootConversation() (*Conversation, error)

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

func (m *DummyMessaging) GetConversationsSummary() ([]*FileConversationSummary, error) {
	var summaries []*FileConversationSummary
	for _, s := range m.Summaries {
		summaries = append(summaries, s)
	}
	return summaries, nil
}

func (m *DummyMessaging) ReplyToConversation(conversationUUID string, message string, author Author) (*Message, error) {
	return &Message{UUID: "reply-1"}, nil
}

func (m *DummyMessaging) CreateConversation(author Author, message, filePath string, lineNumber int, codeVersion string, context string) (*Conversation, error) {
	return &Conversation{UUID: "conv-1"}, nil
}

func (m *DummyMessaging) CreateExplanation(author Author, comment, filePath string, lineNumber int, codeVersion string, context string) (*Conversation, error) {
	return &Conversation{UUID: "expl-1", ConversationType: TypeExplanation, Status: StatusInformal}, nil
}

func (m *DummyMessaging) MarkConversationAs(conversationUUID string, update ConversationUpdate) error {
	return nil
}
func (m *DummyMessaging) MarkMessageAs(messageUUID string, status MessageReadStatus) error {
	return nil
}
func (m *DummyMessaging) LoadRootConversation() (*Conversation, error) {
	return &Conversation{UUID: "root-conv"}, nil
}
func (m *DummyMessaging) Close() error { return nil }

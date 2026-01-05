package messagedb

import (
	"fmt"

	"git.15b.it/eno/critic/pkg/messaging"
	"git.15b.it/eno/critic/internal/logger"
)

// Ensure DB implements the messaging.Messaging interface
var _ messaging.Messaging = (*DB)(nil)

// GetConversations returns a list of conversation IDs
// If status is provided, filters by that status (e.g., "unresolved")
// If status is empty, returns all conversations
func (db *DB) GetConversations(status string) ([]string, error) {
	var query string
	var args []interface{}

	if status == "" {
		// Get all root messages (conversations)
		query = `
			SELECT id
			FROM messages
			WHERE id = conversation_id
			ORDER BY file_path, line_number, created_at ASC
		`
	} else if status == string(messaging.StatusUnresolved) {
		// Get unresolved conversations
		query = `
			SELECT id
			FROM messages
			WHERE id = conversation_id AND status != ?
			ORDER BY file_path, line_number, created_at ASC
		`
		args = []interface{}{string(StatusResolved)}
	} else if status == string(messaging.StatusResolved) {
		// Get resolved conversations
		query = `
			SELECT id
			FROM messages
			WHERE id = conversation_id AND status = ?
			ORDER BY file_path, line_number, created_at ASC
		`
		args = []interface{}{string(StatusResolved)}
	} else {
		return nil, fmt.Errorf("invalid status: %s", status)
	}

	rows, err := db.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversations: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan ID: %w", err)
		}
		ids = append(ids, id)
	}

	logger.Debug("Found %d conversations (status: %s)", len(ids), status)
	return ids, nil
}

// GetFullConversation returns the complete conversation including all replies
// Messages are ordered by created_at (root message first, then replies in chronological order)
func (db *DB) GetFullConversation(conversationID string) (*messaging.Conversation, error) {
	// Get all messages in the conversation
	messages, err := db.GetThreadMessages(conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation messages: %w", err)
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("conversation not found: %s", conversationID)
	}

	// First message is the root
	rootMsg := messages[0]

	// Convert messages to messaging.Message type
	criticMessages := make([]messaging.Message, len(messages))
	for i, msg := range messages {
		criticMessages[i] = messaging.Message{
			UUID:      msg.ID,
			Author:    messaging.Author(msg.Author),
			Message:   msg.Message,
			CreatedAt: msg.CreatedAt,
			UpdatedAt: msg.UpdatedAt,
			IsUnread:  msg.ReadStatus == ReadStatusUnread,
		}
	}

	conversation := &messaging.Conversation{
		UUID:        rootMsg.ID,
		Status:      convertToCriticStatus(rootMsg.Status),
		FilePath:    rootMsg.FilePath,
		LineNumber:  rootMsg.LineNumber,
		CodeVersion: rootMsg.CodeVersion,
		Messages:    criticMessages,
		CreatedAt:   rootMsg.CreatedAt,
		UpdatedAt:   rootMsg.UpdatedAt,
	}

	logger.Debug("Retrieved conversation %s with %d messages", conversationID, len(criticMessages))
	return conversation, nil
}

// ReplyToConversation adds a reply to an existing conversation
func (db *DB) ReplyToConversation(conversationID string, message string, author messaging.Author) (*messaging.Message, error) {
	// Verify conversation exists
	rootMsg, err := db.GetMessage(conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}
	if rootMsg == nil {
		return nil, fmt.Errorf("conversation not found: %s", conversationID)
	}

	// Create the reply
	dbAuthor := Author(author)
	reply, err := db.CreateReply(dbAuthor, message, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to create reply: %w", err)
	}

	criticMsg := &messaging.Message{
		UUID:      reply.ID,
		Author:    messaging.Author(reply.Author),
		Message:   reply.Message,
		CreatedAt: reply.CreatedAt,
		UpdatedAt: reply.UpdatedAt,
		IsUnread:  reply.ReadStatus == ReadStatusUnread,
	}

	logger.Info("Created reply %s to conversation %s", reply.ID, conversationID)
	return criticMsg, nil
}

// CreateConversation creates a new conversation (root message)
func (db *DB) CreateConversation(author messaging.Author, message, filePath string, lineNumber int, codeVersion string) (*messaging.Conversation, error) {
	dbAuthor := Author(author)
	rootMsg, err := db.CreateMessage(dbAuthor, message, filePath, lineNumber, codeVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	conversation := &messaging.Conversation{
		UUID:        rootMsg.ID,
		Status:      convertToCriticStatus(rootMsg.Status),
		FilePath:    rootMsg.FilePath,
		LineNumber:  rootMsg.LineNumber,
		CodeVersion: rootMsg.CodeVersion,
		Messages: []messaging.Message{
			{
				UUID:      rootMsg.ID,
				Author:    messaging.Author(rootMsg.Author),
				Message:   rootMsg.Message,
				CreatedAt: rootMsg.CreatedAt,
				UpdatedAt: rootMsg.UpdatedAt,
				IsUnread:  rootMsg.ReadStatus == ReadStatusUnread,
			},
		},
		CreatedAt: rootMsg.CreatedAt,
		UpdatedAt: rootMsg.UpdatedAt,
	}

	logger.Info("Created conversation %s at %s:%d", conversation.UUID, filePath, lineNumber)
	return conversation, nil
}

// convertToCriticStatus converts messagedb.Status to messaging.ConversationStatus
func convertToCriticStatus(status Status) messaging.ConversationStatus {
	if status == StatusResolved {
		return messaging.StatusResolved
	}
	return messaging.StatusUnresolved
}

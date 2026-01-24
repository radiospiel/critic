package messagedb

import (
	"fmt"

	"git.15b.it/eno/critic/pkg/critic"
	"git.15b.it/eno/critic/simple-go/logger"
)

// Ensure DB implements the critic.Messaging interface
var _ critic.Messaging = (*DB)(nil)

// GetConversations returns a list of root-level conversations
// If status is provided, filters by that status (e.g., "unresolved")
// If status is empty, returns all conversations
func (db *DB) GetConversations(status string) ([]critic.Conversation, error) {
	var query string
	var args []interface{}

	if status == "" {
		// Get all root messages (conversations)
		query = `
			SELECT id, status, file_path, lineno, sha1, context, created_at, updated_at
			FROM messages
			WHERE id = conversation_id
			ORDER BY file_path, lineno, created_at ASC
		`
	} else if status == string(critic.StatusUnresolved) {
		// Get unresolved conversations
		query = `
			SELECT id, status, file_path, lineno, sha1, context, created_at, updated_at
			FROM messages
			WHERE id = conversation_id AND status != ?
			ORDER BY file_path, lineno, created_at ASC
		`
		args = []interface{}{string(StatusResolved)}
	} else if status == string(critic.StatusResolved) {
		// Get resolved conversations
		query = `
			SELECT id, status, file_path, lineno, sha1, context, created_at, updated_at
			FROM messages
			WHERE id = conversation_id AND status = ?
			ORDER BY file_path, lineno, created_at ASC
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

	var conversations []critic.Conversation
	for rows.Next() {
		var conv critic.Conversation
		var status string
		var context *string
		if err := rows.Scan(&conv.UUID, &status, &conv.FilePath, &conv.LineNumber, &conv.CodeVersion, &context, &conv.CreatedAt, &conv.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan conversation: %w", err)
		}
		conv.Status = convertToCriticStatus(Status(status))
		if context != nil {
			conv.Context = *context
		}
		conversations = append(conversations, conv)
	}

	logger.Debug("Found %d conversations (status: %s)", len(conversations), status)
	return conversations, nil
}

// GetFullConversation returns the complete conversation including all replies
// Messages are ordered by created_at (root message first, then replies in chronological order)
func (db *DB) GetFullConversation(conversationID string) (*critic.Conversation, error) {
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

	// Convert messages to critic.Message type
	criticMessages := make([]critic.Message, len(messages))
	for i, msg := range messages {
		criticMessages[i] = critic.Message{
			UUID:      msg.ID,
			Author:    critic.Author(msg.Author),
			Message:   msg.Message,
			CreatedAt: msg.CreatedAt,
			UpdatedAt: msg.UpdatedAt,
			IsUnread:  msg.ReadStatus == ReadStatusUnread,
		}
	}

	conversation := &critic.Conversation{
		UUID:        rootMsg.ID,
		Status:      convertToCriticStatus(rootMsg.Status),
		FilePath:    rootMsg.FilePath,
		LineNumber:  rootMsg.Lineno,
		CodeVersion: rootMsg.Commit,
		Context:     rootMsg.Context,
		Messages:    criticMessages,
		CreatedAt:   rootMsg.CreatedAt,
		UpdatedAt:   rootMsg.UpdatedAt,
		ReadByAI:    rootMsg.ReadByAI,
	}

	logger.Debug("Retrieved conversation %s with %d messages", conversationID, len(criticMessages))
	return conversation, nil
}

// GetConversationsForFile returns all conversations for a specific file
func (db *DB) GetConversationsForFile(filePath string) ([]*critic.Conversation, error) {
	// Get all root messages for this file
	rootMessages, err := db.GetMessagesByFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages by file: %w", err)
	}

	// Build full conversations for each root message
	conversations := make([]*critic.Conversation, 0, len(rootMessages))
	for _, rootMsg := range rootMessages {
		conv, err := db.GetFullConversation(rootMsg.ID)
		if err != nil {
			logger.Warn("Failed to get conversation %s: %v", rootMsg.ID, err)
			continue
		}
		conversations = append(conversations, conv)
	}

	return conversations, nil
}

// GetFileConversationSummary returns a summary of conversations for a file
func (db *DB) GetFileConversationSummary(filePath string) (*critic.FileConversationSummary, error) {
	summary := &critic.FileConversationSummary{
		FilePath: filePath,
	}

	// Query to check for unresolved, resolved, and unread conversations in one go
	query := `
		SELECT
			CASE WHEN status != 'resolved' THEN 1 ELSE 0 END as is_unresolved,
			CASE WHEN status = 'resolved' THEN 1 ELSE 0 END as is_resolved,
			CASE WHEN author = 'ai' AND read_status = 'unread' THEN 1 ELSE 0 END as has_unread_ai
		FROM messages
		WHERE file_path = ? AND id = conversation_id
	`

	rows, err := db.db.Query(query, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to query file summary: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var isUnresolved, isResolved, hasUnreadAI int
		if err := rows.Scan(&isUnresolved, &isResolved, &hasUnreadAI); err != nil {
			return nil, fmt.Errorf("failed to scan file summary: %w", err)
		}
		if isUnresolved == 1 {
			summary.HasUnresolvedComments = true
		}
		if isResolved == 1 {
			summary.HasResolvedComments = true
		}
		if hasUnreadAI == 1 {
			summary.HasUnreadAIMessages = true
		}
	}

	// Also check for unread AI messages in replies (not just root messages)
	if !summary.HasUnreadAIMessages {
		unreadQuery := `
			SELECT COUNT(*) FROM messages
			WHERE file_path = ? AND author = 'ai' AND read_status = 'unread'
		`
		var count int
		if err := db.db.QueryRow(unreadQuery, filePath).Scan(&count); err == nil && count > 0 {
			summary.HasUnreadAIMessages = true
		}
	}

	return summary, nil
}

// ReplyToConversation adds a reply to an existing conversation
func (db *DB) ReplyToConversation(conversationID string, message string, author critic.Author) (*critic.Message, error) {
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

	criticMsg := &critic.Message{
		UUID:      reply.ID,
		Author:    critic.Author(reply.Author),
		Message:   reply.Message,
		CreatedAt: reply.CreatedAt,
		UpdatedAt: reply.UpdatedAt,
		IsUnread:  reply.ReadStatus == ReadStatusUnread,
	}

	logger.Info("Created reply %s to conversation %s", reply.ID, conversationID)
	return criticMsg, nil
}

// CreateConversation creates a new conversation (root message)
func (db *DB) CreateConversation(author critic.Author, message, filePath string, lineNumber int, codeVersion string, context string) (*critic.Conversation, error) {
	dbAuthor := Author(author)
	rootMsg, err := db.CreateMessage(dbAuthor, message, filePath, lineNumber, codeVersion, context)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	conversation := &critic.Conversation{
		UUID:        rootMsg.ID,
		Status:      convertToCriticStatus(rootMsg.Status),
		FilePath:    rootMsg.FilePath,
		LineNumber:  rootMsg.Lineno,
		CodeVersion: rootMsg.Commit,
		Context:     rootMsg.Context,
		Messages: []critic.Message{
			{
				UUID:      rootMsg.ID,
				Author:    critic.Author(rootMsg.Author),
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

// convertToCriticStatus converts messagedb.Status to critic.ConversationStatus
func convertToCriticStatus(status Status) critic.ConversationStatus {
	if status == StatusResolved {
		return critic.StatusResolved
	}
	return critic.StatusUnresolved
}

package messagedb

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/radiospiel/critic/simple-go/logger"
	"github.com/radiospiel/critic/simple-go/preconditions"
	"github.com/radiospiel/critic/src/pkg/critic"
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
			SELECT id, status, file_path, lineno, sha1, context, conversation_type, created_at, updated_at
			FROM messages
			WHERE id = conversation_id
			ORDER BY file_path, lineno, created_at ASC
		`
	} else if status == string(critic.StatusUnresolved) {
		// Get unresolved conversations
		query = `
			SELECT id, status, file_path, lineno, sha1, context, conversation_type, created_at, updated_at
			FROM messages
			WHERE id = conversation_id AND status != ?
			ORDER BY file_path, lineno, created_at ASC
		`
		args = []interface{}{string(StatusResolved)}
	} else if status == string(critic.StatusResolved) {
		// Get resolved conversations
		query = `
			SELECT id, status, file_path, lineno, sha1, context, conversation_type, created_at, updated_at
			FROM messages
			WHERE id = conversation_id AND status = ?
			ORDER BY file_path, lineno, created_at ASC
		`
		args = []interface{}{string(StatusResolved)}
	} else {
		return nil, fmt.Errorf("invalid status: %s", status)
	}

	rows, err := db.query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversations: %w", err)
	}
	defer rows.Close()

	var conversations []critic.Conversation
	for rows.Next() {
		var conv critic.Conversation
		var status string
		var convType string
		var context *string
		if err := rows.Scan(&conv.UUID, &status, &conv.FilePath, &conv.LineNumber, &conv.CodeVersion, &context, &convType, &conv.CreatedAt, &conv.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan conversation: %w", err)
		}
		conv.Status = convertToCriticStatus(Status(status))
		conv.ConversationType = convertToCriticType(ConversationType(convType))
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
		UUID:             rootMsg.ID,
		Status:           convertToCriticStatus(rootMsg.Status),
		ConversationType: convertToCriticType(rootMsg.ConversationType),
		FilePath:         rootMsg.FilePath,
		LineNumber:       rootMsg.Lineno,
		CodeVersion:      rootMsg.Commit,
		Context:          rootMsg.Context,
		Messages:         criticMessages,
		CreatedAt:        rootMsg.CreatedAt,
		UpdatedAt:        rootMsg.UpdatedAt,
		ReadByAI:         rootMsg.ReadByAI,
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

	preconditions.Check(rootMessages != nil, "rootMessages cannot be nil")
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

// GetConversationsSummary returns summaries for all files that have conversations
func (db *DB) GetConversationsSummary() ([]*critic.FileConversationSummary, error) {
	// Query to get summaries for all files that have conversations
	query := `
		SELECT
			file_path,
			SUM(CASE WHEN status NOT IN ('resolved', 'informal') THEN 1 ELSE 0 END) as unresolved_count,
			SUM(CASE WHEN status = 'resolved' THEN 1 ELSE 0 END) as resolved_count,
			SUM(CASE WHEN conversation_type = 'explanation' THEN 1 ELSE 0 END) as explanation_count,
			COUNT(*) as total_count
		FROM messages
		WHERE id = conversation_id
		GROUP BY file_path
		ORDER BY file_path
	`

	rows, err := db.query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query file summaries: %w", err)
	}
	defer rows.Close()

	summaryMap := make(map[string]*critic.FileConversationSummary)
	for rows.Next() {
		var filePath string
		var unresolvedCount, resolvedCount, explanationCount, totalCount int
		if err := rows.Scan(&filePath, &unresolvedCount, &resolvedCount, &explanationCount, &totalCount); err != nil {
			return nil, fmt.Errorf("failed to scan file summary: %w", err)
		}
		summaryMap[filePath] = &critic.FileConversationSummary{
			FilePath:              filePath,
			TotalCount:            totalCount,
			UnresolvedCount:       unresolvedCount,
			ResolvedCount:         resolvedCount,
			ExplanationCount:      explanationCount,
			HasUnresolvedComments: unresolvedCount > 0,
			HasResolvedComments:   resolvedCount > 0,
		}
	}

	// Check for unread AI messages per file
	unreadQuery := `
		SELECT file_path, COUNT(*) as unread_count
		FROM messages
		WHERE author = 'ai' AND read_status = 'unread'
		GROUP BY file_path
	`
	unreadRows, err := db.query(unreadQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query unread AI messages: %w", err)
	}
	defer unreadRows.Close()

	for unreadRows.Next() {
		var filePath string
		var unreadCount int
		if err := unreadRows.Scan(&filePath, &unreadCount); err != nil {
			return nil, fmt.Errorf("failed to scan unread count: %w", err)
		}
		if summary, ok := summaryMap[filePath]; ok && unreadCount > 0 {
			summary.HasUnreadAIMessages = true
		}
	}

	// Convert map to slice
	summaries := make([]*critic.FileConversationSummary, 0, len(summaryMap))
	for _, summary := range summaryMap {
		summaries = append(summaries, summary)
	}

	logger.Debug("Found conversation summaries for %d files", len(summaries))
	return summaries, nil
}

// ReplyToConversation adds a reply to an existing conversation.
// If conversationID is empty, it replies to the root conversation.
func (db *DB) ReplyToConversation(conversationID string, message string, author critic.Author) (*critic.Message, error) {
	// If conversationID is empty, use the root conversation
	if conversationID == "" {
		rootConv, err := db.LoadRootConversation()
		if err != nil {
			return nil, fmt.Errorf("failed to load root conversation: %w", err)
		}
		conversationID = rootConv.UUID
	}

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

	logger.Info("Created %s reply %s to conversation %s", critic.Author(reply.Author), reply.ID, conversationID)
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
		UUID:             rootMsg.ID,
		Status:           convertToCriticStatus(rootMsg.Status),
		ConversationType: convertToCriticType(rootMsg.ConversationType),
		FilePath:         rootMsg.FilePath,
		LineNumber:       rootMsg.Lineno,
		CodeVersion:      rootMsg.Commit,
		Context:          rootMsg.Context,
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

// CreateExplanation creates a new explanation (informal annotation on a code line)
func (db *DB) CreateExplanation(author critic.Author, comment, filePath string, lineNumber int, codeVersion string, context string) (*critic.Conversation, error) {
	dbAuthor := Author(author)
	rootMsg, err := db.CreateMessageWithType(dbAuthor, comment, filePath, lineNumber, codeVersion, context, ConversationTypeExplanation)
	if err != nil {
		return nil, fmt.Errorf("failed to create explanation: %w", err)
	}

	// Set the status to informal
	if err := db.UpdateMessageStatus(rootMsg.ID, StatusInformal); err != nil {
		return nil, fmt.Errorf("failed to set explanation status: %w", err)
	}

	conversation := &critic.Conversation{
		UUID:             rootMsg.ID,
		Status:           critic.StatusInformal,
		ConversationType: critic.TypeExplanation,
		FilePath:         rootMsg.FilePath,
		LineNumber:       rootMsg.Lineno,
		CodeVersion:      rootMsg.Commit,
		Context:          rootMsg.Context,
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

	logger.Info("Created explanation %s at %s:%d", conversation.UUID, filePath, lineNumber)
	return conversation, nil
}

// LoadRootConversation returns the root conversation (filePath="", lineNumber=0).
// If it doesn't exist, it creates one.
func (db *DB) LoadRootConversation() (*critic.Conversation, error) {
	// Query for a conversation with file_path="" AND lineno=0 AND id=conversation_id (root message)
	query := `
		SELECT id FROM messages
		WHERE file_path = '' AND lineno = 0 AND id = conversation_id
		LIMIT 1
	`
	var id string
	err := logRuntime(query, func() error {
		return db.db.QueryRow(query).Scan(&id)
	})
	if err != nil {
		// Not found — insert a sentinel root message.
		id = uuid.Must(uuid.NewV7()).String()
		now := time.Now()
		msg := &Message{
			ID:             id,
			Author:         AuthorAI,
			Status:         StatusNew,
			ReadStatus:     ReadStatusRead,
			Message:        "",
			FilePath:       "",
			Lineno:         0,
			ConversationID: id,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		err := db.insertMessage(msg)
		preconditions.Check(err == nil, "failed to create root conversation: %v", err)

		logger.Info("Created root conversation w/id %s", id)
	}

	return db.GetFullConversation(id)
}

// convertToCriticStatus converts messagedb.Status to critic.ConversationStatus
func convertToCriticStatus(status Status) critic.ConversationStatus {
	switch status {
	case StatusResolved:
		return critic.StatusResolved
	case StatusInformal:
		return critic.StatusInformal
	case StatusArchived:
		return critic.StatusArchived
	default:
		return critic.StatusUnresolved
	}
}

// convertToCriticType converts messagedb.ConversationType to critic.ConversationType
func convertToCriticType(ct ConversationType) critic.ConversationType {
	switch ct {
	case ConversationTypeExplanation:
		return critic.TypeExplanation
	default:
		return critic.TypeConversation
	}
}

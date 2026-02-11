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
		query = `
			SELECT id, status, file_path, lineno, sha1, context, conversation_type, created_at, updated_at
			FROM messages
			WHERE id = conversation_id
			ORDER BY file_path, lineno, created_at ASC
		`
	} else if status == string(critic.StatusUnresolved) {
		query = `
			SELECT id, status, file_path, lineno, sha1, context, conversation_type, created_at, updated_at
			FROM messages
			WHERE id = conversation_id AND status != ?
			ORDER BY file_path, lineno, created_at ASC
		`
		args = []interface{}{string(StatusResolved)}
	} else if status == string(critic.StatusResolved) {
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

	conversations, err := all(db, query, scanConversation, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversations: %w", err)
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

	summaries, err := all(db, query, scanFileSummary)
	if err != nil {
		return nil, fmt.Errorf("failed to query file summaries: %w", err)
	}

	summaryMap := make(map[string]*critic.FileConversationSummary, len(summaries))
	for i := range summaries {
		summaryMap[summaries[i].FilePath] = summaries[i]
	}

	// Check for unread AI messages per file
	unreadQuery := `
		SELECT file_path
		FROM messages
		WHERE author = 'ai' AND read_status = 'unread'
		GROUP BY file_path
	`

	unreadFiles, err := all(db, unreadQuery, scanString)
	if err != nil {
		return nil, fmt.Errorf("failed to query unread AI messages: %w", err)
	}

	for _, filePath := range unreadFiles {
		if summary, ok := summaryMap[filePath]; ok {
			summary.HasUnreadAIMessages = true
		}
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

// CreateConversation creates a new conversation (root message).
// The conversationType determines the DB conversation type and initial status:
//
//	TypeConversation → StatusUnresolved (default)
//	TypeExplanation  → StatusInformal
func (db *DB) CreateConversation(author critic.Author, message, filePath string, lineNumber int, codeVersion string, context string, conversationType critic.ConversationType) (*critic.Conversation, error) {
	dbAuthor := Author(author)

	dbConvType := ConversationTypeConversation
	if conversationType == critic.TypeExplanation {
		dbConvType = ConversationTypeExplanation
	}

	rootMsg, err := db.CreateMessageWithType(dbAuthor, message, filePath, lineNumber, codeVersion, context, dbConvType)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s: %w", conversationType, err)
	}

	// Explanations get informal status instead of the default unresolved
	if conversationType == critic.TypeExplanation {
		if err := db.UpdateMessageStatus(rootMsg.ID, StatusInformal); err != nil {
			return nil, fmt.Errorf("failed to set explanation status: %w", err)
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

	// Re-read status after potential update
	if conversationType == critic.TypeExplanation {
		conversation.Status = critic.StatusInformal
	}

	logger.Info("Created %s %s at %s:%d", conversationType, conversation.UUID, filePath, lineNumber)
	return conversation, nil
}

// LoadRootConversation returns the root conversation (filePath="", lineNumber=0).
// If it doesn't exist, it creates one.
func (db *DB) LoadRootConversation() (*critic.Conversation, error) {
	var id string
	err := db.ask(`
		SELECT id FROM messages
		WHERE file_path = '' AND lineno = 0 AND id = conversation_id
		LIMIT 1
	`, nil, &id)
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

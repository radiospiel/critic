package messagedb

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/radiospiel/critic/simple-go/logger"
	"github.com/radiospiel/critic/simple-go/preconditions"
	"github.com/radiospiel/critic/src/pkg/critic"
)

// joinConditions joins SQL conditions with AND
func joinConditions(conditions []string) string {
	return strings.Join(conditions, " AND ")
}

// Ensure DB implements the critic.Messaging interface
var _ critic.Messaging = (*DB)(nil)

// GetConversations returns a list of root-level conversations
// If status is provided, filters by that status (e.g., "unresolved")
// If status is empty, returns all conversations
// If paths is provided, filters to conversations in those file paths
func (db *DB) GetConversations(status string, paths []string) ([]*critic.Conversation, error) {
	var conditions []string
	var args []interface{}

	if status == string(critic.StatusUnresolved) {
		conditions = append(conditions, "status != ?")
		args = append(args, string(StatusResolved))
	} else if status == string(critic.StatusResolved) {
		conditions = append(conditions, "status = ?")
		args = append(args, string(StatusResolved))
	} else if status != "" {
		return nil, fmt.Errorf("invalid status: %s", status)
	}

	if len(paths) > 0 {
		inQuery, inArgs, err := sqlx.In("file_path IN (?)", paths)
		if err != nil {
			return nil, fmt.Errorf("failed to build paths filter: %w", err)
		}
		conditions = append(conditions, inQuery)
		args = append(args, inArgs...)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + joinConditions(conditions)
	}

	query := fmt.Sprintf(`
		SELECT * FROM conversations
		%s
		ORDER BY file_path, lineno, created_at ASC
	`, whereClause)

	query = db.db.Rebind(query)
	var rows []ConversationRecord
	err := db.db.Select(&rows, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversations: %w", err)
	}

	conversations := make([]*critic.Conversation, len(rows))
	for i := range rows {
		conv := rows[i].toConversation()
		conversations[i] = &conv
	}

	logger.Debug("Found %d conversations (status: %s, paths: %v)", len(conversations), status, paths)
	return conversations, nil
}

// GetFullConversations returns complete conversations including all replies
// for the given conversation UUIDs.
func (db *DB) GetFullConversations(uuids []string) ([]*critic.Conversation, error) {
	if len(uuids) == 0 {
		return nil, nil
	}

	// Fetch conversation records
	convQuery, convArgs, err := sqlx.In(`SELECT * FROM conversations WHERE id IN (?)`, uuids)
	if err != nil {
		return nil, fmt.Errorf("failed to build conversations query: %w", err)
	}
	convQuery = db.db.Rebind(convQuery)

	var convRows []ConversationRecord
	if err := db.db.Select(&convRows, convQuery, convArgs...); err != nil {
		return nil, fmt.Errorf("failed to batch-fetch conversations: %w", err)
	}

	convMap := make(map[string]*ConversationRecord, len(convRows))
	for i := range convRows {
		convMap[convRows[i].ID] = &convRows[i]
	}

	// Fetch all messages for these conversations
	msgQuery, msgArgs, err := sqlx.In(`
		SELECT * FROM messages
		WHERE conversation_id IN (?)
		ORDER BY conversation_id, created_at ASC
	`, uuids)
	if err != nil {
		return nil, fmt.Errorf("failed to build messages query: %w", err)
	}
	msgQuery = db.db.Rebind(msgQuery)

	var messages []*MessageRecord
	if err := db.db.Select(&messages, msgQuery, msgArgs...); err != nil {
		return nil, fmt.Errorf("failed to batch-fetch messages: %w", err)
	}

	// Group messages by conversation_id
	grouped := make(map[string][]*MessageRecord)
	for _, msg := range messages {
		grouped[msg.ConversationID] = append(grouped[msg.ConversationID], msg)
	}

	// Build conversations preserving requested order
	conversations := make([]*critic.Conversation, 0, len(uuids))
	for _, id := range uuids {
		convRec, ok := convMap[id]
		if !ok {
			continue
		}

		conv := convRec.toConversation()
		if msgs, ok := grouped[id]; ok {
			conv.Messages = toCriticMessages(msgs)
		}

		conversations = append(conversations, &conv)
	}

	logger.Debug("Batch-fetched %d conversations", len(conversations))
	return conversations, nil
}

// GetFullConversation returns the complete conversation including all replies
// Messages are ordered by created_at (root message first, then replies in chronological order)
func (db *DB) GetFullConversation(conversationID string) (*critic.Conversation, error) {
	convRec, err := db.getConversation(conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}
	if convRec == nil {
		return nil, fmt.Errorf("conversation not found: %s", conversationID)
	}

	messages, err := db.GetThreadMessages(conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation messages: %w", err)
	}

	conv := convRec.toConversation()
	conv.Messages = toCriticMessages(messages)

	logger.Debug("Retrieved conversation %s with %d messages", conversationID, len(conv.Messages))
	return &conv, nil
}

// GetConversationsForFile returns all conversations for a specific file
func (db *DB) GetConversationsForFile(filePath string) ([]*critic.Conversation, error) {
	roots, err := db.GetConversationsByFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversations by file: %w", err)
	}

	preconditions.Check(roots != nil, "roots cannot be nil")

	conversations := make([]*critic.Conversation, 0, len(roots))
	for _, root := range roots {
		conv, err := db.GetFullConversation(root.ID)
		if err != nil {
			logger.Warn("Failed to get conversation %s: %v", root.ID, err)
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
		FROM conversations
		GROUP BY file_path
		ORDER BY file_path
	`

	var rows []fileSummaryRow
	err := db.db.Select(&rows, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query file summaries: %w", err)
	}

	summaries := make([]*critic.FileConversationSummary, len(rows))
	summaryMap := make(map[string]*critic.FileConversationSummary, len(rows))
	for i, row := range rows {
		summaries[i] = row.toSummary()
		summaryMap[summaries[i].FilePath] = summaries[i]
	}

	// Check for unread AI messages per file (needs join)
	unreadQuery := `
		SELECT DISTINCT c.file_path
		FROM messages m
		JOIN conversations c ON m.conversation_id = c.id
		WHERE m.author = 'ai' AND m.read_status = 'unread'
	`

	var unreadFiles []string
	err = db.db.Select(&unreadFiles, unreadQuery)
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
	conv, err := db.getConversation(conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}
	if conv == nil {
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

	convRec, msgRec, err := db.createConversationWithMessage(dbAuthor, message, filePath, lineNumber, codeVersion, context, dbConvType)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s: %w", conversationType, err)
	}

	// Explanations get informal status instead of the default unresolved
	if conversationType == critic.TypeExplanation {
		if err := db.UpdateConversationStatus(convRec.ID, StatusInformal); err != nil {
			return nil, fmt.Errorf("failed to set explanation status: %w", err)
		}
		convRec.Status = StatusInformal
	}

	conversation := convRec.toConversation()
	conversation.Messages = []critic.Message{
		{
			UUID:      msgRec.ID,
			Author:    critic.Author(msgRec.Author),
			Message:   msgRec.Message,
			CreatedAt: msgRec.CreatedAt,
			UpdatedAt: msgRec.UpdatedAt,
			IsUnread:  msgRec.ReadStatus == ReadStatusUnread,
		},
	}

	logger.Info("Created %s %s at %s:%d", conversationType, conversation.UUID, filePath, lineNumber)
	return &conversation, nil
}

// LoadRootConversation returns the root conversation (filePath="", lineNumber=0).
// If it doesn't exist, it creates one.
func (db *DB) LoadRootConversation() (*critic.Conversation, error) {
	var id string
	err := db.db.Get(&id, `
		SELECT id FROM conversations
		WHERE file_path = '' AND lineno = 0
		LIMIT 1
	`)
	if err != nil {
		// Not found — insert a sentinel conversation + message.
		id = uuid.Must(uuid.NewV7()).String()
		now := time.Now()

		conv := &ConversationRecord{
			ID:               id,
			Status:           StatusNew,
			FilePath:         "",
			Lineno:           0,
			ConversationType: ConversationTypeConversation,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		err := db.insertConversation(conv)
		preconditions.Check(err == nil, "failed to create root conversation: %v", err)

		msg := &MessageRecord{
			ID:             id,
			ConversationID: id,
			Author:         AuthorAI,
			Message:        "",
			ReadStatus:     ReadStatusRead,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		err = db.insertMessage(msg)
		preconditions.Check(err == nil, "failed to create root message: %v", err)

		logger.Info("Created root conversation w/id %s", id)
	}

	return db.GetFullConversation(id)
}

// toCriticMessages converts a slice of MessageRecord to critic.Message.
func toCriticMessages(msgs []*MessageRecord) []critic.Message {
	result := make([]critic.Message, len(msgs))
	for i, msg := range msgs {
		result[i] = critic.Message{
			UUID:      msg.ID,
			Author:    critic.Author(msg.Author),
			Message:   msg.Message,
			CreatedAt: msg.CreatedAt,
			UpdatedAt: msg.UpdatedAt,
			IsUnread:  msg.ReadStatus == ReadStatusUnread,
		}
	}
	return result
}

package messagedb

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/radiospiel/critic/simple-go/logger"
	"github.com/radiospiel/critic/simple-go/preconditions"
	"github.com/radiospiel/critic/src/pkg/critic"
	"github.com/samber/lo"
)

// Author represents who created a message
type Author string

const (
	AuthorHuman Author = "human"
	AuthorAI    Author = "ai"
)

// Status represents the state of a conversation
type Status string

const (
	StatusNew       Status = "new"
	StatusDelivered Status = "delivered"
	StatusResolved  Status = "resolved"
	StatusInformal  Status = "informal"
	StatusArchived  Status = "archived"
)

// ConversationType represents the type of a conversation
type ConversationType string

const (
	ConversationTypeConversation ConversationType = "conversation"
	ConversationTypeExplanation  ConversationType = "explanation"
)

// ReadStatus represents whether an AI message has been shown to the user
type ReadStatus string

const (
	ReadStatusUnread ReadStatus = "unread"
	ReadStatusRead   ReadStatus = "read"
)

// ConversationRecord represents a row in the conversations table.
type ConversationRecord struct {
	ID               string           `db:"id"`
	Status           Status           `db:"status"`
	FilePath         string           `db:"file_path"`
	Lineno           int              `db:"lineno"`
	Commit           string           `db:"sha1"`
	Context          string           `db:"context"`
	ConversationType ConversationType `db:"conversation_type"`
	ReadByAI         bool             `db:"read_by_ai"`
	CreatedAt        time.Time        `db:"created_at"`
	UpdatedAt        time.Time        `db:"updated_at"`
}

func (r *ConversationRecord) toConversation() critic.Conversation {
	return critic.Conversation{
		UUID:             r.ID,
		Status:           convertToCriticStatus(r.Status),
		ConversationType: convertToCriticType(r.ConversationType),
		FilePath:         r.FilePath,
		LineNumber:       r.Lineno,
		CodeVersion:      r.Commit,
		Context:          r.Context,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
		ReadByAI:         r.ReadByAI,
	}
}

// MessageRecord represents a row in the messages table.
type MessageRecord struct {
	ID             string     `db:"id"`
	ConversationID string     `db:"conversation_id"`
	Author         Author     `db:"author"`
	Message        string     `db:"message"`
	ReadStatus     ReadStatus `db:"read_status"`
	CreatedAt      time.Time  `db:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at"`
}

// fileSummaryRow is a scan target for file summary queries.
type fileSummaryRow struct {
	FilePath         string `db:"file_path"`
	UnresolvedCount  int    `db:"unresolved_count"`
	ResolvedCount    int    `db:"resolved_count"`
	ExplanationCount int    `db:"explanation_count"`
	TotalCount       int    `db:"total_count"`
}

func (r fileSummaryRow) toSummary() *critic.FileConversationSummary {
	return &critic.FileConversationSummary{
		FilePath:              r.FilePath,
		TotalCount:            r.TotalCount,
		UnresolvedCount:       r.UnresolvedCount,
		ResolvedCount:         r.ResolvedCount,
		ExplanationCount:      r.ExplanationCount,
		HasUnresolvedComments: r.UnresolvedCount > 0,
		HasResolvedComments:   r.ResolvedCount > 0,
	}
}

// DB manages the SQLite database for messages
type DB struct {
	db     *sqlx.DB
	dbPath string
}

// New creates or opens the message database at the specified git root
func New(gitRoot string) (*DB, error) {
	preconditions.Check(gitRoot != "", "gitRoot must not be empty")

	// Create .critic directory if it doesn't exist
	criticDir := filepath.Join(gitRoot, ".critic")
	if err := os.MkdirAll(criticDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .critic directory: %w", err)
	}

	dbPath := filepath.Join(criticDir, "critic.db")
	logger.Info("Opening message database at: %s", dbPath)

	db, err := sqlx.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys and WAL mode for better concurrency
	_, err = db.Exec(`
		PRAGMA foreign_keys = ON;
		PRAGMA journal_mode = WAL;
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set pragmas: %w", err)
	}

	mdb := &DB{
		db:     db,
		dbPath: dbPath,
	}

	if err := mdb.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	logger.Info("Message database initialized successfully")
	return mdb, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	logger.Info("Closing message database")
	return db.db.Close()
}

// createConversationWithMessage creates a new conversation and its initial message in a transaction.
func (db *DB) createConversationWithMessage(author Author, message, filePath string, lineno int, commit string, context string, convType ConversationType) (*ConversationRecord, *MessageRecord, error) {
	preconditions.Check(author == AuthorHuman || author == AuthorAI, "invalid author: %s", author)
	preconditions.Check(message != "", "message must not be empty")
	preconditions.Check(filePath != "", "filePath must not be empty")
	preconditions.Check(lineno > 0, "lineno must be positive: %d", lineno)

	now := time.Now()
	convID := uuid.Must(uuid.NewV7()).String()

	conv := &ConversationRecord{
		ID:               convID,
		Status:           StatusNew,
		FilePath:         filePath,
		Lineno:           lineno,
		Commit:           commit,
		Context:          context,
		ConversationType: convType,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	msg := &MessageRecord{
		ID:             convID, // first message shares conversation ID
		ConversationID: convID,
		Author:         author,
		Message:        message,
		ReadStatus:     lo.Ternary(author == AuthorAI, ReadStatusUnread, ReadStatusRead),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	tx, err := db.db.Beginx()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	if err := insertConversationTx(tx, conv); err != nil {
		tx.Rollback()
		return nil, nil, fmt.Errorf("failed to insert conversation: %w", err)
	}

	if err := insertMessageTx(tx, msg); err != nil {
		tx.Rollback()
		return nil, nil, fmt.Errorf("failed to insert message: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("failed to commit: %w", err)
	}

	logger.Info("Created %s conversation %s for %s:%d", author, convID, filePath, lineno)
	return conv, msg, nil
}

// CreateReply creates a reply to an existing conversation
func (db *DB) CreateReply(author Author, message, conversationID string) (*MessageRecord, error) {
	preconditions.Check(author == AuthorHuman || author == AuthorAI, "invalid author: %s", author)
	preconditions.Check(message != "", "message must not be empty")
	preconditions.Check(conversationID != "", "conversationID must not be empty")

	// Verify conversation exists
	conv, err := db.getConversation(conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}
	if conv == nil {
		return nil, fmt.Errorf("conversation not found: %s", conversationID)
	}

	now := time.Now()
	msg := &MessageRecord{
		ID:             uuid.Must(uuid.NewV7()).String(),
		ConversationID: conversationID,
		Author:         author,
		Message:        message,
		ReadStatus:     lo.Ternary(author == AuthorAI, ReadStatusUnread, ReadStatusRead),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := db.insertMessage(msg); err != nil {
		return nil, fmt.Errorf("failed to create reply: %w", err)
	}

	logger.Info("Created %s reply %s to conversation %s", author, msg.ID, conversationID)
	return msg, nil
}

// insertConversationTx inserts a conversation record within a transaction.
func insertConversationTx(tx *sqlx.Tx, conv *ConversationRecord) error {
	query := `
		INSERT INTO conversations (
			id, status, file_path, lineno, sha1, context,
			conversation_type, read_by_ai, created_at, updated_at
		) VALUES (
			:id, :status, :file_path, :lineno, :sha1, :context,
			:conversation_type, :read_by_ai, :created_at, :updated_at
		)
	`
	_, err := tx.NamedExec(query, conv)
	return err
}

// insertConversation inserts a conversation record.
func (db *DB) insertConversation(conv *ConversationRecord) error {
	query := `
		INSERT INTO conversations (
			id, status, file_path, lineno, sha1, context,
			conversation_type, read_by_ai, created_at, updated_at
		) VALUES (
			:id, :status, :file_path, :lineno, :sha1, :context,
			:conversation_type, :read_by_ai, :created_at, :updated_at
		)
	`
	_, err := db.db.NamedExec(query, conv)
	return err
}

// insertMessageTx inserts a message record within a transaction.
func insertMessageTx(tx *sqlx.Tx, msg *MessageRecord) error {
	query := `
		INSERT INTO messages (
			id, conversation_id, author, message, read_status, created_at, updated_at
		) VALUES (
			:id, :conversation_id, :author, :message, :read_status, :created_at, :updated_at
		)
	`
	_, err := tx.NamedExec(query, msg)
	return err
}

// insertMessage inserts a message record.
func (db *DB) insertMessage(msg *MessageRecord) error {
	query := `
		INSERT INTO messages (
			id, conversation_id, author, message, read_status, created_at, updated_at
		) VALUES (
			:id, :conversation_id, :author, :message, :read_status, :created_at, :updated_at
		)
	`
	_, err := db.db.NamedExec(query, msg)
	return err
}

// getConversation retrieves a conversation record by ID.
func (db *DB) getConversation(id string) (*ConversationRecord, error) {
	preconditions.Check(id != "", "id must not be empty")

	var conv ConversationRecord
	err := db.db.Get(&conv, `SELECT * FROM conversations WHERE id = ?`, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}
	return &conv, nil
}

// GetMessage retrieves a message by ID
func (db *DB) GetMessage(id string) (*MessageRecord, error) {
	preconditions.Check(id != "", "id must not be empty")

	var msg MessageRecord
	err := db.db.Get(&msg, `SELECT * FROM messages WHERE id = ?`, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}
	return &msg, nil
}

// GetThreadMessages retrieves all messages in a conversation
func (db *DB) GetThreadMessages(conversationID string) ([]*MessageRecord, error) {
	preconditions.Check(conversationID != "", "conversationID must not be empty")

	var messages []*MessageRecord
	err := db.db.Select(&messages, `
		SELECT * FROM messages
		WHERE conversation_id = ?
		ORDER BY created_at ASC
	`, conversationID)
	return messages, err
}

// GetUnresolvedConversations retrieves all unresolved conversations
func (db *DB) GetUnresolvedConversations() ([]*ConversationRecord, error) {
	var conversations []*ConversationRecord
	err := db.db.Select(&conversations, `
		SELECT * FROM conversations
		WHERE status != ?
		ORDER BY file_path, lineno, created_at ASC
	`, string(StatusResolved))
	if err != nil {
		return nil, fmt.Errorf("failed to get unresolved conversations: %w", err)
	}

	logger.Debug("Found %d unresolved conversations", len(conversations))
	return conversations, nil
}

// GetConversationsByFile retrieves all conversations for a specific file
func (db *DB) GetConversationsByFile(filePath string) ([]*ConversationRecord, error) {
	preconditions.Check(filePath != "", "filePath must not be empty")

	var conversations []*ConversationRecord
	err := db.db.Select(&conversations, `
		SELECT * FROM conversations
		WHERE file_path = ?
		ORDER BY lineno, created_at ASC
	`, filePath)
	return conversations, err
}

// MarkConversationAs applies an update to a conversation (resolved, unresolved, read_by_ai)
func (db *DB) MarkConversationAs(conversationID string, update critic.ConversationUpdate) error {
	preconditions.Check(conversationID != "", "conversationID must not be empty")

	var query string
	var args []interface{}

	switch update {
	case critic.ConversationResolved:
		query = `UPDATE conversations SET status = ?, updated_at = ? WHERE id = ?`
		args = []interface{}{string(StatusResolved), time.Now(), conversationID}
	case critic.ConversationUnresolved:
		query = `UPDATE conversations SET status = ?, updated_at = ? WHERE id = ?`
		args = []interface{}{string(StatusNew), time.Now(), conversationID}
	case critic.ConversationArchived:
		query = `UPDATE conversations SET status = ?, updated_at = ? WHERE id = ?`
		args = []interface{}{string(StatusArchived), time.Now(), conversationID}
	case critic.ConversationReadByAI:
		query = `UPDATE conversations SET read_by_ai = 1, updated_at = ? WHERE id = ?`
		args = []interface{}{time.Now(), conversationID}
	default:
		return fmt.Errorf("unknown conversation update: %s", update)
	}

	result, err := db.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to mark conversation as %s: %w", update, err)
	}

	affected, _ := result.RowsAffected()
	logger.Info("Marked conversation %s as %s (affected: %d)", conversationID, update, affected)
	return nil
}

// MarkMessageAs marks a message with a given read status
func (db *DB) MarkMessageAs(messageID string, status critic.MessageReadStatus) error {
	preconditions.Check(messageID != "", "messageID must not be empty")

	var dbStatus string
	switch status {
	case critic.MessageRead:
		dbStatus = string(ReadStatusRead)
	case critic.MessageUnread:
		dbStatus = string(ReadStatusUnread)
	default:
		return fmt.Errorf("unknown message read status: %s", status)
	}

	query := `
		UPDATE messages
		SET read_status = ?, updated_at = ?
		WHERE id = ? AND author = ?
	`

	_, err := db.db.Exec(query, dbStatus, time.Now(), messageID, string(AuthorAI))
	if err != nil {
		return fmt.Errorf("failed to mark message as %s: %w", status, err)
	}

	logger.Debug("Marked AI message %s as %s", messageID, status)
	return nil
}

// GetFilesWithUnreadAIMessages retrieves all file paths that have unread AI messages
func (db *DB) GetFilesWithUnreadAIMessages() ([]string, error) {
	query := `
		SELECT DISTINCT c.file_path
		FROM messages m
		JOIN conversations c ON m.conversation_id = c.id
		WHERE m.author = ? AND m.read_status = ?
		ORDER BY c.file_path
	`

	var files []string
	err := db.db.Select(&files, query, string(AuthorAI), string(ReadStatusUnread))
	return files, err
}

// UpdateConversationStatus updates the status of a conversation
func (db *DB) UpdateConversationStatus(id string, status Status) error {
	preconditions.Check(id != "", "id must not be empty")

	_, err := db.db.Exec(`
		UPDATE conversations SET status = ?, updated_at = ? WHERE id = ?
	`, string(status), time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update conversation status: %w", err)
	}

	logger.Debug("Updated conversation %s status to %s", id, status)
	return nil
}

// UpdateMessage updates the content of an existing message
func (db *DB) UpdateMessage(id string, newMessage string) error {
	preconditions.Check(id != "", "id must not be empty")
	preconditions.Check(newMessage != "", "newMessage must not be empty")

	result, err := db.db.Exec(`
		UPDATE messages SET message = ?, updated_at = ? WHERE id = ?
	`, newMessage, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update message: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("message not found: %s", id)
	}

	logger.Debug("Updated message %s content", id)
	return nil
}

// UpsertConversation creates a new conversation or updates an existing one.
// If existingID is provided and exists, updates the first message's text.
// Otherwise, creates a new conversation.
func (db *DB) UpsertConversation(author Author, message, filePath string, lineno int, commit string, context string, existingID string) (*ConversationRecord, *MessageRecord, error) {
	preconditions.Check(author == AuthorHuman || author == AuthorAI, "invalid author: %s", author)
	preconditions.Check(message != "", "message must not be empty")
	preconditions.Check(filePath != "", "filePath must not be empty")
	preconditions.Check(lineno > 0, "lineno must be positive: %d", lineno)

	if existingID != "" {
		existing, err := db.getConversation(existingID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to check for existing conversation: %w", err)
		}

		if existing != nil {
			// Update the first message's text
			if err := db.UpdateMessage(existingID, message); err != nil {
				return nil, nil, err
			}
			msg, err := db.GetMessage(existingID)
			if err != nil {
				return nil, nil, err
			}
			return existing, msg, nil
		}
	}

	return db.createConversationWithMessage(author, message, filePath, lineno, commit, context, ConversationTypeConversation)
}

// GetMessagesMtime returns the mtime_msec for the messages table from _db_mtime.
// Returns 0 if the table doesn't exist or has no entry.
func (db *DB) GetMessagesMtime() (int64, error) {
	var mtime int64
	err := db.db.Get(&mtime, `SELECT mtime_msec FROM _db_mtime WHERE tablename = 'messages'`)
	if err != nil {
		return 0, nil
	}
	return mtime, nil
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

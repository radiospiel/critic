package messagedb

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
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

// Status represents the state of a message
type Status string

const (
	StatusNew       Status = "new"
	StatusDelivered Status = "delivered"
	StatusResolved  Status = "resolved"
	StatusInformal  Status = "informal"
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

// Message represents a comment/reply in the system
type Message struct {
	ID               string
	Author           Author
	Status           Status
	ReadStatus       ReadStatus
	ReadByAI         bool             // Whether the AI has read this conversation via MCP
	Message          string
	FilePath         string           // File this message is attached to (git-relative path)
	Lineno           int              // Line number in the file
	ConversationID   string           // ID of the root message in the conversation
	Commit           string           // Git commit SHA1
	Context          string           // Code context around the commented line
	ConversationType ConversationType // Type of conversation (conversation or explanation)
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// DB manages the SQLite database for messages
type DB struct {
	db     *sql.DB
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

	db, err := sql.Open("sqlite3", dbPath)
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

// CreateMessage creates a new root message (not a reply)
func (db *DB) CreateMessage(author Author, message, filePath string, lineno int, commit string, context string) (*Message, error) {
	return db.CreateMessageWithType(author, message, filePath, lineno, commit, context, ConversationTypeConversation)
}

// CreateMessageWithType creates a new root message with an explicit conversation type
func (db *DB) CreateMessageWithType(author Author, message, filePath string, lineno int, commit string, context string, convType ConversationType) (*Message, error) {
	preconditions.Check(author == AuthorHuman || author == AuthorAI, "invalid author: %s", author)
	preconditions.Check(message != "", "message must not be empty")
	preconditions.Check(filePath != "", "filePath must not be empty")
	preconditions.Check(lineno > 0, "lineno must be positive: %d", lineno)

	id := uuid.Must(uuid.NewV7()).String()
	msg := &Message{
		ID:               id,
		Author:           author,
		Status:           StatusNew,
		ReadStatus:       lo.Ternary(author == AuthorAI, ReadStatusUnread, ReadStatusRead),
		Message:          message,
		FilePath:         filePath,
		Lineno:           lineno,
		ConversationID:   id, // Root message points to itself
		Commit:           commit,
		Context:          context,
		ConversationType: convType,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	err := db.insertMessage(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	logger.Info("Created %s message %s for %s:%d", author, msg.ID, filePath, lineno)
	return msg, nil
}

// CreateReply creates a reply to an existing conversation
func (db *DB) CreateReply(author Author, message, conversationID string) (*Message, error) {
	preconditions.Check(author == AuthorHuman || author == AuthorAI, "invalid author: %s", author)
	preconditions.Check(message != "", "message must not be empty")
	preconditions.Check(conversationID != "", "conversationID must not be empty")

	// Get root message to inherit file path and line number
	root, err := db.GetMessage(conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation root message: %w", err)
	}
	if root == nil {
		return nil, fmt.Errorf("conversation not found: %s", conversationID)
	}

	msg := &Message{
		ID:             uuid.Must(uuid.NewV7()).String(),
		Author:         author,
		Status:         StatusNew,
		ReadStatus:     lo.Ternary(author == AuthorAI, ReadStatusUnread, ReadStatusRead),
		Message:        message,
		FilePath:       root.FilePath,
		Lineno:         root.Lineno,
		ConversationID: conversationID,
		Commit:         root.Commit,
		Context:        root.Context,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	err = db.insertMessage(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to create reply: %w", err)
	}

	logger.Info("Created %s reply %s to conversation %s", author, msg.ID, conversationID)
	return msg, nil
}

// insertMessage inserts a message into the database
func (db *DB) insertMessage(msg *Message) error {
	query := `
		INSERT INTO messages (
			id, author, status, read_status, read_by_ai, message, file_path, lineno,
			conversation_id, sha1, context, conversation_type, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.db.Exec(query,
		msg.ID,
		string(msg.Author),
		string(msg.Status),
		string(msg.ReadStatus),
		lo.Ternary(msg.ReadByAI, 1, 0),
		msg.Message,
		msg.FilePath,
		msg.Lineno,
		msg.ConversationID,
		msg.Commit,
		msg.Context,
		string(msg.ConversationType),
		msg.CreatedAt,
		msg.UpdatedAt,
	)

	return err
}

// GetMessage retrieves a message by ID
func (db *DB) GetMessage(id string) (*Message, error) {
	preconditions.Check(id != "", "id must not be empty")

	query := `
		SELECT id, author, status, read_status, read_by_ai, message, file_path, lineno,
		       conversation_id, sha1, context, conversation_type, created_at, updated_at
		FROM messages
		WHERE id = ?
	`

	var msg Message
	var readByAI int

	err := db.db.QueryRow(query, id).Scan(
		&msg.ID,
		&msg.Author,
		&msg.Status,
		&msg.ReadStatus,
		&readByAI,
		&msg.Message,
		&msg.FilePath,
		&msg.Lineno,
		&msg.ConversationID,
		&msg.Commit,
		&msg.Context,
		&msg.ConversationType,
		&msg.CreatedAt,
		&msg.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	msg.ReadByAI = readByAI != 0
	return &msg, nil
}

// GetThreadMessages retrieves all messages in a conversation
func (db *DB) GetThreadMessages(conversationID string) ([]*Message, error) {
	preconditions.Check(conversationID != "", "conversationID must not be empty")

	// Get all messages with the same conversation_id
	query := `
		SELECT id, author, status, read_status, read_by_ai, message, file_path, lineno,
		       conversation_id, sha1, context, conversation_type, created_at, updated_at
		FROM messages
		WHERE conversation_id = ?
		ORDER BY created_at ASC
	`

	rows, err := db.db.Query(query, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation messages: %w", err)
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		var msg Message
		var readByAI int

		err := rows.Scan(
			&msg.ID,
			&msg.Author,
			&msg.Status,
			&msg.ReadStatus,
			&readByAI,
			&msg.Message,
			&msg.FilePath,
			&msg.Lineno,
			&msg.ConversationID,
			&msg.Commit,
			&msg.Context,
			&msg.ConversationType,
			&msg.CreatedAt,
			&msg.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		msg.ReadByAI = readByAI != 0
		messages = append(messages, &msg)
	}

	return messages, nil
}

// GetUnresolvedRootMessages retrieves all unresolved root messages (not replies)
func (db *DB) GetUnresolvedRootMessages() ([]*Message, error) {
	query := `
		SELECT id, author, status, read_status, read_by_ai, message, file_path, lineno,
		       conversation_id, sha1, context, conversation_type, created_at, updated_at
		FROM messages
		WHERE status != ? AND id = conversation_id
		ORDER BY file_path, lineno, created_at ASC
	`

	rows, err := db.db.Query(query, string(StatusResolved))
	if err != nil {
		return nil, fmt.Errorf("failed to get unresolved messages: %w", err)
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		var msg Message
		var readByAI int

		err := rows.Scan(
			&msg.ID,
			&msg.Author,
			&msg.Status,
			&msg.ReadStatus,
			&readByAI,
			&msg.Message,
			&msg.FilePath,
			&msg.Lineno,
			&msg.ConversationID,
			&msg.Commit,
			&msg.Context,
			&msg.ConversationType,
			&msg.CreatedAt,
			&msg.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		msg.ReadByAI = readByAI != 0
		messages = append(messages, &msg)
	}

	logger.Debug("Found %d unresolved root messages", len(messages))
	return messages, nil
}

// GetMessagesByFile retrieves all root messages for a specific file
func (db *DB) GetMessagesByFile(filePath string) ([]*Message, error) {
	preconditions.Check(filePath != "", "filePath must not be empty")

	query := `
		SELECT id, author, status, read_status, read_by_ai, message, file_path, lineno,
		       conversation_id, sha1, context, conversation_type, created_at, updated_at
		FROM messages
		WHERE file_path = ? AND id = conversation_id
		ORDER BY lineno, created_at ASC
	`

	rows, err := db.db.Query(query, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages by file: %w", err)
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		var msg Message
		var readByAI int

		err := rows.Scan(
			&msg.ID,
			&msg.Author,
			&msg.Status,
			&msg.ReadStatus,
			&readByAI,
			&msg.Message,
			&msg.FilePath,
			&msg.Lineno,
			&msg.ConversationID,
			&msg.Commit,
			&msg.Context,
			&msg.ConversationType,
			&msg.CreatedAt,
			&msg.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		msg.ReadByAI = readByAI != 0
		messages = append(messages, &msg)
	}

	return messages, nil
}

// MarkConversationAs applies an update to a conversation (resolved, unresolved, read_by_ai)
func (db *DB) MarkConversationAs(conversationID string, update critic.ConversationUpdate) error {
	preconditions.Check(conversationID != "", "conversationID must not be empty")

	var query string
	var args []interface{}

	switch update {
	case critic.ConversationResolved:
		query = `UPDATE messages SET status = ?, updated_at = ? WHERE conversation_id = ?`
		args = []interface{}{string(StatusResolved), time.Now(), conversationID}
	case critic.ConversationUnresolved:
		query = `UPDATE messages SET status = ?, updated_at = ? WHERE conversation_id = ?`
		args = []interface{}{string(StatusNew), time.Now(), conversationID}
	case critic.ConversationReadByAI:
		query = `UPDATE messages SET read_by_ai = 1, updated_at = ? WHERE conversation_id = ?`
		args = []interface{}{time.Now(), conversationID}
	default:
		return fmt.Errorf("unknown conversation update: %s", update)
	}

	result, err := db.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to mark conversation as %s: %w", update, err)
	}

	affected, _ := result.RowsAffected()
	logger.Info("Marked conversation %s (%d messages) as %s", conversationID, affected, update)
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
		SELECT DISTINCT file_path
		FROM messages
		WHERE author = ? AND read_status = ?
		ORDER BY file_path
	`

	rows, err := db.db.Query(query, string(AuthorAI), string(ReadStatusUnread))
	if err != nil {
		return nil, fmt.Errorf("failed to get files with unread messages: %w", err)
	}
	defer rows.Close()

	var files []string
	for rows.Next() {
		var filePath string
		if err := rows.Scan(&filePath); err != nil {
			return nil, fmt.Errorf("failed to scan file path: %w", err)
		}
		files = append(files, filePath)
	}

	return files, nil
}

// UpdateMessageStatus updates the status of a message
func (db *DB) UpdateMessageStatus(id string, status Status) error {
	preconditions.Check(id != "", "id must not be empty")

	query := `
		UPDATE messages
		SET status = ?, updated_at = ?
		WHERE id = ?
	`

	_, err := db.db.Exec(query, string(status), time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update message status: %w", err)
	}

	logger.Debug("Updated message %s status to %s", id, status)
	return nil
}

// UpdateMessage updates the content of an existing message
func (db *DB) UpdateMessage(id string, newMessage string) error {
	preconditions.Check(id != "", "id must not be empty")
	preconditions.Check(newMessage != "", "newMessage must not be empty")

	query := `
		UPDATE messages
		SET message = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := db.db.Exec(query, newMessage, time.Now(), id)
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

// UpsertMessage creates a new message or updates an existing one
// If existingID is provided and exists, updates that message
// Otherwise, creates a new message with a new ID
func (db *DB) UpsertMessage(author Author, message, filePath string, lineno int, commit string, context string, existingID string) (*Message, error) {
	preconditions.Check(author == AuthorHuman || author == AuthorAI, "invalid author: %s", author)
	preconditions.Check(message != "", "message must not be empty")
	preconditions.Check(filePath != "", "filePath must not be empty")
	preconditions.Check(lineno > 0, "lineno must be positive: %d", lineno)

	// If ID provided, try to update existing message
	if existingID != "" {
		existing, err := db.GetMessage(existingID)
		if err != nil {
			return nil, fmt.Errorf("failed to check for existing message: %w", err)
		}

		if existing != nil {
			// Update existing message
			if err := db.UpdateMessage(existingID, message); err != nil {
				return nil, err
			}
			// Return updated message
			return db.GetMessage(existingID)
		}
	}

	// Create new message
	return db.CreateMessage(author, message, filePath, lineno, commit, context)
}

// GetMessagesMtime returns the mtime_msec for the messages table from _db_mtime.
// Returns 0 if the table doesn't exist or has no entry.
func (db *DB) GetMessagesMtime() (int64, error) {
	var mtime int64
	query := `SELECT mtime_msec FROM _db_mtime WHERE tablename = 'messages'`
	err := db.db.QueryRow(query).Scan(&mtime)
	if err != nil {
		// Table might not exist or no entry - return 0
		return 0, nil
	}
	return mtime, nil
}

// WalCheckpoint performs a passive WAL checkpoint to flush pending writes
// to the main database file, making them visible to other connections.
func (db *DB) WalCheckpoint() error {
	_, err := db.db.Exec("PRAGMA wal_checkpoint(PASSIVE)")
	return err
}

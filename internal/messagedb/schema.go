package messagedb

import (
	"database/sql"
	"fmt"

	"git.15b.it/eno/critic/internal/logger"
)

const currentSchemaVersion = "1"

// schema_v1 defines the initial database schema
var schema_v1 = `
	-- Settings table for metadata
	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);

	-- Messages table for threaded comments
	CREATE TABLE IF NOT EXISTS messages (
		id TEXT PRIMARY KEY,
		author TEXT NOT NULL CHECK(author IN ('human', 'ai')),
		status TEXT NOT NULL CHECK(status IN ('new', 'delivered', 'resolved')),
		read_status TEXT NOT NULL DEFAULT 'read' CHECK(read_status IN ('unread', 'read')),
		message TEXT NOT NULL,
		file_path TEXT NOT NULL,
		line_number INTEGER NOT NULL,
		conversation_id TEXT NOT NULL,
		code_version TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		FOREIGN KEY (conversation_id) REFERENCES messages(id) ON DELETE CASCADE
	);

	-- Indexes for performance
	CREATE INDEX IF NOT EXISTS idx_messages_status ON messages(status);
	CREATE INDEX IF NOT EXISTS idx_messages_file_path ON messages(file_path);
	CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id);
	CREATE INDEX IF NOT EXISTS idx_messages_read_status ON messages(read_status) WHERE author = 'ai';
`

// initSchema creates the database schema if it doesn't exist
func (db *DB) initSchema() error {
	// Check if we need to initialize
	version, err := db.getSchemaVersion()
	if err != nil {
		// Settings table doesn't exist, need to initialize
		if err := db.createInitialSchema(); err != nil {
			return err
		}
		logger.Info("Database schema initialized to version %s", currentSchemaVersion)
		return nil
	}

	// Check schema version
	if version != currentSchemaVersion {
		return fmt.Errorf("schema version mismatch: database is at version %s, expected %s", version, currentSchemaVersion)
	}

	logger.Debug("Database schema version %s is current", version)
	return nil
}

// createInitialSchema creates all tables and initial data
func (db *DB) createInitialSchema() error {
	// Create tables using schema v1
	_, err := db.db.Exec(schema_v1)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Set schema version
	if err := db.setSchemaVersion(currentSchemaVersion); err != nil {
		return fmt.Errorf("failed to set schema version: %w", err)
	}

	return nil
}

// getSchemaVersion retrieves the current schema version from settings
func (db *DB) getSchemaVersion() (string, error) {
	var version string
	query := `SELECT value FROM settings WHERE key = ?`
	err := db.db.QueryRow(query, "db_schema").Scan(&version)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("schema version not found")
	}
	if err != nil {
		return "", err
	}
	return version, nil
}

// setSchemaVersion sets the schema version in settings
func (db *DB) setSchemaVersion(version string) error {
	query := `INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)`
	_, err := db.db.Exec(query, "db_schema", version)
	return err
}

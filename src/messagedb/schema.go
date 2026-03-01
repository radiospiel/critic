package messagedb

import (
	"fmt"

	"github.com/radiospiel/critic/simple-go/logger"
)

const currentSchemaVersion = "1"

// schema defines the complete database schema.
// NOTE: Triggers must be created with separate Exec calls because go-sqlite3's
// multi-statement Exec doesn't handle BEGIN...END blocks correctly.
var schema = `
	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS messages (
		id TEXT PRIMARY KEY,
		author TEXT NOT NULL CHECK(author IN ('human', 'ai')),
		status TEXT NOT NULL CHECK(status IN ('new', 'delivered', 'resolved', 'informal')),
		read_status TEXT NOT NULL DEFAULT 'read' CHECK(read_status IN ('unread', 'read')),
		read_by_ai INTEGER NOT NULL DEFAULT 0,
		message TEXT NOT NULL,
		file_path TEXT NOT NULL,
		lineno INTEGER NOT NULL,
		conversation_id TEXT NOT NULL,
		sha1 TEXT NOT NULL,
		context TEXT,
		conversation_type TEXT NOT NULL DEFAULT 'conversation',
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		FOREIGN KEY (conversation_id) REFERENCES messages(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_messages_status ON messages(status);
	CREATE INDEX IF NOT EXISTS idx_messages_file_path ON messages(file_path);
	CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id);
	CREATE INDEX IF NOT EXISTS idx_messages_read_status ON messages(read_status) WHERE author = 'ai';
	CREATE INDEX IF NOT EXISTS idx_messages_read_by_ai ON messages(read_by_ai);

	CREATE TABLE IF NOT EXISTS _db_mtime (
		tablename TEXT PRIMARY KEY,
		mtime_msec INTEGER NOT NULL DEFAULT 0
	);
	INSERT OR IGNORE INTO _db_mtime (tablename, mtime_msec) VALUES ('messages', CAST(unixepoch('subsec') * 1000 AS INTEGER));
`

var schemaTriggers = []string{
	`CREATE TRIGGER IF NOT EXISTS _messages_insert_mtime
	AFTER INSERT ON messages
	BEGIN
		UPDATE _db_mtime SET mtime_msec = CAST(unixepoch('subsec') * 1000 AS INTEGER) WHERE tablename = 'messages';
	END`,
	`CREATE TRIGGER IF NOT EXISTS _messages_update_mtime
	AFTER UPDATE ON messages
	BEGIN
		UPDATE _db_mtime SET mtime_msec = CAST(unixepoch('subsec') * 1000 AS INTEGER) WHERE tablename = 'messages';
	END`,
	`CREATE TRIGGER IF NOT EXISTS _messages_delete_mtime
	AFTER DELETE ON messages
	BEGIN
		UPDATE _db_mtime SET mtime_msec = CAST(unixepoch('subsec') * 1000 AS INTEGER) WHERE tablename = 'messages';
	END`,
}

// initSchema creates the database schema if it doesn't exist
func (db *DB) initSchema() error {
	version, err := db.getSchemaVersion()
	if err != nil {
		// Settings table doesn't exist, need to initialize from scratch
		if err := db.createInitialSchema(); err != nil {
			return err
		}
		logger.Info("Database schema initialized to version %s", currentSchemaVersion)
		return nil
	}

	if version != currentSchemaVersion {
		return fmt.Errorf("schema version mismatch: database is at version %s, expected %s. Please delete .critic/critic.db to start fresh", version, currentSchemaVersion)
	}

	// Ensure triggers exist (they may be missing from older databases)
	if err := db.ensureTriggers(); err != nil {
		return err
	}

	logger.Debug("Database schema version %s is current", currentSchemaVersion)
	return nil
}

// createInitialSchema creates all tables and initial data for a new database
func (db *DB) createInitialSchema() error {
	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	if err := db.ensureTriggers(); err != nil {
		return fmt.Errorf("failed to create triggers: %w", err)
	}

	if err := db.setSchemaVersion(currentSchemaVersion); err != nil {
		return fmt.Errorf("failed to set schema version: %w", err)
	}

	return nil
}

// ensureTriggers creates the mtime triggers if they don't exist.
// Each trigger is executed separately because go-sqlite3 can't handle
// BEGIN...END blocks in multi-statement Exec calls.
func (db *DB) ensureTriggers() error {
	for _, trigger := range schemaTriggers {
		if _, err := db.Exec(trigger); err != nil {
			return fmt.Errorf("failed to create trigger: %w", err)
		}
	}
	return nil
}

// getSchemaVersion retrieves the current schema version from settings
func (db *DB) getSchemaVersion() (string, error) {
	var version string
	err := db.Get(&version, `SELECT value FROM settings WHERE key = ?`, "db_schema")
	if err != nil {
		return "", fmt.Errorf("schema version not found")
	}
	return version, nil
}

// setSchemaVersion sets the schema version in settings
func (db *DB) setSchemaVersion(version string) error {
	query := `INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)`
	_, err := db.Exec(query, "db_schema", version)
	return err
}


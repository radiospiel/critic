package messagedb

import (
	"database/sql"
	"fmt"

	"git.15b.it/eno/critic/simple-go/logger"
)

const currentSchemaVersion = "4"

// schema_v2 defines the v2 database schema with renamed columns and context
var schema_v2 = `
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
		lineno INTEGER NOT NULL,
		conversation_id TEXT NOT NULL,
		sha1 TEXT NOT NULL,
		context TEXT,
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

// schema_v3 migration adds read_by_ai column for tracking AI-read conversations
// This is applied on top of schema_v2
var schema_v3_migration = `
	-- Add read_by_ai column to messages table
	ALTER TABLE messages ADD COLUMN read_by_ai INTEGER NOT NULL DEFAULT 0;

	-- Index for read_by_ai column
	CREATE INDEX IF NOT EXISTS idx_messages_read_by_ai ON messages(read_by_ai);
`

// schema_v4 migration adds _db_mtime table and triggers for change detection
// This is applied on top of schema_v3
var schema_v4_migration = `
	-- Modification time tracking table for change detection (single row)
	CREATE TABLE IF NOT EXISTS _db_mtime (
		mtime INTEGER NOT NULL DEFAULT 0
	);

	-- Initialize mtime with current timestamp in milliseconds
	INSERT INTO _db_mtime (mtime) SELECT CAST(unixepoch('subsec') * 1000 AS INTEGER) WHERE NOT EXISTS (SELECT 1 FROM _db_mtime);

	-- Triggers to update mtime on messages table changes
	CREATE TRIGGER IF NOT EXISTS _messages_insert_mtime
	AFTER INSERT ON messages
	BEGIN
		UPDATE _db_mtime SET mtime = CAST(unixepoch('subsec') * 1000 AS INTEGER);
	END;

	CREATE TRIGGER IF NOT EXISTS _messages_update_mtime
	AFTER UPDATE ON messages
	BEGIN
		UPDATE _db_mtime SET mtime = CAST(unixepoch('subsec') * 1000 AS INTEGER);
	END;

	CREATE TRIGGER IF NOT EXISTS _messages_delete_mtime
	AFTER DELETE ON messages
	BEGIN
		UPDATE _db_mtime SET mtime = CAST(unixepoch('subsec') * 1000 AS INTEGER);
	END;
`

// initSchema creates the database schema if it doesn't exist
func (db *DB) initSchema() error {
	// Check if we need to initialize
	version, err := db.getSchemaVersion()
	if err != nil {
		// Settings table doesn't exist, need to initialize from scratch
		if err := db.createInitialSchema(); err != nil {
			return err
		}
		logger.Info("Database schema initialized to version %s", currentSchemaVersion)
		return nil
	}

	// Apply migrations if needed
	if version != currentSchemaVersion {
		if err := db.migrateSchema(version); err != nil {
			return err
		}
	}

	logger.Debug("Database schema version %s is current", currentSchemaVersion)
	return nil
}

// createInitialSchema creates all tables and initial data for a new database
func (db *DB) createInitialSchema() error {
	// Create tables using schema v2 (base schema)
	_, err := db.db.Exec(schema_v2)
	if err != nil {
		return fmt.Errorf("failed to create schema v2: %w", err)
	}

	// Apply v3 migration (adds read_by_ai column)
	_, err = db.db.Exec(schema_v3_migration)
	if err != nil {
		return fmt.Errorf("failed to apply schema v3 migration: %w", err)
	}

	// Apply v4 migration (adds _db_mtime table and triggers)
	_, err = db.db.Exec(schema_v4_migration)
	if err != nil {
		return fmt.Errorf("failed to apply schema v4 migration: %w", err)
	}

	// Set schema version
	if err := db.setSchemaVersion(currentSchemaVersion); err != nil {
		return fmt.Errorf("failed to set schema version: %w", err)
	}

	return nil
}

// migrateSchema applies migrations to bring the database up to current version
func (db *DB) migrateSchema(fromVersion string) error {
	switch fromVersion {
	case "2":
		// Migrate from v2 to v3
		logger.Info("Migrating database schema from v2 to v3")
		_, err := db.db.Exec(schema_v3_migration)
		if err != nil {
			return fmt.Errorf("failed to apply schema v3 migration: %w", err)
		}
		if err := db.setSchemaVersion("3"); err != nil {
			return fmt.Errorf("failed to set schema version: %w", err)
		}
		logger.Info("Database schema migrated to version 3")
		// Continue to apply v4 migration
		fallthrough
	case "3":
		// Migrate from v3 to v4
		logger.Info("Migrating database schema from v3 to v4")
		_, err := db.db.Exec(schema_v4_migration)
		if err != nil {
			return fmt.Errorf("failed to apply schema v4 migration: %w", err)
		}
		if err := db.setSchemaVersion("4"); err != nil {
			return fmt.Errorf("failed to set schema version: %w", err)
		}
		logger.Info("Database schema migrated to version 4")
		return nil
	default:
		return fmt.Errorf("schema version mismatch: database is at version %s, expected %s. Please delete .critic.db to start fresh", fromVersion, currentSchemaVersion)
	}
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

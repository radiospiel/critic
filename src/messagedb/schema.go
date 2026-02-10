package messagedb

import (
	"database/sql"
	"fmt"

	"github.com/radiospiel/critic/simple-go/logger"
)

const currentSchemaVersion = "5"

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
		status TEXT NOT NULL CHECK(status IN ('new', 'delivered', 'resolved', 'informal')),
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

// schema_v5 migration adds conversation_type column and updates the status CHECK constraint
// to allow 'informal'. SQLite doesn't support ALTER CONSTRAINT, so we recreate the table.
var schema_v5_migration = `
	-- Back up existing messages
	CREATE TABLE messages_backup AS SELECT * FROM messages;

	-- Drop the old table and recreate with updated constraints
	DROP TABLE messages;

	CREATE TABLE messages (
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

	-- Copy data back
	INSERT INTO messages (id, author, status, read_status, read_by_ai, message, file_path, lineno, conversation_id, sha1, context, created_at, updated_at)
	SELECT id, author, status, read_status, read_by_ai, message, file_path, lineno, conversation_id, sha1, context, created_at, updated_at
	FROM messages_backup;

	DROP TABLE messages_backup;

	-- Recreate indexes
	CREATE INDEX IF NOT EXISTS idx_messages_status ON messages(status);
	CREATE INDEX IF NOT EXISTS idx_messages_file_path ON messages(file_path);
	CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id);
	CREATE INDEX IF NOT EXISTS idx_messages_read_status ON messages(read_status) WHERE author = 'ai';
	CREATE INDEX IF NOT EXISTS idx_messages_read_by_ai ON messages(read_by_ai);
`

// schema_v4 migration adds _db_mtime table and triggers for change detection
// This is applied on top of schema_v3
var schema_v4_migration = `
	-- Modification time tracking table for change detection (one row per tracked table)
	CREATE TABLE IF NOT EXISTS _db_mtime (
		tablename TEXT PRIMARY KEY,
		mtime_msec INTEGER NOT NULL DEFAULT 0
	);

	-- Initialize mtime for messages table
	INSERT OR IGNORE INTO _db_mtime (tablename, mtime_msec) VALUES ('messages', CAST(unixepoch('subsec') * 1000 AS INTEGER));

	-- Triggers to update mtime on messages table changes
	CREATE TRIGGER IF NOT EXISTS _messages_insert_mtime
	AFTER INSERT ON messages
	BEGIN
		UPDATE _db_mtime SET mtime_msec = CAST(unixepoch('subsec') * 1000 AS INTEGER) WHERE tablename = 'messages';
	END;

	CREATE TRIGGER IF NOT EXISTS _messages_update_mtime
	AFTER UPDATE ON messages
	BEGIN
		UPDATE _db_mtime SET mtime_msec = CAST(unixepoch('subsec') * 1000 AS INTEGER) WHERE tablename = 'messages';
	END;

	CREATE TRIGGER IF NOT EXISTS _messages_delete_mtime
	AFTER DELETE ON messages
	BEGIN
		UPDATE _db_mtime SET mtime_msec = CAST(unixepoch('subsec') * 1000 AS INTEGER) WHERE tablename = 'messages';
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

	// Apply v5 migration (adds conversation_type column)
	_, err = db.db.Exec(schema_v5_migration)
	if err != nil {
		return fmt.Errorf("failed to apply schema v5 migration: %w", err)
	}

	// Set schema version
	if err := db.setSchemaVersion(currentSchemaVersion); err != nil {
		return fmt.Errorf("failed to set schema version: %w", err)
	}

	return nil
}

// migration defines a single schema migration step
type migration struct {
	fromVersion string
	toVersion   string
	sql         string
}

// migrations defines the ordered list of schema migrations
var migrations = []migration{
	{"2", "3", schema_v3_migration},
	{"3", "4", schema_v4_migration},
	{"4", "5", schema_v5_migration},
}

// migrateSchema applies migrations to bring the database up to current version
func (db *DB) migrateSchema(fromVersion string) error {
	currentVersion := fromVersion

	for currentVersion != currentSchemaVersion {
		// Find the migration for the current version
		var mig *migration
		for i := range migrations {
			if migrations[i].fromVersion == currentVersion {
				mig = &migrations[i]
				break
			}
		}

		if mig == nil {
			return fmt.Errorf("schema version mismatch: database is at version %s, expected %s. Please delete .critic/critic.db to start fresh", currentVersion, currentSchemaVersion)
		}

		// Run migration in a transaction
		if err := db.applyMigration(mig); err != nil {
			return err
		}

		currentVersion = mig.toVersion
	}

	return nil
}

// applyMigration applies a single migration step within a transaction
func (db *DB) applyMigration(mig *migration) error {
	logger.Info("Migrating database schema from v%s to v%s", mig.fromVersion, mig.toVersion)

	tx, err := db.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction for migration v%s→v%s: %w", mig.fromVersion, mig.toVersion, err)
	}

	_, err = tx.Exec(mig.sql)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to apply schema v%s migration: %w", mig.toVersion, err)
	}

	_, err = tx.Exec(`INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)`, "db_schema", mig.toVersion)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to set schema version to %s: %w", mig.toVersion, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration v%s→v%s: %w", mig.fromVersion, mig.toVersion, err)
	}

	logger.Info("Database schema migrated to version %s", mig.toVersion)
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

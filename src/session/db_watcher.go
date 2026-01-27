package session

import (
	"database/sql"
	"path/filepath"
	"sync"
	"time"

	"github.com/radiospiel/critic/simple-go/logger"
	_ "github.com/mattn/go-sqlite3"
)

// DBWatcher watches the messaging database for changes using SQLite triggers
type DBWatcher struct {
	dbPath       string
	db           *sql.DB
	onChange     func()
	stopChan     chan struct{}
	mu           sync.Mutex
	running      bool
	pollInterval time.Duration
	lastMtime    int64
}

// NewDBWatcher creates a new database watcher
func NewDBWatcher(gitRoot string, onChange func()) (*DBWatcher, error) {
	dbPath := filepath.Join(gitRoot, ".critic.db")

	return &DBWatcher{
		dbPath:       dbPath,
		onChange:     onChange,
		stopChan:     make(chan struct{}),
		pollInterval: 500 * time.Millisecond,
	}, nil
}

// SetPollInterval sets the polling interval
func (w *DBWatcher) SetPollInterval(interval time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.pollInterval = interval
}

// Start starts watching the database
// The database must have been initialized with the schema (which creates the _db_mtime table)
func (w *DBWatcher) Start() error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return nil
	}
	w.mu.Unlock()

	// Open database connection
	db, err := sql.Open("sqlite3", w.dbPath)
	if err != nil {
		return err
	}

	// Enable WAL mode for better concurrency
	_, err = db.Exec("PRAGMA journal_mode = WAL")
	if err != nil {
		db.Close()
		return err
	}

	// Get initial mtime (requires _db_mtime table from schema initialization)
	mtime, err := w.getMtime(db)
	if err != nil {
		db.Close()
		return err
	}

	w.mu.Lock()
	w.db = db
	w.lastMtime = mtime
	w.running = true
	w.mu.Unlock()

	go w.pollLoop()
	logger.Info("DBWatcher: Started watching %s (mtime=%d)", w.dbPath, mtime)
	return nil
}

// getMtime reads the current mtime for the messages table from the database
func (w *DBWatcher) getMtime(db *sql.DB) (int64, error) {
	var mtime int64
	err := db.QueryRow("SELECT mtime_msec FROM _db_mtime WHERE tablename = 'messages'").Scan(&mtime)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return mtime, err
}

// Stop stops the watcher
func (w *DBWatcher) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.running = false
	w.mu.Unlock()

	close(w.stopChan)

	w.mu.Lock()
	if w.db != nil {
		w.db.Close()
		w.db = nil
	}
	w.mu.Unlock()

	logger.Info("DBWatcher: Stopped")
}

// pollLoop periodically checks for version changes
func (w *DBWatcher) pollLoop() {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.checkForChanges()
		case <-w.stopChan:
			return
		}
	}
}

// checkForChanges checks if the database mtime has changed
func (w *DBWatcher) checkForChanges() {
	w.mu.Lock()
	db := w.db
	lastMtime := w.lastMtime
	w.mu.Unlock()

	if db == nil {
		return
	}

	mtime, err := w.getMtime(db)
	if err != nil {
		logger.Error("DBWatcher: Failed to get mtime: %v", err)
		return
	}

	if mtime != lastMtime {
		w.mu.Lock()
		w.lastMtime = mtime
		callback := w.onChange
		w.mu.Unlock()

		logger.Info("DBWatcher: mtime changed from %d to %d", lastMtime, mtime)

		if callback != nil {
			callback()
		}
	}
}

// IsRunning returns whether the watcher is running
func (w *DBWatcher) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.running
}

// SetDebounceMs is a no-op for compatibility (polling interval is used instead)
// Deprecated: Use SetPollInterval instead
func (w *DBWatcher) SetDebounceMs(ms int) {
	w.SetPollInterval(time.Duration(ms) * time.Millisecond)
}

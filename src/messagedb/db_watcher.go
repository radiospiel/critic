package messagedb

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/radiospiel/critic/simple-go/logger"
	"github.com/radiospiel/critic/simple-go/utils"
)

// DBWatcher watches the critic database for changes and notifies when the
// messages table mtime changes.
type DBWatcher struct {
	fileWatcher *utils.FileWatcher
	db          *DB
	gitRoot     string

	lastMtime   int64
	lastMtimeMu sync.Mutex

	changesChan chan struct{}
	stopChan    chan struct{}
}

// NewDBWatcher creates a watcher that monitors the critic database for message changes.
// It watches the .critic/critic.db file and compares the _db_mtime to detect actual data changes.
func NewDBWatcher(gitRoot string, debounceMs int) (*DBWatcher, error) {
	dbPath := filepath.Join(gitRoot, ".critic", "critic.db")
	logger.Info("NewDBWatcher: Creating watcher for %s", dbPath)

	// Create file watcher for the database file
	fileWatcher, err := utils.NewFileWatcher(dbPath, debounceMs)
	if err != nil {
		return nil, err
	}

	// Open database connection to query mtime
	db, err := New(gitRoot)
	if err != nil {
		fileWatcher.Close()
		return nil, err
	}

	// Get initial mtime
	initialMtime, err := db.GetMessagesMtime()
	if err != nil {
		db.Close()
		fileWatcher.Close()
		return nil, err
	}

	dw := &DBWatcher{
		fileWatcher: fileWatcher,
		db:          db,
		gitRoot:     gitRoot,
		lastMtime:   initialMtime,
		changesChan: make(chan struct{}, 10),
		stopChan:    make(chan struct{}),
	}

	// Start the mtime checking loop
	go dw.checkLoop()

	logger.Info("NewDBWatcher: Started watching database, initial mtime=%d", initialMtime)
	return dw, nil
}

// checkLoop listens for file changes and checks if mtime has actually changed
func (dw *DBWatcher) checkLoop() {
	for {
		select {
		case <-dw.fileWatcher.Changes():
			dw.checkMtime()
		case <-dw.stopChan:
			logger.Info("DBWatcher: Stop signal received")
			return
		}
	}
}

// checkMtime queries the current mtime and sends notification if it changed
func (dw *DBWatcher) checkMtime() {
	currentMtime, err := dw.db.GetMessagesMtime()
	if err != nil {
		logger.Error("DBWatcher: Failed to get mtime: %v", err)
		return
	}

	dw.lastMtimeMu.Lock()
	lastMtime := dw.lastMtime
	if currentMtime != lastMtime {
		dw.lastMtime = currentMtime
		dw.lastMtimeMu.Unlock()

		logger.Info("DBWatcher: Messages mtime changed from %d to %d", lastMtime, currentMtime)

		// Checkpoint WAL to flush pending writes to main database
		if err := dw.db.WalCheckpoint(); err != nil {
			logger.Error("DBWatcher: WAL checkpoint failed: %v", err)
		}

		// Additional delay to ensure other connections see the changes
		time.Sleep(100 * time.Millisecond)

		select {
		case dw.changesChan <- struct{}{}:
			logger.Info("DBWatcher: Change notification sent")
		default:
			logger.Debug("DBWatcher: Notification channel full, dropping")
		}
	} else {
		dw.lastMtimeMu.Unlock()
		logger.Debug("DBWatcher: File changed but mtime unchanged (%d)", currentMtime)
	}
}

// Changes returns a channel that receives notifications when the messages table changes.
func (dw *DBWatcher) Changes() <-chan struct{} {
	return dw.changesChan
}

// LastChangeTime returns the last recorded mtime in milliseconds.
func (dw *DBWatcher) LastChangeTime() int64 {
	dw.lastMtimeMu.Lock()
	defer dw.lastMtimeMu.Unlock()
	return dw.lastMtime
}

// Close stops the watcher and releases resources.
func (dw *DBWatcher) Close() error {
	logger.Info("DBWatcher: Closing")

	// Stop the check loop
	close(dw.stopChan)

	// Small delay to let goroutine exit
	time.Sleep(10 * time.Millisecond)

	// Close file watcher
	if err := dw.fileWatcher.Close(); err != nil {
		logger.Error("DBWatcher: Failed to close file watcher: %v", err)
	}

	// Close database connection
	return dw.db.Close()
}

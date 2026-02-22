package messagedb

import (
	"sync"
	"time"

	"github.com/radiospiel/critic/simple-go/logger"
)

// DBWatcher watches the critic database for changes by polling the _db_mtime table.
// It opens a fresh connection for each poll to ensure it sees the latest data.
type DBWatcher struct {
	pollInterval time.Duration

	lastMtime   int64
	lastMtimeMu sync.Mutex

	changesChan chan struct{}
	stopChan    chan struct{}

	db *DB
}

const MTIME_UNINITIALIZED int64 = -1

// NewDBWatcher creates a watcher that polls the database for message changes.
// pollIntervalMs specifies how often to check for changes (in milliseconds).
// It uses the provided DB instance to ensure it sees writes from the same connection pool.
func NewDBWatcher(db *DB, pollIntervalMs int) (*DBWatcher, error) {
	logger.Info("NewDBWatcher: Creating polling watcher with interval=%dms", pollIntervalMs)

	dw := &DBWatcher{
		pollInterval: time.Duration(pollIntervalMs) * time.Millisecond,
		changesChan:  make(chan struct{}, 10),
		stopChan:     make(chan struct{}),
		lastMtime:    MTIME_UNINITIALIZED,
		db:           db,
	}

	dw.checkMtime()

	// Start the polling loop
	go dw.pollLoop()

	logger.Info("NewDBWatcher: Started polling database, initial mtime=%d", dw.lastMtime)
	return dw, nil
}

// pollLoop periodically checks the mtime using fresh connections
func (dw *DBWatcher) pollLoop() {
	ticker := time.NewTicker(dw.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			dw.checkMtime()
		case <-dw.stopChan:
			logger.Info("DBWatcher: Stop signal received")
			return
		}
	}
}

// checkMtime queries the current mtime with a fresh connection and sends notification if changed
func (dw *DBWatcher) checkMtime() {
	// currentMtime, err := getMtimeWithFreshConnection(dw.gitRoot)
	currentMtime, err := dw.db.GetMessagesMtime()

	if err != nil {
		logger.Error("DBWatcher: Failed to get mtime: %v", err)
		return
	}

	dw.lastMtimeMu.Lock()
	prevMtime := dw.lastMtime
	dw.lastMtime = currentMtime
	dw.lastMtimeMu.Unlock()

	if prevMtime != currentMtime && prevMtime != MTIME_UNINITIALIZED {
		logger.Info("DBWatcher: Messages mtime changed from %d to %d", prevMtime, currentMtime)
		select {
		case dw.changesChan <- struct{}{}:
			logger.Debug("DBWatcher: Change notification sent")
		default:
			logger.Warn("DBWatcher: Notification channel full, dropping")
		}
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

// Close stops the watcher.
func (dw *DBWatcher) Close() error {
	logger.Info("DBWatcher: Closing")
	close(dw.stopChan)
	return nil
}

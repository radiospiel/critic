package messagedb

import (
	"sync"
	"time"

	"github.com/radiospiel/critic/simple-go/logger"
)

// DBWatcher watches the critic database for changes by polling the _db_mtime table.
// It opens a fresh connection for each poll to ensure it sees the latest data.
type DBWatcher struct {
	gitRoot     string
	pollInterval time.Duration

	lastMtime   int64
	lastMtimeMu sync.Mutex

	changesChan chan struct{}
	stopChan    chan struct{}
}

// NewDBWatcher creates a watcher that polls the database for message changes.
// pollIntervalMs specifies how often to check for changes (in milliseconds).
func NewDBWatcher(gitRoot string, pollIntervalMs int) (*DBWatcher, error) {
	logger.Info("NewDBWatcher: Creating polling watcher for %s with interval=%dms", gitRoot, pollIntervalMs)

	// Get initial mtime with a fresh connection
	initialMtime, err := getMtimeWithFreshConnection(gitRoot)
	if err != nil {
		return nil, err
	}

	dw := &DBWatcher{
		gitRoot:      gitRoot,
		pollInterval: time.Duration(pollIntervalMs) * time.Millisecond,
		lastMtime:    initialMtime,
		changesChan:  make(chan struct{}, 10),
		stopChan:     make(chan struct{}),
	}

	// Start the polling loop
	go dw.pollLoop()

	logger.Info("NewDBWatcher: Started polling database, initial mtime=%d", initialMtime)
	return dw, nil
}

// getMtimeWithFreshConnection opens a new connection, queries mtime, and closes it
func getMtimeWithFreshConnection(gitRoot string) (int64, error) {
	db, err := New(gitRoot)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	return db.GetMessagesMtime()
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
	currentMtime, err := getMtimeWithFreshConnection(dw.gitRoot)
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

		select {
		case dw.changesChan <- struct{}{}:
			logger.Info("DBWatcher: Change notification sent")
		default:
			logger.Debug("DBWatcher: Notification channel full, dropping")
		}
	} else {
		dw.lastMtimeMu.Unlock()
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

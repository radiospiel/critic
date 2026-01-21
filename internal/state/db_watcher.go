package state

import (
	"path/filepath"
	"sync"
	"time"

	"git.15b.it/eno/critic/simple-go/logger"
	"github.com/fsnotify/fsnotify"
)

// DBWatcher watches the messaging database for changes
type DBWatcher struct {
	dbPath    string
	watcher   *fsnotify.Watcher
	onChange  func()
	stopChan  chan struct{}
	mu        sync.Mutex
	running   bool
	debounceMs int

	// Debouncing
	lastEvent time.Time
	timer     *time.Timer
}

// NewDBWatcher creates a new database watcher
func NewDBWatcher(gitRoot string, onChange func()) (*DBWatcher, error) {
	dbPath := filepath.Join(gitRoot, ".critic.db")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &DBWatcher{
		dbPath:     dbPath,
		watcher:    watcher,
		onChange:   onChange,
		stopChan:   make(chan struct{}),
		debounceMs: 100, // 100ms debounce
	}, nil
}

// SetDebounceMs sets the debounce duration in milliseconds
func (w *DBWatcher) SetDebounceMs(ms int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.debounceMs = ms
}

// Start starts watching the database file
func (w *DBWatcher) Start() error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return nil
	}
	w.running = true
	w.mu.Unlock()

	// Watch the database directory (to catch file creation)
	dir := filepath.Dir(w.dbPath)
	if err := w.watcher.Add(dir); err != nil {
		logger.Warn("DBWatcher: Failed to watch directory %s: %v", dir, err)
		// Try watching the file directly if it exists
		if err := w.watcher.Add(w.dbPath); err != nil {
			return err
		}
	}

	// Also watch WAL and SHM files if they exist
	walPath := w.dbPath + "-wal"
	shmPath := w.dbPath + "-shm"
	_ = w.watcher.Add(walPath)
	_ = w.watcher.Add(shmPath)

	go w.eventLoop()
	logger.Info("DBWatcher: Started watching %s", w.dbPath)
	return nil
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

	// Stop any pending timer
	if w.timer != nil {
		w.timer.Stop()
	}

	_ = w.watcher.Close()
	logger.Info("DBWatcher: Stopped")
}

// eventLoop processes file system events
func (w *DBWatcher) eventLoop() {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			w.handleEvent(event)

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			logger.Error("DBWatcher: Error: %v", err)

		case <-w.stopChan:
			return
		}
	}
}

// handleEvent processes a single file event
func (w *DBWatcher) handleEvent(event fsnotify.Event) {
	// Only care about the database file and its WAL
	base := filepath.Base(event.Name)
	if base != ".critic.db" && base != ".critic.db-wal" {
		return
	}

	// Only care about write events
	if event.Op&fsnotify.Write != fsnotify.Write &&
		event.Op&fsnotify.Create != fsnotify.Create {
		return
	}

	logger.Debug("DBWatcher: Detected change in %s", event.Name)

	w.mu.Lock()
	defer w.mu.Unlock()

	// Reset debounce timer
	if w.timer != nil {
		w.timer.Stop()
	}

	w.timer = time.AfterFunc(time.Duration(w.debounceMs)*time.Millisecond, func() {
		w.mu.Lock()
		callback := w.onChange
		w.mu.Unlock()

		if callback != nil {
			logger.Info("DBWatcher: Triggering onChange callback")
			callback()
		}
	})
}

// IsRunning returns whether the watcher is running
func (w *DBWatcher) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.running
}

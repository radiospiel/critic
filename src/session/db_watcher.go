package session

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/radiospiel/critic/simple-go/logger"
)

// DBWatcher watches the .critic directory for changes using fsnotify
type DBWatcher struct {
	criticDir  string
	watcher    *fsnotify.Watcher
	onChange   func()
	stopChan   chan struct{}
	mu         sync.Mutex
	running    bool
	debounceMs int

	// Debouncing state
	debounceTimer *time.Timer
	debounceMu    sync.Mutex
}

// NewDBWatcher creates a new database watcher for the .critic directory
func NewDBWatcher(gitRoot string, onChange func()) (*DBWatcher, error) {
	criticDir := filepath.Join(gitRoot, ".critic")

	return &DBWatcher{
		criticDir:  criticDir,
		onChange:   onChange,
		stopChan:   make(chan struct{}),
		debounceMs: 100,
	}, nil
}

// Start starts watching the .critic directory
func (w *DBWatcher) Start() error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return nil
	}
	w.mu.Unlock()

	// Create .critic directory if it doesn't exist
	if err := os.MkdirAll(w.criticDir, 0755); err != nil {
		return err
	}

	// Create fsnotify watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	// Watch the .critic directory
	if err := watcher.Add(w.criticDir); err != nil {
		watcher.Close()
		return err
	}

	w.mu.Lock()
	w.watcher = watcher
	w.running = true
	w.mu.Unlock()

	go w.eventLoop()
	logger.Info("DBWatcher: Started watching %s", w.criticDir)
	return nil
}

// eventLoop handles fsnotify events
func (w *DBWatcher) eventLoop() {
	// Get local reference to watcher channels (safe since watcher is set before goroutine starts)
	w.mu.Lock()
	watcher := w.watcher
	w.mu.Unlock()

	if watcher == nil {
		return
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				logger.Info("DBWatcher: Events channel closed")
				return
			}

			// Only care about write, create, remove events
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) != 0 {
				logger.Debug("DBWatcher: Event: %s %s", event.Op, event.Name)
				w.scheduleNotification()
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				logger.Info("DBWatcher: Errors channel closed")
				return
			}
			logger.Error("DBWatcher: Error: %v", err)

		case <-w.stopChan:
			logger.Info("DBWatcher: Stop signal received")
			return
		}
	}
}

// scheduleNotification schedules a debounced change notification
func (w *DBWatcher) scheduleNotification() {
	w.debounceMu.Lock()
	defer w.debounceMu.Unlock()

	// Cancel existing timer if any
	if w.debounceTimer != nil {
		w.debounceTimer.Stop()
	}

	// Schedule new notification
	w.debounceTimer = time.AfterFunc(time.Duration(w.debounceMs)*time.Millisecond, func() {
		w.mu.Lock()
		callback := w.onChange
		w.mu.Unlock()

		if callback != nil {
			logger.Info("DBWatcher: Change detected in .critic directory")
			callback()
		}
	})
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
	w.debounceMu.Lock()
	if w.debounceTimer != nil {
		w.debounceTimer.Stop()
	}
	w.debounceMu.Unlock()

	w.mu.Lock()
	if w.watcher != nil {
		w.watcher.Close()
		w.watcher = nil
	}
	w.mu.Unlock()

	logger.Info("DBWatcher: Stopped")
}

// IsRunning returns whether the watcher is running
func (w *DBWatcher) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.running
}

// SetDebounceMs sets the debounce interval in milliseconds
func (w *DBWatcher) SetDebounceMs(ms int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.debounceMs = ms
}

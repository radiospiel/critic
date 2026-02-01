package git

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/radiospiel/critic/simple-go/logger"
)

// GitWatcher watches the .git directory for changes to detect git operations
// like commits, checkouts, pulls, etc. It debounces rapid changes and notifies
// via a channel when the git state has changed.
type GitWatcher struct {
	watcher     *fsnotify.Watcher
	debounceMs  int
	changesChan chan struct{}
	lastChange  atomic.Int64 // Unix milliseconds

	// Debouncing state
	debounceTimer *time.Timer
	debounceMu    sync.Mutex

	// Lifecycle
	stopChan chan struct{}
}

// NewGitWatcher creates a watcher that monitors the .git directory for changes.
// debounceMs specifies how long to wait after the last change before emitting a notification.
func NewGitWatcher(gitDir string, debounceMs int) (*GitWatcher, error) {
	logger.Info("NewGitWatcher: Creating watcher for %s with debounce=%dms", gitDir, debounceMs)

	w, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Error("NewGitWatcher: Failed to create fsnotify watcher: %v", err)
		return nil, err
	}

	gw := &GitWatcher{
		watcher:     w,
		debounceMs:  debounceMs,
		changesChan: make(chan struct{}, 10),
		stopChan:    make(chan struct{}),
	}

	// Set initial last change time
	gw.lastChange.Store(time.Now().UnixMilli())

	// Watch the git directory (non-recursive - just the .git directory itself)
	if err := w.Add(gitDir); err != nil {
		w.Close()
		return nil, err
	}

	// Start the event loop
	go gw.eventLoop()

	logger.Info("NewGitWatcher: Started watching %s", gitDir)
	return gw, nil
}

// eventLoop handles fsnotify events and debounces them
func (gw *GitWatcher) eventLoop() {
	logger.Info("GitWatcher: Event loop started")
	for {
		select {
		case event, ok := <-gw.watcher.Events:
			if !ok {
				logger.Info("GitWatcher: Events channel closed")
				return
			}

			// Only care about write, create, remove, rename events
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
				logger.Debug("GitWatcher: Event: %s %s", event.Op, event.Name)
				gw.scheduleNotification()
			}

		case err, ok := <-gw.watcher.Errors:
			if !ok {
				logger.Info("GitWatcher: Errors channel closed")
				return
			}
			logger.Error("GitWatcher: Error: %v", err)

		case <-gw.stopChan:
			logger.Info("GitWatcher: Stop signal received")
			return
		}
	}
}

// scheduleNotification schedules a debounced change notification
func (gw *GitWatcher) scheduleNotification() {
	gw.debounceMu.Lock()
	defer gw.debounceMu.Unlock()

	// Cancel existing timer if any
	if gw.debounceTimer != nil {
		gw.debounceTimer.Stop()
	}

	// Schedule new notification
	gw.debounceTimer = time.AfterFunc(time.Duration(gw.debounceMs)*time.Millisecond, func() {
		// Update last change time
		gw.lastChange.Store(time.Now().UnixMilli())

		// Send notification
		select {
		case gw.changesChan <- struct{}{}:
			logger.Info("GitWatcher: Change notification sent")
		default:
			logger.Debug("GitWatcher: Notification channel full, dropping")
		}
	})
}

// Changes returns a channel that receives notifications when the git state changes.
// Each notification is a struct{} - the actual change details are not provided.
func (gw *GitWatcher) Changes() <-chan struct{} {
	return gw.changesChan
}

// LastChangeTime returns the timestamp of the last detected change in Unix milliseconds.
func (gw *GitWatcher) LastChangeTime() int64 {
	return gw.lastChange.Load()
}

// Close stops the watcher and releases resources.
func (gw *GitWatcher) Close() error {
	logger.Info("GitWatcher: Closing")

	// Stop the event loop
	close(gw.stopChan)

	// Stop any pending timer
	gw.debounceMu.Lock()
	if gw.debounceTimer != nil {
		gw.debounceTimer.Stop()
	}
	gw.debounceMu.Unlock()

	// Close the fsnotify watcher
	return gw.watcher.Close()
}

package git

import (
	"os"
	"path/filepath"
	"time"

	"git.15b.it/eno/critic/internal/logger"
	"github.com/fsnotify/fsnotify"
)

// Watcher watches files for changes
type Watcher struct {
	watcher     *fsnotify.Watcher
	debouncer   *time.Timer
	debounceMs  int
	changesChan chan struct{}
}

// NewWatcher creates a new file watcher
func NewWatcher(debounceMs int) (*Watcher, error) {
	logger.Info("NewWatcher: Creating watcher with debounce=%dms", debounceMs)
	w, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Error("NewWatcher: Failed to create fsnotify watcher: %v", err)
		return nil, err
	}

	watcher := &Watcher{
		watcher:     w,
		debounceMs:  debounceMs,
		changesChan: make(chan struct{}, 1),
	}

	// Start event loop immediately
	go watcher.eventLoop()
	logger.Info("NewWatcher: Event loop started")

	return watcher, nil
}

// WatchPaths starts watching the specified paths
// If paths is empty, watches the current directory recursively
func (w *Watcher) WatchPaths(paths []string) error {
	logger.Info("WatchPaths: Called with %d paths: %v", len(paths), paths)
	var err error

	if len(paths) == 0 {
		// Watch entire git repository recursively
		logger.Info("WatchPaths: Watching recursively from '.'")
		err = w.watchRecursive(".")
	} else {
		// Extract unique parent directories
		dirs := make(map[string]bool)
		for _, path := range paths {
			// Get absolute path
			absPath, err := filepath.Abs(path)
			if err != nil {
				absPath = path
			}

			// Get directory
			dir := filepath.Dir(absPath)
			dirs[dir] = true
		}

		// Watch each directory
		for dir := range dirs {
			logger.Info("WatchPaths: Adding directory: %s", dir)
			if err := w.watcher.Add(dir); err != nil {
				logger.Error("WatchPaths: Failed to add directory %s: %v", dir, err)
				return err
			}
		}
	}

	if err != nil {
		logger.Error("WatchPaths: Error during setup: %v", err)
		return err
	}

	logger.Info("WatchPaths: Setup complete")
	return nil
}

// watchRecursive watches a directory and all its subdirectories
func (w *Watcher) watchRecursive(root string) error {
	logger.Info("watchRecursive: Starting from root=%s", root)
	dirCount := 0
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Error("watchRecursive: Error walking path %s: %v", path, err)
			return err
		}

		// Skip .git directory
		if info.IsDir() && info.Name() == ".git" {
			logger.Debug("watchRecursive: Skipping .git directory")
			return filepath.SkipDir
		}

		// Skip hidden directories and node_modules, but not the root "." directory
		if info.IsDir() && info.Name() != "." && (len(info.Name()) > 0 && info.Name()[0] == '.' || info.Name() == "node_modules") {
			logger.Debug("watchRecursive: Skipping hidden/node_modules directory: %s", path)
			return filepath.SkipDir
		}

		// Watch directories only
		if info.IsDir() {
			if err := w.watcher.Add(path); err != nil {
				logger.Error("watchRecursive: Failed to add directory %s: %v", path, err)
				return err
			}
			dirCount++
			logger.Debug("watchRecursive: Added directory %s (total: %d)", path, dirCount)
		}

		return nil
	})
}

// eventLoop processes fsnotify events
func (w *Watcher) eventLoop() {
	logger.Info("eventLoop: Started")
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				logger.Info("eventLoop: Events channel closed, exiting")
				return
			}

			logger.Debug("eventLoop: Received event: %s %s", event.Op, event.Name)

			// Detect file changes (write, create, rename, or remove)
			// Some editors use rename or remove+create instead of direct write
			if event.Op&fsnotify.Write == fsnotify.Write ||
				event.Op&fsnotify.Create == fsnotify.Create ||
				event.Op&fsnotify.Rename == fsnotify.Rename ||
				event.Op&fsnotify.Remove == fsnotify.Remove {
				logger.Info("eventLoop: File change detected: %s %s", event.Op, event.Name)
				w.debounceChange()
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				logger.Info("eventLoop: Errors channel closed, exiting")
				return
			}
			logger.Error("eventLoop: Watcher error: %v", err)
		}
	}
}

// debounceChange debounces file change events
func (w *Watcher) debounceChange() {
	logger.Debug("debounceChange: Called")
	// Reset debounce timer
	if w.debouncer != nil {
		w.debouncer.Stop()
		logger.Debug("debounceChange: Stopped previous timer")
	}

	w.debouncer = time.AfterFunc(time.Duration(w.debounceMs)*time.Millisecond, func() {
		logger.Info("debounceChange: Timer fired, sending change notification")
		// Signal change on non-blocking send
		select {
		case w.changesChan <- struct{}{}:
			logger.Info("debounceChange: Change notification sent")
		default:
			logger.Info("debounceChange: Channel already has pending change, skipping")
		}
	})
}

// Changes returns a channel that receives notifications when files change
func (w *Watcher) Changes() <-chan struct{} {
	return w.changesChan
}

// Close stops the watcher
func (w *Watcher) Close() error {
	if w.debouncer != nil {
		w.debouncer.Stop()
	}
	return w.watcher.Close()
}

package git

import (
	"path/filepath"
	"time"

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
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		watcher:     w,
		debounceMs:  debounceMs,
		changesChan: make(chan struct{}, 1),
	}, nil
}

// WatchPaths starts watching the specified paths
// If paths is empty, watches the current directory
func (w *Watcher) WatchPaths(paths []string) error {
	if len(paths) == 0 {
		// Watch current directory
		paths = []string{"."}
	}

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
		if err := w.watcher.Add(dir); err != nil {
			return err
		}
	}

	// Start event loop in background
	go w.eventLoop()

	return nil
}

// eventLoop processes fsnotify events
func (w *Watcher) eventLoop() {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Only care about write and create events
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				w.debounceChange()
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			// Log error but continue watching
			_ = err // TODO: proper error handling
		}
	}
}

// debounceChange debounces file change events
func (w *Watcher) debounceChange() {
	// Reset debounce timer
	if w.debouncer != nil {
		w.debouncer.Stop()
	}

	w.debouncer = time.AfterFunc(time.Duration(w.debounceMs)*time.Millisecond, func() {
		// Signal change on non-blocking send
		select {
		case w.changesChan <- struct{}{}:
		default:
			// Channel already has a pending change
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

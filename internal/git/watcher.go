package git

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"git.15b.it/eno/critic/internal/logger"
	"github.com/fsnotify/fsnotify"
	"github.com/samber/lo"
)

// FileChange represents a file change event
type FileChange struct {
	Path string
}

// Watcher watches files for changes using a pipeline architecture:
// fsnotify → eventLoop → filterLoop → debounceLoop → changesChan
type Watcher struct {
	watcher     *fsnotify.Watcher
	debounceMs  int
	changesChan chan FileChange

	// Filtering
	paths   []string // Git-relative paths to filter
	gitRoot string   // Git repository root

	// Pipeline channels
	rawEvents      chan fsnotify.Event
	filteredEvents chan string // Absolute file paths

	// Per-file debouncing
	debouncers map[string]*time.Timer
	debounceMu sync.Mutex

	// Lifecycle
	stopChan chan struct{}
}

// NewWatcher creates a new file watcher
func NewWatcher(debounceMs int) (*Watcher, error) {
	logger.Info("NewWatcher: Creating watcher with debounce=%dms", debounceMs)
	w, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Error("NewWatcher: Failed to create fsnotify watcher: %v", err)
		return nil, err
	}

	// Get git root for path filtering
	initPathCache()
	gitRoot := gitRootCache

	watcher := &Watcher{
		watcher:        w,
		debounceMs:     debounceMs,
		changesChan:    make(chan FileChange, 10), // Buffered for multiple files
		rawEvents:      make(chan fsnotify.Event, 100),
		filteredEvents: make(chan string, 100),
		debouncers:     make(map[string]*time.Timer),
		stopChan:       make(chan struct{}),
		gitRoot:        gitRoot,
	}

	// Start pipeline goroutines
	go watcher.eventLoop()
	go watcher.filterLoop()
	go watcher.debounceLoop()
	logger.Info("NewWatcher: Pipeline started (eventLoop, filterLoop, debounceLoop)")

	return watcher, nil
}

// WatchPaths starts watching the specified paths recursively
// If paths is empty or ["."], watches the current directory recursively
// If paths contains files, watches their parent directories recursively
// If paths contains directories, watches those directories recursively
func (w *Watcher) WatchPaths(paths []string) error {
	logger.Info("WatchPaths: Called with %d paths: %v", len(paths), paths)

	// Store paths for filtering (convert to git-relative)
	w.paths = lo.Map(paths, func(p string, _ int) string {
		if filepath.IsAbs(p) {
			return AbsPathToGitPath(p)
		}
		return p
	})
	logger.Info("WatchPaths: Stored %d paths for filtering: %v", len(w.paths), w.paths)

	// Watch current directory recursively if no paths or just "."
	if len(paths) == 0 || (len(paths) == 1 && paths[0] == ".") {
		logger.Info("WatchPaths: Watching current directory recursively")
		return w.watchRecursive(".")
	}

	// Collect unique directories to watch
	dirsToWatch := make(map[string]bool)

	for _, path := range paths {
		// Get absolute path
		absPath, err := filepath.Abs(path)
		if err != nil {
			absPath = path
		}

		// Check if path is a directory
		info, statErr := os.Stat(absPath)
		if statErr == nil && info.IsDir() {
			// Path is a directory - watch it recursively
			logger.Info("WatchPaths: Will watch directory recursively: %s", absPath)
			dirsToWatch[absPath] = true
		} else {
			// Path is a file (or doesn't exist) - watch parent directory recursively
			dir := filepath.Dir(absPath)
			logger.Info("WatchPaths: Will watch parent directory recursively: %s", dir)
			dirsToWatch[dir] = true
		}
	}

	// Watch each directory recursively
	for dir := range dirsToWatch {
		logger.Info("WatchPaths: Watching directory recursively: %s", dir)
		if err := w.watchRecursive(dir); err != nil {
			logger.Error("WatchPaths: Failed to watch directory %s: %v", dir, err)
			return err
		}
	}

	logger.Info("WatchPaths: Setup complete, watching %d directories recursively", len(dirsToWatch))
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

// eventLoop forwards fsnotify events to the pipeline
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
			if event.Op&fsnotify.Write == fsnotify.Write ||
				event.Op&fsnotify.Create == fsnotify.Create ||
				event.Op&fsnotify.Rename == fsnotify.Rename ||
				event.Op&fsnotify.Remove == fsnotify.Remove {
				logger.Info("eventLoop: File change detected: %s %s", event.Op, event.Name)
				select {
				case w.rawEvents <- event:
					logger.Debug("eventLoop: Forwarded event to filter")
				default:
					logger.Debug("eventLoop: Dropped event (channel full): %s", event.Name)
				}
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				logger.Info("eventLoop: Errors channel closed, exiting")
				return
			}
			logger.Error("eventLoop: Watcher error: %v", err)

		case <-w.stopChan:
			logger.Info("eventLoop: Stop signal received, exiting")
			return
		}
	}
}

// filterLoop filters events based on configured paths
func (w *Watcher) filterLoop() {
	logger.Info("filterLoop: Started")
	for {
		select {
		case event := <-w.rawEvents:
			if w.shouldIncludeFile(event.Name) {
				logger.Debug("filterLoop: File passed filter: %s", event.Name)
				select {
				case w.filteredEvents <- event.Name:
					logger.Debug("filterLoop: Forwarded to debouncer")
				default:
					logger.Debug("filterLoop: Dropped event (channel full): %s", event.Name)
				}
			} else {
				logger.Debug("filterLoop: File filtered out: %s", event.Name)
			}

		case <-w.stopChan:
			logger.Info("filterLoop: Stop signal received, exiting")
			return
		}
	}
}

// debounceLoop debounces events per-file
func (w *Watcher) debounceLoop() {
	logger.Info("debounceLoop: Started")
	for {
		select {
		case path := <-w.filteredEvents:
			logger.Debug("debounceLoop: Received event for %s", path)

			w.debounceMu.Lock()

			// Cancel existing timer for this file
			if timer, exists := w.debouncers[path]; exists {
				timer.Stop()
				logger.Debug("debounceLoop: Cancelled previous timer for %s", path)
			}

			// Create new timer for this file
			w.debouncers[path] = time.AfterFunc(
				time.Duration(w.debounceMs)*time.Millisecond,
				func() {
					logger.Info("debounceLoop: Timer fired for %s, emitting change", path)
					// Send change notification
					select {
					case w.changesChan <- FileChange{Path: path}:
						logger.Info("debounceLoop: Change notification sent for %s", path)
					default:
						logger.Debug("debounceLoop: Dropped change (channel full): %s", path)
					}

					// Clean up debouncer
					w.debounceMu.Lock()
					delete(w.debouncers, path)
					w.debounceMu.Unlock()
				},
			)

			w.debounceMu.Unlock()

		case <-w.stopChan:
			logger.Info("debounceLoop: Stop signal received, exiting")
			return
		}
	}
}

// shouldIncludeFile checks if a file should be included based on configured paths
func (w *Watcher) shouldIncludeFile(absPath string) bool {
	// If no paths or just ".", include everything
	if len(w.paths) == 0 || (len(w.paths) == 1 && w.paths[0] == ".") {
		return true
	}

	// Convert to git-relative path
	gitPath := AbsPathToGitPath(absPath)
	if gitPath == "" {
		logger.Debug("shouldIncludeFile: Failed to convert to git path: %s", absPath)
		return false
	}

	// Check if file matches any of the specified paths
	for _, p := range w.paths {
		// Exact match
		if gitPath == p {
			logger.Debug("shouldIncludeFile: Exact match for %s", gitPath)
			return true
		}
		// Path is a directory and file is inside it
		if strings.HasSuffix(p, "/") && strings.HasPrefix(gitPath, p) {
			logger.Debug("shouldIncludeFile: Directory match for %s (in %s)", gitPath, p)
			return true
		}
		// Check if p is a directory (no trailing slash) and file is inside
		if strings.HasPrefix(gitPath, p+"/") {
			logger.Debug("shouldIncludeFile: Directory prefix match for %s (in %s/)", gitPath, p)
			return true
		}
	}

	return false
}

// Changes returns a channel that receives notifications when files change
func (w *Watcher) Changes() <-chan FileChange {
	return w.changesChan
}

// Close stops the watcher and all goroutines
func (w *Watcher) Close() error {
	logger.Info("Close: Stopping watcher")

	// Signal all goroutines to stop
	close(w.stopChan)

	// Stop all active debounce timers
	w.debounceMu.Lock()
	for path, timer := range w.debouncers {
		timer.Stop()
		logger.Debug("Close: Stopped timer for %s", path)
	}
	w.debounceMu.Unlock()

	// Close the fsnotify watcher
	return w.watcher.Close()
}

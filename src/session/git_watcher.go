package session

import (
	"sync"
	"time"

	"github.com/radiospiel/critic/simple-go/logger"
	"github.com/radiospiel/critic/src/git"
)

// GitWatcher watches for changes to git references and kicks off fileDiffs reloading
type GitWatcher struct {
	state         *Session
	bases         []string          // Base refs to watch (e.g., ["main", "origin/main", "HEAD"])
	resolvedBases map[string]string // Current resolved SHAs
	mu            sync.RWMutex

	pollInterval time.Duration
	stopChan     chan struct{}
	running      bool

	// Callbacks
	onBasesChanged func()
	onDiffNeeded   func(baseRef string, baseSHA string) // Called when fileDiffs needs to be reloaded
}

// NewGitWatcher creates a new git watcher
func NewGitWatcher(state *Session) *GitWatcher {
	return &GitWatcher{
		state:         state,
		resolvedBases: make(map[string]string),
		pollInterval:  10 * time.Second,
		stopChan:      make(chan struct{}),
	}
}

// SetBases sets the base refs to watch
func (w *GitWatcher) SetBases(bases []string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.bases = bases
}

// SetPollInterval sets the polling interval
func (w *GitWatcher) SetPollInterval(interval time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.pollInterval = interval
}

// OnBasesChanged sets the callback for when bases change
func (w *GitWatcher) OnBasesChanged(callback func()) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.onBasesChanged = callback
}

// OnDiffNeeded sets the callback for when fileDiffs needs reloading
func (w *GitWatcher) OnDiffNeeded(callback func(baseRef string, baseSHA string)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.onDiffNeeded = callback
}

// Start starts the git watcher
func (w *GitWatcher) Start() error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return nil
	}
	w.running = true
	w.mu.Unlock()

	// Do initial resolution
	w.resolveAll()

	go w.pollLoop()
	logger.Info("GitWatcher: Started with poll interval %v", w.pollInterval)
	return nil
}

// Stop stops the watcher
func (w *GitWatcher) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.running = false
	w.mu.Unlock()

	close(w.stopChan)
	logger.Info("GitWatcher: Stopped")
}

// pollLoop periodically checks for changes
func (w *GitWatcher) pollLoop() {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if changed := w.checkForChanges(); changed {
				w.mu.RLock()
				callback := w.onBasesChanged
				w.mu.RUnlock()

				if callback != nil {
					logger.Info("GitWatcher: Bases changed, triggering callback")
					callback()
				}
			}
		case <-w.stopChan:
			return
		}
	}
}

// resolveAll resolves all base refs to SHAs
func (w *GitWatcher) resolveAll() {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, base := range w.bases {
		sha := git.ResolveRef(base)
		w.resolvedBases[base] = sha
	}

	// Update state with resolved bases
	if w.state != nil {
		w.state.SetResolvedBases(w.resolvedBases)
	}
}

// checkForChanges checks if any bases have changed
func (w *GitWatcher) checkForChanges() bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	changed := false
	for _, base := range w.bases {
		sha := git.ResolveRef(base)

		if w.resolvedBases[base] != sha {
			logger.Info("GitWatcher: Base %s changed from %s to %s",
				base, truncateSHA(w.resolvedBases[base]), truncateSHA(sha))
			w.resolvedBases[base] = sha
			changed = true
		}
	}

	if changed && w.state != nil {
		w.state.SetResolvedBases(w.resolvedBases)
	}

	return changed
}

// GetResolvedBase returns the resolved SHA for a base ref
func (w *GitWatcher) GetResolvedBase(baseRef string) (string, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	sha, ok := w.resolvedBases[baseRef]
	return sha, ok
}

// GetResolvedBases returns all resolved bases
func (w *GitWatcher) GetResolvedBases() map[string]string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	result := make(map[string]string, len(w.resolvedBases))
	for k, v := range w.resolvedBases {
		result[k] = v
	}
	return result
}

// ForceRefresh forces a refresh of all bases and triggers callbacks if changed
func (w *GitWatcher) ForceRefresh() bool {
	if changed := w.checkForChanges(); changed {
		w.mu.RLock()
		callback := w.onBasesChanged
		w.mu.RUnlock()

		if callback != nil {
			callback()
		}
		return true
	}
	return false
}

// IsRunning returns whether the watcher is running
func (w *GitWatcher) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.running
}

// truncateSHA returns the first 7 characters of a SHA for logging
func truncateSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

package git

import (
	"fmt"
	"sync"
	"time"
)

// BaseResolver handles resolution of git references to commit SHAs
// and provides polling to detect when bases change.
type BaseResolver struct {
	bases         []string          // Original base refs (e.g., ["merge-base", "origin/main", "HEAD"])
	current       string            // Current target ref
	resolvedBases map[string]string // Map of base ref -> resolved SHA
	mu            sync.RWMutex
	stopChan      chan struct{}
	onChange      func() // Callback when any base changes
}

// NewBaseResolver creates a new base resolver
func NewBaseResolver(bases []string, current string, onChange func()) (*BaseResolver, error) {
	r := &BaseResolver{
		bases:         bases,
		current:       current,
		resolvedBases: make(map[string]string),
		stopChan:      make(chan struct{}),
		onChange:      onChange,
	}

	// Initial resolution
	if err := r.resolve(); err != nil {
		return nil, err
	}

	// Start polling
	go r.poll()

	return r, nil
}

// resolve resolves all bases to commit SHAs
func (r *BaseResolver) resolve() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, base := range r.bases {
		sha, err := r.resolveOne(base)
		if err != nil {
			return fmt.Errorf("failed to resolve base %s: %w", base, err)
		}
		r.resolvedBases[base] = sha
	}

	return nil
}

// resolveOne resolves a single base reference to a commit SHA
func (r *BaseResolver) resolveOne(base string) (string, error) {
	// Special case: "merge-base" resolves to the merge base with main/master
	if base == "merge-base" {
		sha, err := GetMergeBase()
		if err != nil {
			return "", fmt.Errorf("failed to get merge base: %w", err)
		}
		return sha, nil
	}

	// For other refs, use ResolveRef
	sha, err := ResolveRef(base)
	if err != nil {
		return "", err
	}

	return sha, nil
}

// poll checks every 10 seconds if any bases have changed
func (r *BaseResolver) poll() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if r.checkForChanges() {
				if r.onChange != nil {
					r.onChange()
				}
			}
		case <-r.stopChan:
			return
		}
	}
}

// checkForChanges checks if any bases have changed and updates if so
// Returns true if any changes were detected
func (r *BaseResolver) checkForChanges() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	changed := false
	for _, base := range r.bases {
		sha, err := r.resolveOne(base)
		if err != nil {
			// If resolution fails, skip this base
			continue
		}

		if r.resolvedBases[base] != sha {
			r.resolvedBases[base] = sha
			changed = true
		}
	}

	return changed
}

// GetResolvedBases returns a copy of the resolved bases
func (r *BaseResolver) GetResolvedBases() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]string, len(r.resolvedBases))
	for k, v := range r.resolvedBases {
		result[k] = v
	}
	return result
}

// GetResolvedBase returns the resolved SHA for a specific base
func (r *BaseResolver) GetResolvedBase(base string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sha, ok := r.resolvedBases[base]
	return sha, ok
}

// Stop stops the polling goroutine
func (r *BaseResolver) Stop() {
	close(r.stopChan)
}

package critic

import (
	"fmt"
	"sync"

	"github.com/radiospiel/critic/src/git"
	ctypes "github.com/radiospiel/critic/src/pkg/types"
)

// GitDiffState implements DiffState using git as the source
type GitDiffState struct {
	paths        []string
	mode         git.DiffMode
	diff         *ctypes.Diff
	mu           sync.RWMutex
	callbacks    map[int]OnChangeCallback
	nextCallbackID int
}

// NewGitDiffState creates a new GitDiffState
func NewGitDiffState(paths []string, mode git.DiffMode) (*GitDiffState, error) {
	state := &GitDiffState{
		paths:        paths,
		mode:         mode,
		callbacks:    make(map[int]OnChangeCallback),
		nextCallbackID: 0,
	}

	// Initial load
	if err := state.Refresh(); err != nil {
		return nil, err
	}

	return state, nil
}

// GetFiles returns a list of all changed files with their states
func (s *GitDiffState) GetFiles() []FileInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.diff == nil {
		return []FileInfo{}
	}

	files := make([]FileInfo, 0, len(s.diff.Files))
	for _, file := range s.diff.Files {
		info := FileInfo{
			Path: file.NewPath,
		}

		if file.IsNew {
			info.State = FileCreated
		} else if file.IsDeleted {
			info.State = FileDeleted
			info.Path = file.OldPath
		} else {
			info.State = FileChanged
		}

		files = append(files, info)
	}

	return files
}

// GetDiffDetails returns detailed diff information for a specific file
func (s *GitDiffState) GetDiffDetails(path string) (*DiffDetails, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.diff == nil {
		return nil, fmt.Errorf("no diff loaded")
	}

	// Find the file in the diff
	var file *ctypes.FileDiff
	for _, f := range s.diff.Files {
		filePath := f.NewPath
		if f.IsDeleted {
			filePath = f.OldPath
		}
		if filePath == path {
			file = f
			break
		}
	}

	if file == nil {
		return nil, fmt.Errorf("file not found in diff: %s", path)
	}

	details := &DiffDetails{
		Path:  path,
		Hunks: file.Hunks,
	}

	// Get original content (if file existed before)
	if !file.IsNew {
		content, err := git.GetFileContent(file.OldPath, "HEAD")
		if err == nil {
			details.OriginalContent = content
		}
	}

	// Get current content (if file exists now)
	if !file.IsDeleted {
		content, err := git.GetFileContent(file.NewPath, "")
		if err == nil {
			details.CurrentContent = content
		}
	}

	return details, nil
}

// OnChange registers a callback to be notified when diff details change
func (s *GitDiffState) OnChange(callback OnChangeCallback) func() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Assign ID and store callback
	id := s.nextCallbackID
	s.nextCallbackID++
	s.callbacks[id] = callback

	// Return unregister function
	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.callbacks, id)
	}
}

// Refresh updates the diff state by re-reading from git
func (s *GitDiffState) Refresh() error {
	newDiff, err := git.GetDiff(s.paths, s.mode)
	if err != nil {
		return fmt.Errorf("failed to get diff: %w", err)
	}

	s.mu.Lock()
	oldDiff := s.diff
	s.diff = newDiff
	// Copy callbacks to avoid holding lock during notifications
	callbacks := make(map[int]OnChangeCallback, len(s.callbacks))
	for id, cb := range s.callbacks {
		callbacks[id] = cb
	}
	s.mu.Unlock()

	// Notify callbacks of changes
	s.notifyChanges(oldDiff, newDiff, callbacks)

	return nil
}

// notifyChanges compares old and new diffs and notifies callbacks
func (s *GitDiffState) notifyChanges(oldDiff, newDiff *ctypes.Diff, callbacks map[int]OnChangeCallback) {
	if len(callbacks) == 0 {
		return
	}

	// Build maps for quick lookup
	oldFiles := make(map[string]*ctypes.FileDiff)
	if oldDiff != nil {
		for _, file := range oldDiff.Files {
			path := file.NewPath
			if file.IsDeleted {
				path = file.OldPath
			}
			oldFiles[path] = file
		}
	}

	newFiles := make(map[string]*ctypes.FileDiff)
	if newDiff != nil {
		for _, file := range newDiff.Files {
			path := file.NewPath
			if file.IsDeleted {
				path = file.OldPath
			}
			newFiles[path] = file
		}
	}

	// Find all paths (union of old and new)
	allPaths := make(map[string]bool)
	for path := range oldFiles {
		allPaths[path] = true
	}
	for path := range newFiles {
		allPaths[path] = true
	}

	// Notify for each changed/added/removed file
	for path := range allPaths {
		oldFile := oldFiles[path]
		newFile := newFiles[path]

		// Skip if no change
		if oldFile != nil && newFile != nil && filesEqual(oldFile, newFile) {
			continue
		}

		var oldDetails, newDetails *DiffDetails

		if oldFile != nil {
			oldDetails = &DiffDetails{
				Path:  path,
				Hunks: oldFile.Hunks,
			}
		}

		if newFile != nil {
			newDetails = &DiffDetails{
				Path:  path,
				Hunks: newFile.Hunks,
			}
		}

		// Notify all callbacks
		for _, callback := range callbacks {
			callback(oldDetails, newDetails)
		}
	}
}

// filesEqual checks if two FileDiff objects are equal (simplified comparison)
func filesEqual(a, b *ctypes.FileDiff) bool {
	if a.NewPath != b.NewPath || a.OldPath != b.OldPath {
		return false
	}
	if a.IsNew != b.IsNew || a.IsDeleted != b.IsDeleted || a.IsRenamed != b.IsRenamed {
		return false
	}
	if len(a.Hunks) != len(b.Hunks) {
		return false
	}
	// For now, just compare hunk count. Could do deeper comparison if needed.
	return true
}

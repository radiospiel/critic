package server

import (
	"strings"
	"sync"

	"github.com/radiospiel/critic/simple-go/logger"
	"github.com/radiospiel/critic/simple-go/preconditions"
	"github.com/radiospiel/critic/simple-go/tasks"
	"github.com/radiospiel/critic/simple-go/utils"
	"github.com/radiospiel/critic/src/git"
	"github.com/radiospiel/critic/src/pkg/critic"
	"github.com/radiospiel/critic/src/pkg/types"
)

// State represents the current state of the session
type State string

const (
	StateInitialising State = "INITIALISING"
	StateReady        State = "READY"
)

// DiffArgs holds the arguments for generating a diff.
// Unlike session.DiffArgs, this does not include currentBase as it is managed
// separately in the Session.
type DiffArgs struct {
	Paths []string `json:"paths"` // File path patterns
}

// Session manages the state for the API server.
// It tracks the current diff state and handles background diff loading.
type Session struct {
	mu sync.RWMutex

	// Configuration
	gitRoot   string
	messaging critic.Messaging
	paths     []string
	diffBases []string

	// State
	state       State
	currentBase string
	diff        []*types.FileDiff

	// Background task management
	currentTask *tasks.Task[diffResult]

	// File watcher for the currently viewed file
	fileWatcher *utils.FileWatcher
}

// diffResult holds the result of a diff loading operation
type diffResult struct {
	files []*types.FileDiff
	err   error
}

// NewSession creates a new API session with the given diff bases.
// The first diff base is used as the current base.
func NewSession(gitRoot string, messaging critic.Messaging, paths []string, diffBases []string) *Session {
	s := &Session{
		gitRoot:   gitRoot,
		messaging: messaging,
		paths:     paths,
		state:     StateReady,
	}
	if len(diffBases) > 0 {
		// SetDiffBases sets all valid diff bases and initializes the current base
		// to be the first of the passed in bases.
		_ = s.SetDiffBases(diffBases)
	}
	return s
}

// GetState returns the current session state
func (s *Session) GetState() State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// GetCurrentBase returns the current base ref
func (s *Session) GetCurrentBase() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentBase
}

// GetDiffBases returns the available diff bases
func (s *Session) GetDiffBases() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]string{}, s.diffBases...)
}

// GetDiffSummary returns the current diff summary (file list without hunks)
func (s *Session) GetDiffSummary() []*types.FileDiff {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.diff
}

// GetFileDiff returns the full diff for a specific file path.
// It loads the diff on-demand using git.GetDiff.
// contextLines specifies the number of context lines (minimum 3, default 3).
func (s *Session) GetFileDiff(path string, contextLines int) *types.FileDiff {
	s.mu.RLock()
	currentBase := s.currentBase
	s.mu.RUnlock()

	if currentBase == "" {
		logger.Error("session does not have current base")
		return nil
	}

	preconditions.Check(path != "", "path required")

	// Load the full diff for the specific file
	fileDiff, err := git.GetDiff(currentBase, path, contextLines)
	if err != nil {
		logger.Error("git.GetDiff returns error %v", err)
		return nil
	}
	if fileDiff == nil {
		logger.Error("git.GetDiff returns empty diff")
		return nil
	}

	return fileDiff
}

// HeadCommit returns the current HEAD commit SHA
func (s *Session) HeadCommit() string {
	return git.ResolveRef("HEAD")
}

// SetDiffBases sets all valid diff bases, and initialises the current base
// to be the first of the passed in bases.
func (s *Session) SetDiffBases(bases []string) error {
	preconditions.Check(len(bases) > 0, "bases required")

	s.mu.Lock()
	s.diffBases = append([]string{}, bases...)
	s.mu.Unlock()

	return s.SetCurrentDiffBase(bases[0])
}

// SetCurrentDiffBase sets the current base ref for the session, and
// starts loading the diff in the background.
func (s *Session) SetCurrentDiffBase(base string) error {
	s.mu.Lock()
	s.currentBase = base
	s.mu.Unlock()

	<-s.TriggerDiff()
	return nil
}

// TriggerDiff reloads the diff with the current base.
// Returns a channel that closes when the diff is ready.
func (s *Session) TriggerDiff() <-chan struct{} {
	done := make(chan struct{})

	currentBase := s.GetCurrentBase()
	if currentBase == "" {
		close(done)
		return done
	}

	s.mu.Lock()

	// Abort any existing task
	if s.currentTask != nil {
		s.currentTask.Abort()
		s.currentTask = nil
	}

	// Set state to initialising
	s.state = StateInitialising
	s.mu.Unlock()

	// Start background task to load diff summary
	task, err := tasks.RunExclusively("api-session-diff", func() diffResult {
		files, err := git.GetDiffNames(currentBase, []string{})
		if err != nil {
			return diffResult{err: err}
		}

		// Filter by paths if specified
		if len(s.paths) > 0 {
			files = filterDiffByPaths(files, s.paths)
		}

		return diffResult{files: files}
	})

	if err != nil {
		s.mu.Lock()
		s.state = StateReady
		s.mu.Unlock()
		close(done)
		return done
	}

	s.mu.Lock()
	s.currentTask = task
	s.mu.Unlock()

	// Wait for result in background, update state, and signal done
	go func() {
		result := <-task.Done()

		s.mu.Lock()
		// Only update if this task is still current
		if s.currentTask == task {
			s.currentTask = nil
			if result.err == nil {
				s.diff = result.files
			}
			s.state = StateReady
		}
		s.mu.Unlock()

		close(done)
	}()

	return done
}

// SetFileWatcher sets the file watcher for the currently viewed file.
// It stops any existing watcher first.
func (s *Session) SetFileWatcher(watcher *utils.FileWatcher) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Stop existing watcher if any
	if s.fileWatcher != nil {
		s.fileWatcher.Close()
	}
	s.fileWatcher = watcher
}

// GetFileWatcher returns the current file watcher.
func (s *Session) GetFileWatcher() *utils.FileWatcher {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.fileWatcher
}

// StopFileWatcher stops the current file watcher if one exists.
func (s *Session) StopFileWatcher() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.fileWatcher != nil {
		s.fileWatcher.Close()
		s.fileWatcher = nil
	}
}

// filterDiffByExtensions filters the diff files to only include files with
// the specified extensions.
func filterDiffByExtensions(files []*types.FileDiff, extensions []string) []*types.FileDiff {
	if files == nil || len(extensions) == 0 {
		return files
	}

	extMap := make(map[string]bool, len(extensions))
	for _, ext := range extensions {
		// Normalize extension (ensure it starts with .)
		if len(ext) > 0 && ext[0] != '.' {
			ext = "." + ext
		}
		extMap[ext] = true
	}

	filtered := make([]*types.FileDiff, 0, len(files))

	for _, file := range files {
		path := file.GetPath()
		for ext := range extMap {
			if len(path) >= len(ext) && path[len(path)-len(ext):] == ext {
				filtered = append(filtered, file)
				break
			}
		}
	}

	return filtered
}

// filterDiffByPaths filters the diff files to only include files that match
// any of the specified path patterns.
func filterDiffByPaths(files []*types.FileDiff, paths []string) []*types.FileDiff {
	if files == nil || len(paths) == 0 {
		return files
	}

	filtered := make([]*types.FileDiff, 0, len(files))

	for _, file := range files {
		filePath := file.GetPath()
		for _, pattern := range paths {
			// Simple prefix matching for now
			if strings.HasPrefix(filePath, pattern) || filePath == pattern {
				filtered = append(filtered, file)
				break
			}
		}
	}

	return filtered
}

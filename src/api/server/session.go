package server

import (
	"strings"
	"sync"

	"github.com/radiospiel/critic/simple-go/logger"
	"github.com/radiospiel/critic/simple-go/preconditions"
	"github.com/radiospiel/critic/simple-go/tasks"
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
	Bases      []string `json:"bases"`      // List of base refs
	Paths      []string `json:"paths"`      // File path patterns
	Extensions []string `json:"extensions"` // File extensions to include
}

// Session manages the state for the API server.
// It tracks the current diff state and handles background diff loading.
type Session struct {
	mu sync.RWMutex

	// Configuration
	gitRoot   string
	messaging critic.Messaging
	args      DiffArgs

	// State
	state       State
	currentBase string
	diff        *types.Diff

	// Background task management
	currentTask *tasks.Task[diffResult]
}

// diffResult holds the result of a diff loading operation
type diffResult struct {
	diff *types.Diff
	err  error
}

// NewSession creates a new API session.
func NewSession(gitRoot string, messaging critic.Messaging, args DiffArgs) *Session {
	return &Session{
		gitRoot:   gitRoot,
		messaging: messaging,
		args:      args,
		state:     StateReady,
	}
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

// GetDiffSummary returns the current diff summary (file list without hunks)
func (s *Session) GetDiffSummary() *types.Diff {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.diff
}

// GetFileDiff returns the full diff for a specific file path.
// It loads the diff on-demand using git.GetDiffBetween.
func (s *Session) GetFileDiff(path string) *types.FileDiff {
	s.mu.RLock()
	currentBase := s.currentBase
	s.mu.RUnlock()

	if currentBase == "" {
		logger.Error("session does not have current base")
		return nil
	}

	preconditions.Check(path != "", "path required")

	// Load the full diff for the specific file
	diff, err := git.GetDiffBetween(currentBase, "current", []string{path})
	if err != nil {
		logger.Error("git.GetDiffBetween returns error %v", err)
		return nil
	}
	if diff == nil || len(diff.Files) == 0 {
		logger.Error("git.GetDiffBetween returns empty diff")
		return nil
	}

	return diff.Files[0]
}

// GetArgs returns the DiffArgs
func (s *Session) GetArgs() DiffArgs {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.args
}

// HeadCommit returns the current HEAD commit SHA
func (s *Session) HeadCommit() string {
	return git.ResolveRef("HEAD")
}

// SetRefs sets the base ref and starts loading the diff in the background.
// It uses tasks.RunExclusively to ensure only one diff loading operation
// runs at a time. If a diff load is already in progress, it will be aborted.
func (s *Session) SetRefs(base string) error {
	s.mu.Lock()

	// Abort any existing task
	if s.currentTask != nil {
		s.currentTask.Abort()
		s.currentTask = nil
	}

	// Set state to initialising
	s.state = StateInitialising
	s.mu.Unlock()

	// Get paths from args
	args := s.GetArgs()

	// Start background task to load diff summary
	task, err := tasks.RunExclusively("api-session-diff", func() diffResult {
		// Resolve the base ref to a commit SHA
		resolvedBase := git.ResolveRef(base)

		// Run git diff --name-status for file summary (no hunks)
		diff, err := git.GetDiffNamesBetween(resolvedBase, "current")
		if err != nil {
			return diffResult{err: err}
		}

		// Filter by extensions if specified
		if len(args.Extensions) > 0 {
			diff = filterDiffByExtensions(diff, args.Extensions)
		}

		// Filter by paths if specified
		if len(args.Paths) > 0 {
			diff = filterDiffByPaths(diff, args.Paths)
		}

		return diffResult{diff: diff}
	})

	if err != nil {
		// Task with same ID already running - this shouldn't happen since we abort above
		s.mu.Lock()
		s.state = StateReady
		s.mu.Unlock()
		return err
	}

	s.mu.Lock()
	s.currentTask = task
	s.mu.Unlock()

	// Wait for result in background and update state
	go func() {
		result := <-task.Done()

		s.mu.Lock()
		defer s.mu.Unlock()

		// Only update if this task is still current
		if s.currentTask == task {
			s.currentTask = nil
			if result.err == nil {
				s.currentBase = base
				s.diff = result.diff
			}
			s.state = StateReady
		}
	}()

	return nil
}

// filterDiffByExtensions filters the diff files to only include files with
// the specified extensions.
func filterDiffByExtensions(diff *types.Diff, extensions []string) *types.Diff {
	if diff == nil || len(extensions) == 0 {
		return diff
	}

	extMap := make(map[string]bool, len(extensions))
	for _, ext := range extensions {
		// Normalize extension (ensure it starts with .)
		if len(ext) > 0 && ext[0] != '.' {
			ext = "." + ext
		}
		extMap[ext] = true
	}

	filtered := &types.Diff{
		Files: make([]*types.FileDiff, 0, len(diff.Files)),
	}

	for _, file := range diff.Files {
		path := file.GetPath()
		for ext := range extMap {
			if len(path) >= len(ext) && path[len(path)-len(ext):] == ext {
				filtered.Files = append(filtered.Files, file)
				break
			}
		}
	}

	return filtered
}

// filterDiffByPaths filters the diff files to only include files that match
// any of the specified path patterns.
func filterDiffByPaths(diff *types.Diff, paths []string) *types.Diff {
	if diff == nil || len(paths) == 0 {
		return diff
	}

	filtered := &types.Diff{
		Files: make([]*types.FileDiff, 0, len(diff.Files)),
	}

	for _, file := range diff.Files {
		filePath := file.GetPath()
		for _, pattern := range paths {
			// Simple prefix matching for now
			if strings.HasPrefix(filePath, pattern) || filePath == pattern {
				filtered.Files = append(filtered.Files, file)
				break
			}
		}
	}

	return filtered
}

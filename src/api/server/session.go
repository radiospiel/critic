package server

import (
	"sync"

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

// GetDiff returns the current diff
func (s *Session) GetDiff() *types.Diff {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.diff
}

// GetArgs returns the DiffArgs
func (s *Session) GetArgs() DiffArgs {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.args
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

	// Start background task to load diff
	task, err := tasks.RunExclusively("api-session-diff", func() diffResult {
		// Resolve the base ref to a commit SHA
		resolvedBase, err := git.ResolveRef(base)
		if err != nil {
			return diffResult{err: err}
		}

		// Run git diff for all files (passing nil paths gets all files)
		diff, err := git.GetDiffBetween(resolvedBase, "current", args.Paths)
		if err != nil {
			return diffResult{err: err}
		}

		// Filter by extensions if specified
		if len(args.Extensions) > 0 {
			diff = filterDiffByExtensions(diff, args.Extensions)
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

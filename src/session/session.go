// Package session provides a data structure for managing application state
// including fileDiffs arguments, files, conversations, and watchers for changes.
package session

import (
	"sync"

	"github.com/radiospiel/critic/simple-go/logger"
	"github.com/radiospiel/critic/simple-go/preconditions"
	"github.com/radiospiel/critic/src/pkg/critic"
	"github.com/radiospiel/critic/src/pkg/types"
)

// FilterMode represents the current file/hunk filter mode
type FilterMode int

const (
	FilterModeNone FilterMode = iota
	FilterModeWithComments
	FilterModeWithUnresolved
)

// String returns a display name for the filter mode
func (m FilterMode) String() string {
	switch m {
	case FilterModeWithComments:
		return "With Comments"
	case FilterModeWithUnresolved:
		return "Unresolved Only"
	default:
		return "All"
	}
}

// DiffArgs holds the arguments for generating a fileDiffs
type DiffArgs struct {
	Bases       []string `json:"bases"`       // List of base refs
	CurrentBase int      `json:"currentBase"` // Index of current base
	Paths       []string `json:"paths"`       // File path patterns
	Extensions  []string `json:"extensions"`  // File extensions to include
}

// Selection holds the current selection state
type Selection struct {
	FileIndex   int    `json:"fileIndex"`
	FilePath    string `json:"filePath"`
	FocusedPane string `json:"focusedPane"`
}

// Session manages the application state for a review session.
type Session struct {
	messaging critic.Messaging
	gitRoot   string
	mu        sync.RWMutex

	// Diff state
	diffArgs  DiffArgs
	fileDiffs []*types.FileDiff
}

// NewSession creates a new Session with the given parameters.
// messaging must not be nil - use critic.DummyMessaging{} for testing.
func NewSession(gitRoot string, messaging critic.Messaging, args DiffArgs) (*Session, error) {
	preconditions.Check(messaging != nil, "messaging must not be nil")

	logger.Warn("*** NewSession: created session w/gitRoot: %v", gitRoot)

	s := &Session{
		messaging: messaging,
		gitRoot:   gitRoot,
		diffArgs:  args,
	}

	return s, nil
}

// --- Diff Args ---

// SetCurrentBase sets the current base index
func (s *Session) SetCurrentBase(index int) {
	s.mu.Lock()
	s.diffArgs.CurrentBase = index
	s.mu.Unlock()
}

// GetCurrentBase returns the current base index
func (s *Session) GetCurrentBase() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.diffArgs.CurrentBase
}

// GetCurrentBaseName returns the name of the current base ref
func (s *Session) GetCurrentBaseName() string {
	args := s.diffArgs
	if args.CurrentBase < 0 || args.CurrentBase >= len(args.Bases) {
		return ""
	}
	return args.Bases[args.CurrentBase]
}

// CycleBase cycles to the next base
func (s *Session) CycleBase() int {
	args := s.diffArgs
	if len(args.Bases) == 0 {
		return 0
	}
	newIndex := (args.CurrentBase + 1) % len(args.Bases)
	s.SetCurrentBase(newIndex)
	return newIndex
}

// --- Diff Data ---

// SetDiff sets the fileDiffs data
func (s *Session) SetDiff(diff []*types.FileDiff) {
	s.mu.Lock()
	s.fileDiffs = diff
	s.mu.Unlock()
}

// GetDiff returns the current diff
func (s *Session) GetDiff() []*types.FileDiff {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.fileDiffs
}

// GetFileCount returns the number of files in the fileDiffs
func (s *Session) GetFileCount() int {
	return len(s.GetDiff())
}

func (s *Session) GetFileFromDiff(filePath string) *types.FileDiff {
	files := s.GetDiff()
	if files == nil {
		return nil
	}

	for _, file := range files {
		if file.GetPath() == filePath {
			return file
		}
	}

	return nil
}

// --- Conversations ---

// GetConversationsForFile returns conversations for a specific file from the messaging interface
func (s *Session) GetConversationsForFile(filePath string) ([]*critic.Conversation, error) {
	return s.messaging.GetConversationsForFile(filePath)
}

// GetConversationSummary returns the conversation summary for a file
func (s *Session) GetConversationSummary(filePath string) (*critic.FileConversationSummary, error) {
	return s.messaging.GetFileConversationSummary(filePath)
}

// RefreshConversations refreshes conversation data for all files in the fileDiffs
func (s *Session) RefreshConversations() error {
	// Conversations are fetched directly from messaging when needed,
	// so this is a no-op now. Keeping the method for API compatibility.
	return nil
}

// Package session provides a data structure for managing application state
// including fileDiffs arguments, files, conversations, and watchers for changes.
package session

import (
	"sync"

	"github.com/radiospiel/critic/simple-go/logger"
	"github.com/radiospiel/critic/simple-go/preconditions"
	"github.com/radiospiel/critic/simple-go/utils"
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
	diffArgs      DiffArgs
	resolvedBases map[string]string
	fileDiffs     []*types.FileDiff

	// TUI state
	selectedFilePath string
	focusedPane      string
	filterMode       FilterMode

	// Watchers
	dbWatcher  *DBWatcher
	gitWatcher *GitWatcher
}

// NewSession creates a new Session with the given parameters.
// messaging must not be nil - use critic.DummyMessaging{} for testing.
func NewSession(gitRoot string, messaging critic.Messaging, args DiffArgs) (*Session, error) {
	preconditions.Check(messaging != nil, "messaging must not be nil")

	logger.Warn("*** NewSession: created session w/gitRoot: %v", gitRoot)

	s := &Session{
		messaging:        messaging,
		gitRoot:          gitRoot,
		diffArgs:         DiffArgs{},
		resolvedBases:    make(map[string]string),
		focusedPane:      "fileList",
		filterMode:       FilterModeNone,
		selectedFilePath: "",
	}

	// Create watchers
	dbWatcher, err := NewDBWatcher(gitRoot, func() {
		logger.Info("Session: DB changed, refreshing conversations")
		if err := s.RefreshConversations(); err != nil {
			logger.Warn("Session: Failed to refresh conversations: %v", err)
		}
	})
	if err != nil {
		return nil, err
	}
	s.dbWatcher = dbWatcher

	gitWatcher := NewGitWatcher(s)
	gitWatcher.SetBases(args.Bases)
	s.gitWatcher = gitWatcher

	// Set initial fileDiffs args
	if len(args.Bases) > 0 {
		s.SetDiffArgs(args)
	}

	return s, nil
}

// --- Diff Args ---

// SetDiffArgs sets the fileDiffs arguments
func (s *Session) SetDiffArgs(args DiffArgs) {
	s.mu.Lock()
	s.diffArgs = args
	if s.gitWatcher != nil {
		s.gitWatcher.SetBases(args.Bases)
	}
	s.mu.Unlock()
}

// GetDiffArgs returns the current fileDiffs arguments
func (s *Session) GetDiffArgs() DiffArgs {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.diffArgs
}

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
	args := s.GetDiffArgs()
	if args.CurrentBase < 0 || args.CurrentBase >= len(args.Bases) {
		return ""
	}
	return args.Bases[args.CurrentBase]
}

// CycleBase cycles to the next base
func (s *Session) CycleBase() int {
	args := s.GetDiffArgs()
	if len(args.Bases) == 0 {
		return 0
	}
	newIndex := (args.CurrentBase + 1) % len(args.Bases)
	s.SetCurrentBase(newIndex)
	return newIndex
}

// --- Resolved Bases ---

// SetResolvedBases sets the resolved base refs
func (s *Session) SetResolvedBases(resolved map[string]string) {
	s.mu.Lock()
	s.resolvedBases = resolved
	s.mu.Unlock()
}

// GetResolvedBase returns the resolved SHA for a base ref
func (s *Session) GetResolvedBase(baseRef string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.resolvedBases == nil {
		return "", false
	}
	sha, ok := s.resolvedBases[baseRef]
	return sha, ok
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

// --- Selection ---

// SetSelectedFile sets the selected file by index
func (s *Session) SetSelectedFile(filePath string) {
	s.SetSelectedFilePath(filePath)
}

// SetSelectedFilePath sets the selected file by path
func (s *Session) SetSelectedFilePath(path string) {
	files := s.GetDiff()
	if files == nil {
		return
	}

	for _, file := range files {
		if file.GetPath() == path {
			s.mu.Lock()
			s.selectedFilePath = path
			s.mu.Unlock()
			return
		}
	}
}

// GetSelectedFilePath returns the path of the selected file
func (s *Session) GetSelectedFilePath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.selectedFilePath
}

// GetSelectedFile returns the selected file fileDiffs
func (s *Session) GetSelectedFile() *types.FileDiff {
	s.mu.RLock()
	filePath := s.selectedFilePath
	s.mu.RUnlock()
	return s.GetFileFromDiff(filePath)
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

func (s *Session) getSelectedFilePath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.selectedFilePath
}

// GetSelectedFileIndex returns the index of the selected file (-1 if no selection)
func getSelectedFileIndex(s *Session) int {
	files := s.GetDiff()
	if files == nil {
		return -1
	}

	filePath := s.getSelectedFilePath()
	for index, file := range files {
		if file.GetPath() == filePath {
			return index
		}
	}

	return -1
}

func (s *Session) moveFileSelection(offset int) bool {
	files := s.GetDiff()
	if files == nil {
		return false
	}

	oldIndex := getSelectedFileIndex(s)
	newIndex := oldIndex + offset
	newIndex = utils.Clamp(newIndex, 0, s.GetFileCount()-1)

	if newIndex == oldIndex {
		return false
	}

	s.SetSelectedFilePath(files[newIndex].GetPath())
	return true
}

// SelectNextFile selects the next file
func (s *Session) SelectNextFile() bool {
	return s.moveFileSelection(1)
}

// SelectPrevFile selects the previous file
func (s *Session) SelectPrevFile() bool {
	return s.moveFileSelection(-1)
}

// --- Focus ---

// SetFocusedPane sets the focused pane ("fileList" or "diffView")
func (s *Session) SetFocusedPane(pane string) {
	s.mu.Lock()
	s.focusedPane = pane
	s.mu.Unlock()
}

// GetFocusedPane returns the focused pane
func (s *Session) GetFocusedPane() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.focusedPane
}

// ToggleFocus toggles focus between file list and fileDiffs view
func (s *Session) ToggleFocus() {
	if s.GetFocusedPane() == "fileList" {
		s.SetFocusedPane("diffView")
	} else {
		s.SetFocusedPane("fileList")
	}
}

// --- Filter ---

// SetFilterMode sets the filter mode
func (s *Session) SetFilterMode(mode FilterMode) {
	s.mu.Lock()
	s.filterMode = mode
	s.mu.Unlock()
}

// GetFilterMode returns the current filter mode
func (s *Session) GetFilterMode() FilterMode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.filterMode
}

// CycleFilterMode cycles through filter modes
func (s *Session) CycleFilterMode() FilterMode {
	mode := (s.GetFilterMode() + 1) % 3
	s.SetFilterMode(mode)
	return mode
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

// --- Watchers ---

// StartWatchers starts all watchers in goroutines
func (s *Session) StartWatchers() error {
	var wg sync.WaitGroup
	var dbErr, gitErr error

	if s.dbWatcher != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.dbWatcher.Start(); err != nil {
				dbErr = err
			}
		}()
	}

	if s.gitWatcher != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.gitWatcher.Start(); err != nil {
				gitErr = err
			}
		}()
	}

	wg.Wait()

	if dbErr != nil {
		return dbErr
	}
	if gitErr != nil {
		return gitErr
	}

	return nil
}

// StopWatchers stops all watchers
func (s *Session) StopWatchers() {
	if s.dbWatcher != nil {
		s.dbWatcher.Stop()
	}
	if s.gitWatcher != nil {
		s.gitWatcher.Stop()
	}
}

// Close cleans up resources
func (s *Session) Close() error {
	s.StopWatchers()
	return nil
}

// Package session provides an observable data structure for managing application state
// including diff arguments, files, conversations, and watchers for changes.
package session

import (
	"sync"

	"git.15b.it/eno/critic/pkg/critic"
	"git.15b.it/eno/critic/pkg/types"
	"git.15b.it/eno/critic/simple-go/logger"
	"git.15b.it/eno/critic/simple-go/observable"
)

// State keys for the observable data structure
const (
	// Diff arguments
	KeyDiffArgs       = "diffArgs"
	KeyBasesRefs      = "diffArgs.bases"       // []string - list of base refs (e.g., ["main", "origin/main", "HEAD"])
	KeyCurrentBase    = "diffArgs.currentBase" // int - index of current base
	KeyPaths          = "diffArgs.paths"       // []string - file path patterns to diff
	KeyExtensions     = "diffArgs.extensions"  // []string - file extensions to include

	// Resolved git refs
	KeyResolvedBases = "resolvedBases" // map[string]string - base ref -> resolved SHA

	// Diff data
	KeyDiff  = "diff"       // *types.Diff - the parsed diff
	KeyFiles = "diff.files" // []*types.FileDiff - list of files in the diff

	// Selection
	KeySelectedFileIndex = "selection.fileIndex"  // int - index of currently selected file
	KeySelectedFilePath  = "selection.filePath"   // string - path of currently selected file
	KeyFocusedPane       = "selection.focusedPane" // string - "fileList" or "diffView"

	// Conversations
	KeyConversations         = "conversations"         // map[string][]*critic.Conversation - file path -> conversations
	KeyConversationSummaries = "conversationSummaries" // map[string]*critic.FileConversationSummary

	// Filter
	KeyFilterMode = "filterMode" // int - current filter mode (0=None, 1=WithComments, 2=WithUnresolved)
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

// DiffArgs holds the arguments for generating a diff
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

// Session manages the observable application state for a review session.
// Session embeds *observable.Observable - users can subscribe to state changes
// using OnKeyChange with the exported Key* constants directly on the Session.
type Session struct {
	*observable.Observable
	messaging critic.Messaging
	gitRoot   string
	mu        sync.RWMutex

	// Direct state (not stored in observable since it's not map/slice)
	diff *types.Diff

	// Watchers
	dbWatcher  *DBWatcher
	gitWatcher *GitWatcher

	// Processor
	processor *DiffProcessor

	// Internal subscriptions (for cleanup)
	internalSubs []observable.Subscription
}

// NewSession creates a new Session with the given parameters
func NewSession(gitRoot string, messaging critic.Messaging, args DiffArgs) (*Session, error) {
	s := &Session{
		Observable:   observable.New(),
		messaging:    messaging,
		gitRoot:      gitRoot,
		internalSubs: make([]observable.Subscription, 0),
	}

	// Initialize with default values
	s.SetValueAtKey(KeyDiffArgs, map[string]any{
		"bases":       []any{},
		"currentBase": 0,
		"paths":       []any{},
		"extensions":  []any{},
	})
	s.SetValueAtKey(KeyResolvedBases, map[string]any{})
	s.SetValueAtKey(KeyFilterMode, int(FilterModeNone))
	s.SetValueAtKey(KeySelectedFileIndex, 0)
	s.SetValueAtKey(KeySelectedFilePath, "")
	s.SetValueAtKey(KeyFocusedPane, "fileList")
	s.SetValueAtKey(KeyConversations, map[string]any{})
	s.SetValueAtKey(KeyConversationSummaries, map[string]any{})

	// Create processor
	s.processor = NewDiffProcessor(s)

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
	gitWatcher.OnBasesChanged(func() {
		logger.Info("Session: Git bases changed, loading diff")
		s.processor.LoadDiff()
	})
	s.gitWatcher = gitWatcher

	// Wire up internal state change subscriptions
	// When diff args change, load diff
	diffArgsSubs := s.OnKeyChange([]string{KeyDiffArgs, KeyCurrentBase}, func(key string, oldValue, newValue any) {
		logger.Info("Session: DiffArgs changed (%s), loading diff", key)
		s.processor.LoadDiff()
	})
	s.internalSubs = append(s.internalSubs, diffArgsSubs...)

	// When selection changes, load selected file
	selectionSubs := s.OnKeyChange([]string{KeySelectedFileIndex}, func(key string, oldValue, newValue any) {
		filePath := s.GetSelectedFilePath()
		fileIndex := s.GetSelectedFileIndex()
		logger.Info("Session: Selection changed to %s (index %d)", filePath, fileIndex)
		s.processor.LoadSelectedFile()
	})
	s.internalSubs = append(s.internalSubs, selectionSubs...)

	// Set initial diff args
	if len(args.Bases) > 0 {
		s.SetDiffArgs(args)
	}

	return s, nil
}


// --- Diff Args ---

// SetDiffArgs sets the diff arguments
func (s *Session) SetDiffArgs(args DiffArgs) {
	// Convert to observable-compatible types
	bases := make([]any, len(args.Bases))
	for i, b := range args.Bases {
		bases[i] = b
	}
	paths := make([]any, len(args.Paths))
	for i, p := range args.Paths {
		paths[i] = p
	}
	extensions := make([]any, len(args.Extensions))
	for i, e := range args.Extensions {
		extensions[i] = e
	}

	// Update git watcher bases (protected by its own mutex)
	s.mu.Lock()
	if s.gitWatcher != nil {
		s.gitWatcher.SetBases(args.Bases)
	}
	s.mu.Unlock()

	// Set values without holding the lock - observable has its own internal mutex
	// and subscriptions may need to access session state
	s.SetValueAtKey(KeyDiffArgs, map[string]any{
		"bases":       bases,
		"currentBase": args.CurrentBase,
		"paths":       paths,
		"extensions":  extensions,
	})
}

// GetDiffArgs returns the current diff arguments
func (s *Session) GetDiffArgs() DiffArgs {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return observable.GetValueAs[DiffArgs](s.Observable, KeyDiffArgs)
}

// SetCurrentBase sets the current base index
func (s *Session) SetCurrentBase(index int) {
	// Set value without holding the lock - observable has its own internal mutex
	// and subscriptions may need to access session state
	s.SetValueAtKey(KeyCurrentBase, index)
}

// GetCurrentBase returns the current base index
func (s *Session) GetCurrentBase() int {
	return observable.GetValueAs[int](s.Observable, KeyCurrentBase)
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
	// Convert to observable-compatible type
	bases := make(map[string]any, len(resolved))
	for k, v := range resolved {
		bases[k] = v
	}

	// Set value without holding the lock - observable has its own internal mutex
	s.SetValueAtKey(KeyResolvedBases, bases)
}

// GetResolvedBase returns the resolved SHA for a base ref
func (s *Session) GetResolvedBase(baseRef string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resolved := observable.GetValueAs[map[string]any](s.Observable, KeyResolvedBases)
	if resolved == nil {
		return "", false
	}
	sha, ok := resolved[baseRef].(string)
	return sha, ok
}

// --- Diff Data ---

// SetDiff sets the diff data
func (s *Session) SetDiff(diff *types.Diff) {
	// Store diff in direct field (not observable since it's a struct pointer)
	s.mu.Lock()
	s.diff = diff
	s.mu.Unlock()

	// Prepare file info for observable
	var files []any
	if diff == nil {
		files = []any{}
	} else {
		files = make([]any, len(diff.Files))
		for i, f := range diff.Files {
			files[i] = map[string]any{
				"oldPath":   f.OldPath,
				"newPath":   f.NewPath,
				"isNew":     f.IsNew,
				"isDeleted": f.IsDeleted,
				"isRenamed": f.IsRenamed,
				"isBinary":  f.IsBinary,
			}
		}
	}

	// Set value without holding the lock - observable has its own internal mutex
	// and subscriptions may need to access session state
	s.SetValueAtKey(KeyFiles, files)
}

// GetDiff returns the current diff
func (s *Session) GetDiff() *types.Diff {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.diff
}

// GetFiles returns the list of file diffs
func (s *Session) GetFiles() []*types.FileDiff {
	diff := s.GetDiff()
	if diff == nil {
		return nil
	}
	return diff.Files
}

// GetFileCount returns the number of files in the diff
func (s *Session) GetFileCount() int {
	diff := s.GetDiff()
	if diff == nil {
		return 0
	}
	return len(diff.Files)
}

// --- Selection ---

// SetSelectedFile sets the selected file by index
func (s *Session) SetSelectedFile(index int) {
	// Get file path while holding the lock
	s.mu.RLock()
	var filePath string
	if s.diff != nil && index >= 0 && index < len(s.diff.Files) {
		f := s.diff.Files[index]
		filePath = f.NewPath
		if f.IsDeleted {
			filePath = f.OldPath
		}
	}
	s.mu.RUnlock()

	// Set values without holding the lock - observable has its own internal mutex
	// and subscriptions may need to access session state
	s.SetValueAtKey(KeySelectedFilePath, filePath)
	s.SetValueAtKey(KeySelectedFileIndex, index) // Set index last to trigger subscription after path is set
}

// SetSelectedFilePath sets the selected file by path
func (s *Session) SetSelectedFilePath(path string) {
	diff := s.GetDiff()
	if diff == nil {
		return
	}

	for i, f := range diff.Files {
		fp := f.NewPath
		if f.IsDeleted {
			fp = f.OldPath
		}
		if fp == path {
			s.SetSelectedFile(i)
			return
		}
	}
}

// GetSelectedFileIndex returns the index of the selected file
func (s *Session) GetSelectedFileIndex() int {
	return observable.GetValueAs[int](s.Observable, KeySelectedFileIndex)
}

// GetSelectedFilePath returns the path of the selected file
func (s *Session) GetSelectedFilePath() string {
	return observable.GetValueAs[string](s.Observable, KeySelectedFilePath)
}

// GetSelectedFile returns the selected file diff
func (s *Session) GetSelectedFile() *types.FileDiff {
	diff := s.GetDiff()
	if diff == nil {
		return nil
	}
	index := s.GetSelectedFileIndex()
	if index < 0 || index >= len(diff.Files) {
		return nil
	}
	return diff.Files[index]
}

// SelectNextFile selects the next file
func (s *Session) SelectNextFile() bool {
	index := s.GetSelectedFileIndex()
	count := s.GetFileCount()
	if index < count-1 {
		s.SetSelectedFile(index + 1)
		return true
	}
	return false
}

// SelectPrevFile selects the previous file
func (s *Session) SelectPrevFile() bool {
	index := s.GetSelectedFileIndex()
	if index > 0 {
		s.SetSelectedFile(index - 1)
		return true
	}
	return false
}

// --- Focus ---

// SetFocusedPane sets the focused pane ("fileList" or "diffView")
func (s *Session) SetFocusedPane(pane string) {
	s.SetValueAtKey(KeyFocusedPane, pane)
}

// GetFocusedPane returns the focused pane
func (s *Session) GetFocusedPane() string {
	return observable.GetValueAs[string](s.Observable, KeyFocusedPane)
}

// ToggleFocus toggles focus between file list and diff view
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
	s.SetValueAtKey(KeyFilterMode, int(mode))
}

// GetFilterMode returns the current filter mode
func (s *Session) GetFilterMode() FilterMode {
	return FilterMode(observable.GetValueAs[int](s.Observable, KeyFilterMode))
}

// CycleFilterMode cycles through filter modes
func (s *Session) CycleFilterMode() FilterMode {
	mode := (s.GetFilterMode() + 1) % 3
	s.SetFilterMode(mode)
	return mode
}

// --- Conversations ---

// SetConversationsForFile sets the conversations for a specific file
func (s *Session) SetConversationsForFile(filePath string, conversations []*critic.Conversation) {
	key := KeyConversations + "." + filePath

	// Convert to observable-compatible type
	convs := make([]any, len(conversations))
	for i, c := range conversations {
		convs[i] = map[string]any{
			"uuid":       c.UUID,
			"status":     string(c.Status),
			"filePath":   c.FilePath,
			"lineNumber": c.LineNumber,
			"readByAI":   c.ReadByAI,
		}
	}

	// Set value without holding the lock - observable has its own internal mutex
	s.SetValueAtKey(key, convs)
}

// GetConversationsForFile returns conversations for a specific file from the messaging interface
func (s *Session) GetConversationsForFile(filePath string) ([]*critic.Conversation, error) {
	if s.messaging == nil {
		return nil, nil
	}
	return s.messaging.GetConversationsForFile(filePath)
}

// SetConversationSummary sets the conversation summary for a file
func (s *Session) SetConversationSummary(filePath string, summary *critic.FileConversationSummary) {
	key := KeyConversationSummaries + "." + filePath

	// Prepare value
	var value any
	if summary == nil {
		value = nil
	} else {
		value = map[string]any{
			"filePath":              summary.FilePath,
			"hasUnresolvedComments": summary.HasUnresolvedComments,
			"hasResolvedComments":   summary.HasResolvedComments,
			"hasUnreadAIMessages":   summary.HasUnreadAIMessages,
		}
	}

	// Set value without holding the lock - observable has its own internal mutex
	s.SetValueAtKey(key, value)
}

// GetConversationSummary returns the conversation summary for a file
func (s *Session) GetConversationSummary(filePath string) (*critic.FileConversationSummary, error) {
	if s.messaging == nil {
		return nil, nil
	}
	return s.messaging.GetFileConversationSummary(filePath)
}

// RefreshConversations refreshes conversation data for all files in the diff
func (s *Session) RefreshConversations() error {
	diff := s.GetDiff()
	if diff == nil || s.messaging == nil {
		return nil
	}

	for _, file := range diff.Files {
		filePath := file.NewPath
		if file.IsDeleted {
			filePath = file.OldPath
		}

		conversations, err := s.messaging.GetConversationsForFile(filePath)
		if err != nil {
			return err
		}
		s.SetConversationsForFile(filePath, conversations)

		summary, err := s.messaging.GetFileConversationSummary(filePath)
		if err != nil {
			return err
		}
		s.SetConversationSummary(filePath, summary)
	}

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
	// Clean up internal subscriptions
	if len(s.internalSubs) > 0 {
		s.ClearSubscriptions(s.internalSubs...)
		s.internalSubs = nil
	}
	return nil
}

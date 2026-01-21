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

// Session manages the observable application state for a review session
type Session struct {
	obs       *observable.Observable
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

	// Callbacks for state changes
	onDiffArgsChanged      func()
	onDiffLoaded           func(*types.Diff)
	onSelectionChanged     func(filePath string, fileIndex int)
	onConversationsChanged func(filePath string)
}

// NewSession creates a new Session with the given parameters
func NewSession(gitRoot string, messaging critic.Messaging, args DiffArgs) (*Session, error) {
	s := &Session{
		obs:       observable.New(),
		messaging: messaging,
		gitRoot:   gitRoot,
	}

	// Initialize with default values
	s.obs.SetValueAtKey(KeyDiffArgs, map[string]any{
		"bases":       []any{},
		"currentBase": 0,
		"paths":       []any{},
		"extensions":  []any{},
	})
	s.obs.SetValueAtKey(KeyResolvedBases, map[string]any{})
	s.obs.SetValueAtKey(KeyFilterMode, int(FilterModeNone))
	s.obs.SetValueAtKey(KeySelectedFileIndex, 0)
	s.obs.SetValueAtKey(KeySelectedFilePath, "")
	s.obs.SetValueAtKey(KeyFocusedPane, "fileList")
	s.obs.SetValueAtKey(KeyConversations, map[string]any{})
	s.obs.SetValueAtKey(KeyConversationSummaries, map[string]any{})

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

	// Wire up state change callbacks to processor
	s.OnDiffArgsChanged(func() {
		logger.Info("Session: DiffArgs changed, loading diff")
		s.processor.LoadDiff()
	})

	s.OnSelectionChanged(func(filePath string, fileIndex int) {
		logger.Info("Session: Selection changed to %s (index %d)", filePath, fileIndex)
		s.processor.LoadSelectedFile()
	})

	// Set initial diff args
	if len(args.Bases) > 0 {
		s.SetDiffArgs(args)
	}

	return s, nil
}

// Observable returns the underlying observable for subscription purposes
func (s *Session) Observable() *observable.Observable {
	return s.obs
}

// Processor returns the diff processor
func (s *Session) Processor() *DiffProcessor {
	return s.processor
}

// GitRoot returns the git root directory
func (s *Session) GitRoot() string {
	return s.gitRoot
}

// --- Diff Args ---

// SetDiffArgs sets the diff arguments
func (s *Session) SetDiffArgs(args DiffArgs) {
	s.mu.Lock()

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

	s.obs.SetValueAtKey(KeyDiffArgs, map[string]any{
		"bases":       bases,
		"currentBase": args.CurrentBase,
		"paths":       paths,
		"extensions":  extensions,
	})

	// Update git watcher bases
	if s.gitWatcher != nil {
		s.gitWatcher.SetBases(args.Bases)
	}

	callback := s.onDiffArgsChanged
	s.mu.Unlock()

	// Call callback outside lock to avoid deadlock
	if callback != nil {
		callback()
	}
}

// GetDiffArgs returns the current diff arguments
func (s *Session) GetDiffArgs() DiffArgs {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return observable.GetValueAs[DiffArgs](s.obs, KeyDiffArgs)
}

// SetCurrentBase sets the current base index
func (s *Session) SetCurrentBase(index int) {
	s.mu.Lock()

	s.obs.SetValueAtKey(KeyCurrentBase, index)

	callback := s.onDiffArgsChanged
	s.mu.Unlock()

	// Call callback outside lock to avoid deadlock
	if callback != nil {
		callback()
	}
}

// GetCurrentBase returns the current base index
func (s *Session) GetCurrentBase() int {
	return s.obs.GetInt(KeyCurrentBase)
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
	defer s.mu.Unlock()

	// Convert to observable-compatible type
	bases := make(map[string]any, len(resolved))
	for k, v := range resolved {
		bases[k] = v
	}
	s.obs.SetValueAtKey(KeyResolvedBases, bases)
}

// GetResolvedBase returns the resolved SHA for a base ref
func (s *Session) GetResolvedBase(baseRef string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resolved := s.obs.GetMap(KeyResolvedBases)
	if resolved == nil {
		return "", false
	}
	sha, ok := resolved[baseRef].(string)
	return sha, ok
}

// --- Diff Data ---

// SetDiff sets the diff data
func (s *Session) SetDiff(diff *types.Diff) {
	s.mu.Lock()

	// Store diff in direct field (not observable since it's a struct pointer)
	s.diff = diff

	if diff == nil {
		s.obs.SetValueAtKey(KeyFiles, []any{})
	} else {
		// Store file info for observable tracking and subscriptions
		files := make([]any, len(diff.Files))
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
		s.obs.SetValueAtKey(KeyFiles, files)
	}

	callback := s.onDiffLoaded
	s.mu.Unlock()

	// Call callback outside lock to avoid deadlock
	if callback != nil {
		callback(diff)
	}
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
	s.mu.Lock()

	s.obs.SetValueAtKey(KeySelectedFileIndex, index)

	// Update the file path (access diff directly since we hold the lock)
	var filePath string
	if s.diff != nil && index >= 0 && index < len(s.diff.Files) {
		f := s.diff.Files[index]
		filePath = f.NewPath
		if f.IsDeleted {
			filePath = f.OldPath
		}
	}
	s.obs.SetValueAtKey(KeySelectedFilePath, filePath)

	callback := s.onSelectionChanged
	s.mu.Unlock()

	// Call callback outside lock to avoid deadlock
	if callback != nil {
		callback(filePath, index)
	}
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
	return s.obs.GetInt(KeySelectedFileIndex)
}

// GetSelectedFilePath returns the path of the selected file
func (s *Session) GetSelectedFilePath() string {
	return s.obs.GetString(KeySelectedFilePath)
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
	s.obs.SetValueAtKey(KeyFocusedPane, pane)
}

// GetFocusedPane returns the focused pane
func (s *Session) GetFocusedPane() string {
	return s.obs.GetString(KeyFocusedPane)
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
	s.obs.SetValueAtKey(KeyFilterMode, int(mode))
}

// GetFilterMode returns the current filter mode
func (s *Session) GetFilterMode() FilterMode {
	return FilterMode(s.obs.GetInt(KeyFilterMode))
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
	s.mu.Lock()

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
	s.obs.SetValueAtKey(key, convs)

	callback := s.onConversationsChanged
	s.mu.Unlock()

	// Call callback outside lock to avoid deadlock
	if callback != nil {
		callback(filePath)
	}
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
	s.mu.Lock()
	defer s.mu.Unlock()

	key := KeyConversationSummaries + "." + filePath
	if summary == nil {
		s.obs.SetValueAtKey(key, nil)
	} else {
		s.obs.SetValueAtKey(key, map[string]any{
			"filePath":              summary.FilePath,
			"hasUnresolvedComments": summary.HasUnresolvedComments,
			"hasResolvedComments":   summary.HasResolvedComments,
			"hasUnreadAIMessages":   summary.HasUnreadAIMessages,
		})
	}
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

// --- Callbacks ---

// OnDiffArgsChanged sets the callback for when diff args change
func (s *Session) OnDiffArgsChanged(callback func()) {
	s.onDiffArgsChanged = callback
}

// OnDiffLoaded sets the callback for when diff is loaded
func (s *Session) OnDiffLoaded(callback func(*types.Diff)) {
	s.onDiffLoaded = callback
}

// OnSelectionChanged sets the callback for when selection changes
func (s *Session) OnSelectionChanged(callback func(filePath string, fileIndex int)) {
	s.onSelectionChanged = callback
}

// OnConversationsChanged sets the callback for when conversations change
func (s *Session) OnConversationsChanged(callback func(filePath string)) {
	s.onConversationsChanged = callback
}

// --- Subscriptions ---

// Subscribe registers a callback for changes at the given key patterns
func (s *Session) Subscribe(patterns []string, callback observable.ChangeCallback) []observable.Subscription {
	return s.obs.OnKeyChange(patterns, callback)
}

// Unsubscribe removes the specified subscriptions
func (s *Session) Unsubscribe(subs ...observable.Subscription) {
	s.obs.ClearSubscriptions(subs...)
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

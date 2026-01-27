package session

import (
	"fmt"
	"sync"

	"github.org/radiospiel/critic/src/git"
	"github.org/radiospiel/critic/src/pkg/types"
	"github.org/radiospiel/critic/simple-go/logger"
	"github.org/radiospiel/critic/simple-go/observable"
)

// DiffProcessor handles loading and processing diffs
//
// It subscribes to
type DiffProcessor struct {
	state *Session
	mu    sync.Mutex

	// Processing state
	loading bool

	// Callbacks
	onDiffLoaded func(diff *types.Diff, err error)
}

// NewDiffProcessor creates a new diff diffProcessor
func NewDiffProcessor(state *Session) *DiffProcessor {
	diffProcessor := &DiffProcessor{
		state: state,
	}

	state.OnKeyChange(Keys.SelectedFilePath, func(key string) {
		selectedFilePath :=
			observable.GetValueAs[string](state.Observable, Keys.SelectedFilePath)
		diffProcessor.loadSelectedFile(selectedFilePath)
	})

	return diffProcessor
}

// LoadDiff loads the diff based on current state
func (p *DiffProcessor) LoadDiff() error {
	p.mu.Lock()
	if p.loading {
		p.mu.Unlock()
		return nil
	}
	p.loading = true
	p.mu.Unlock()

	go p.loadDiffAsync()
	return nil
}

// loadDiffAsync loads the diff asynchronously
func (p *DiffProcessor) loadDiffAsync() {
	defer func() {
		p.mu.Lock()
		p.loading = false
		p.mu.Unlock()
	}()

	args := p.state.GetDiffArgs()
	if len(args.Bases) == 0 {
		p.notifyDiffLoaded(nil, fmt.Errorf("no bases configured"))
		return
	}

	if args.CurrentBase < 0 || args.CurrentBase >= len(args.Bases) {
		p.notifyDiffLoaded(nil, fmt.Errorf("invalid current base index: %d", args.CurrentBase))
		return
	}

	baseName := args.Bases[args.CurrentBase]

	// Get resolved SHA for the base
	baseSHA, ok := p.state.GetResolvedBase(baseName)
	if !ok {
		// Try to resolve it now
		var err error
		baseSHA, err = resolveBaseRef(baseName)
		if err != nil {
			p.notifyDiffLoaded(nil, fmt.Errorf("failed to resolve base %s: %w", baseName, err))
			return
		}
	}

	// Resolve HEAD
	targetSHA, err := git.ResolveRef("HEAD")
	if err != nil {
		p.notifyDiffLoaded(nil, fmt.Errorf("failed to resolve HEAD: %w", err))
		return
	}

	logger.Info("DiffProcessor: Loading diff from %s (%s) to HEAD (%s)", baseName, truncateSHA(baseSHA), truncateSHA(targetSHA))

	// Get the diff
	diff, err := git.GetDiffBetween(baseSHA, targetSHA, args.Paths)
	if err != nil {
		p.notifyDiffLoaded(nil, fmt.Errorf("failed to get diff: %w", err))
		return
	}

	// Filter files by extension if specified
	if len(args.Extensions) > 0 {
		diff.Files = filterFilesByExtension(diff.Files, args.Extensions)
	}

	logger.Info("DiffProcessor: Loaded diff with %d files", len(diff.Files))

	// Update state
	p.state.SetDiff(diff)

	// Refresh conversations for all files
	if err := p.state.RefreshConversations(); err != nil {
		logger.Warn("DiffProcessor: Failed to refresh conversations: %v", err)
	}

	p.notifyDiffLoaded(diff, nil)
}

// loadSelectedFile loads/parses the currently selected file
func (p *DiffProcessor) loadSelectedFile(filePath string) error {
	conversations, err := p.state.GetConversationsForFile(filePath)
	if err != nil {
		logger.Warn("DiffProcessor: Failed to get conversations for %s: %v", filePath, err)
	} else {
		p.state.SetConversationsForFile(filePath, conversations)
	}

	summary, err := p.state.GetConversationSummary(filePath)
	if err != nil {
		logger.Warn("DiffProcessor: Failed to get summary for %s: %v", filePath, err)
	} else {
		p.state.SetConversationSummary(filePath, summary)
	}

	return nil
}

// notifyDiffLoaded calls the diff loaded callback
func (p *DiffProcessor) notifyDiffLoaded(diff *types.Diff, err error) {
	p.mu.Lock()
	callback := p.onDiffLoaded
	p.mu.Unlock()

	if callback != nil {
		callback(diff, err)
	}
}

// IsLoading returns whether a diff is currently loading
func (p *DiffProcessor) IsLoading() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.loading
}

// resolveBaseRef resolves a base ref to a SHA
func resolveBaseRef(base string) (string, error) {
	if git.IsCommitSHA(base) {
		return git.ResolveRef(base)
	}

	baseSHA, err := git.ResolveRef(base)
	if err != nil {
		return "", err
	}

	mergeBase, err := git.GetMergeBaseBetween("HEAD", baseSHA)
	if err != nil {
		return baseSHA, nil // Fallback
	}

	return mergeBase, nil
}

// filterFilesByExtension filters files by extension
func filterFilesByExtension(files []*types.FileDiff, extensions []string) []*types.FileDiff {
	if len(extensions) == 0 {
		return files
	}

	extMap := make(map[string]bool)
	for _, ext := range extensions {
		extMap[ext] = true
	}

	var filtered []*types.FileDiff
	for _, file := range files {
		path := file.NewPath
		if file.IsDeleted {
			path = file.OldPath
		}

		// Get extension
		for i := len(path) - 1; i >= 0; i-- {
			if path[i] == '.' {
				ext := path[i+1:]
				if extMap[ext] {
					filtered = append(filtered, file)
				}
				break
			}
		}
	}

	return filtered
}

package session

import (
	"database/sql"
	"sync"
	"testing"
	"time"

	"git.15b.it/eno/critic/internal/messagedb"
	"git.15b.it/eno/critic/pkg/critic"
	"git.15b.it/eno/critic/pkg/types"
	"git.15b.it/eno/critic/simple-go/assert"
	"git.15b.it/eno/critic/simple-go/observable"
	_ "github.com/mattn/go-sqlite3"
)

// mockMessaging implements critic.Messaging for testing
type mockMessaging struct {
	conversations map[string][]*critic.Conversation
	summaries     map[string]*critic.FileConversationSummary
}

func newMockMessaging() *mockMessaging {
	return &mockMessaging{
		conversations: make(map[string][]*critic.Conversation),
		summaries:     make(map[string]*critic.FileConversationSummary),
	}
}

func (m *mockMessaging) GetConversations(status string) ([]critic.Conversation, error) {
	var all []critic.Conversation
	for _, convs := range m.conversations {
		for _, c := range convs {
			all = append(all, *c)
		}
	}
	return all, nil
}

func (m *mockMessaging) GetConversationsByFile(filePath string) ([]critic.Conversation, error) {
	convs := m.conversations[filePath]
	result := make([]critic.Conversation, len(convs))
	for i, c := range convs {
		result[i] = *c
	}
	return result, nil
}

func (m *mockMessaging) GetFullConversation(uuid string) (*critic.Conversation, error) {
	for _, convs := range m.conversations {
		for _, c := range convs {
			if c.UUID == uuid {
				return c, nil
			}
		}
	}
	return nil, nil
}

func (m *mockMessaging) GetConversationsForFile(filePath string) ([]*critic.Conversation, error) {
	return m.conversations[filePath], nil
}

func (m *mockMessaging) GetFileConversationSummary(filePath string) (*critic.FileConversationSummary, error) {
	return m.summaries[filePath], nil
}

func (m *mockMessaging) ReplyToConversation(conversationUUID string, message string, author critic.Author) (*critic.Message, error) {
	return &critic.Message{UUID: "reply-1"}, nil
}

func (m *mockMessaging) CreateConversation(author critic.Author, message, filePath string, lineNumber int, codeVersion string, context string) (*critic.Conversation, error) {
	return &critic.Conversation{UUID: "conv-1"}, nil
}

func (m *mockMessaging) MarkAsResolved(conversationUUID string) error   { return nil }
func (m *mockMessaging) MarkAsUnresolved(conversationUUID string) error { return nil }
func (m *mockMessaging) MarkAsRead(messageUUID string) error            { return nil }
func (m *mockMessaging) MarkAsReadByAI(conversationUUID string) error   { return nil }
func (m *mockMessaging) Close() error                                   { return nil }

// createTestSession creates a Session for testing with a temp directory
func createTestSession(t *testing.T, messaging critic.Messaging) *Session {
	tempDir := t.TempDir()
	session, err := NewSession(tempDir, messaging, DiffArgs{})
	assert.NoError(t, err, "should create session")
	return session
}

func TestNewSession(t *testing.T) {
	tempDir := t.TempDir()
	session, err := NewSession(tempDir, nil, DiffArgs{})
	assert.NoError(t, err, "should create session")
	assert.NotNil(t, session, "session should not be nil")
	assert.NotNil(t, session.Observable, "embedded observable should not be nil")
}

func TestDiffArgs(t *testing.T) {
	session := createTestSession(t, nil)

	// Set diff args
	args := DiffArgs{
		Bases:       []string{"main", "origin/main", "HEAD"},
		CurrentBase: 1,
		Paths:       []string{"internal/"},
		Extensions:  []string{"go"},
	}
	session.SetDiffArgs(args)

	// Get diff args
	retrieved := session.GetDiffArgs()
	assert.Equals(t, len(retrieved.Bases), 3, "should have 3 bases")
	assert.Equals(t, retrieved.CurrentBase, 1, "current base should be 1")
	assert.Equals(t, len(retrieved.Paths), 1, "should have 1 path")
	assert.Equals(t, len(retrieved.Extensions), 1, "should have 1 extension")
}

func TestCurrentBase(t *testing.T) {
	session := createTestSession(t, nil)

	args := DiffArgs{
		Bases:       []string{"main", "origin/main", "HEAD"},
		CurrentBase: 0,
	}
	session.SetDiffArgs(args)

	assert.Equals(t, session.GetCurrentBase(), 0, "initial current base should be 0")
	assert.Equals(t, session.GetCurrentBaseName(), "main", "initial base name should be main")

	// Cycle through bases
	newIndex := session.CycleBase()
	assert.Equals(t, newIndex, 1, "cycled index should be 1")
	assert.Equals(t, session.GetCurrentBaseName(), "origin/main", "base name should be origin/main")

	newIndex = session.CycleBase()
	assert.Equals(t, newIndex, 2, "cycled index should be 2")

	newIndex = session.CycleBase()
	assert.Equals(t, newIndex, 0, "cycled index should wrap to 0")
}

func TestResolvedBases(t *testing.T) {
	session := createTestSession(t, nil)

	resolved := map[string]string{
		"main":        "abc123",
		"origin/main": "def456",
	}
	session.SetResolvedBases(resolved)

	sha, ok := session.GetResolvedBase("main")
	assert.True(t, ok, "should find main")
	assert.Equals(t, sha, "abc123", "main SHA should match")

	sha, ok = session.GetResolvedBase("origin/main")
	assert.True(t, ok, "should find origin/main")
	assert.Equals(t, sha, "def456", "origin/main SHA should match")

	_, ok = session.GetResolvedBase("nonexistent")
	assert.False(t, ok, "should not find nonexistent")
}

func TestDiff(t *testing.T) {
	session := createTestSession(t, nil)

	// Initially nil
	assert.Nil(t, session.GetDiff(), "initial diff should be nil")
	assert.Equals(t, session.GetFileCount(), 0, "initial file count should be 0")

	// Set diff
	diff := &types.Diff{
		Files: []*types.FileDiff{
			{NewPath: "file1.go", OldPath: "file1.go"},
			{NewPath: "file2.go", OldPath: "file2.go"},
		},
	}
	session.SetDiff(diff)

	assert.NotNil(t, session.GetDiff(), "diff should be set")
	assert.Equals(t, session.GetFileCount(), 2, "file count should be 2")

	files := session.GetFiles()
	assert.Equals(t, len(files), 2, "should have 2 files")
	assert.Equals(t, files[0].NewPath, "file1.go", "first file should be file1.go")
}

func TestSelection(t *testing.T) {
	session := createTestSession(t, nil)

	// Set diff first
	diff := &types.Diff{
		Files: []*types.FileDiff{
			{NewPath: "file1.go", OldPath: "file1.go"},
			{NewPath: "file2.go", OldPath: "file2.go"},
			{NewPath: "file3.go", OldPath: "file3.go"},
		},
	}
	session.SetDiff(diff)

	// Initial selection
	assert.Equals(t, session.GetSelectedFileIndex(), 0, "initial index should be 0")

	// Select by index
	session.SetSelectedFile(1)
	assert.Equals(t, session.GetSelectedFileIndex(), 1, "index should be 1")
	assert.Equals(t, session.GetSelectedFilePath(), "file2.go", "path should be file2.go")

	selected := session.GetSelectedFile()
	assert.NotNil(t, selected, "selected file should not be nil")
	assert.Equals(t, selected.NewPath, "file2.go", "selected file should be file2.go")

	// Select next
	ok := session.SelectNextFile()
	assert.True(t, ok, "should select next")
	assert.Equals(t, session.GetSelectedFileIndex(), 2, "index should be 2")

	// Select next at end
	ok = session.SelectNextFile()
	assert.False(t, ok, "should not select next at end")
	assert.Equals(t, session.GetSelectedFileIndex(), 2, "index should still be 2")

	// Select prev
	ok = session.SelectPrevFile()
	assert.True(t, ok, "should select prev")
	assert.Equals(t, session.GetSelectedFileIndex(), 1, "index should be 1")

	// Select by path
	session.SetSelectedFilePath("file1.go")
	assert.Equals(t, session.GetSelectedFileIndex(), 0, "index should be 0")
}

func TestFocus(t *testing.T) {
	session := createTestSession(t, nil)

	// Initial focus
	assert.Equals(t, session.GetFocusedPane(), "fileList", "initial focus should be fileList")

	// Toggle focus
	session.ToggleFocus()
	assert.Equals(t, session.GetFocusedPane(), "diffView", "focus should be diffView")

	session.ToggleFocus()
	assert.Equals(t, session.GetFocusedPane(), "fileList", "focus should be fileList again")

	// Set focus directly
	session.SetFocusedPane("diffView")
	assert.Equals(t, session.GetFocusedPane(), "diffView", "focus should be diffView")
}

func TestFilterMode(t *testing.T) {
	session := createTestSession(t, nil)

	// Initial filter mode
	assert.Equals(t, session.GetFilterMode(), FilterModeNone, "initial filter mode should be None")
	assert.Equals(t, session.GetFilterMode().String(), "All", "filter mode string should be All")

	// Cycle filter mode
	mode := session.CycleFilterMode()
	assert.Equals(t, mode, FilterModeWithComments, "should cycle to WithComments")
	assert.Equals(t, session.GetFilterMode().String(), "With Comments", "filter mode string should match")

	mode = session.CycleFilterMode()
	assert.Equals(t, mode, FilterModeWithUnresolved, "should cycle to WithUnresolved")

	mode = session.CycleFilterMode()
	assert.Equals(t, mode, FilterModeNone, "should cycle back to None")
}

func TestConversations(t *testing.T) {
	messaging := newMockMessaging()
	messaging.conversations["file1.go"] = []*critic.Conversation{
		{UUID: "conv-1", FilePath: "file1.go", LineNumber: 10},
		{UUID: "conv-2", FilePath: "file1.go", LineNumber: 20},
	}
	messaging.summaries["file1.go"] = &critic.FileConversationSummary{
		FilePath:              "file1.go",
		HasUnresolvedComments: true,
	}

	session := createTestSession(t, messaging)

	// Get conversations from messaging
	convs, err := session.GetConversationsForFile("file1.go")
	assert.NoError(t, err, "should get conversations")
	assert.Equals(t, len(convs), 2, "should have 2 conversations")

	// Get summary from messaging
	summary, err := session.GetConversationSummary("file1.go")
	assert.NoError(t, err, "should get summary")
	assert.NotNil(t, summary, "summary should not be nil")
	assert.True(t, summary.HasUnresolvedComments, "should have unresolved comments")
}

func TestOnKeyChange(t *testing.T) {
	session := createTestSession(t, nil)

	// Test subscription to KeyFiles changes (replaces old OnDiffLoaded callback)
	filesChangeCalled := false
	var changedKey string
	subs := session.OnKeyChange(KeyFiles, func(obs *observable.Observable, key string) {
		filesChangeCalled = true
		changedKey = key
	})
	defer session.ClearSubscriptions(subs)

	diff := &types.Diff{Files: []*types.FileDiff{
		{NewPath: "test1.go"},
		{NewPath: "test2.go"},
	}}
	session.SetDiff(diff)
	assert.True(t, filesChangeCalled, "OnKeyChange callback should be called")
	assert.Equals(t, changedKey, KeyFiles, "changed key should be KeyFiles")

	// Test subscription to selection changes
	selectionChangeCalled := false
	selSubs := session.OnKeyChange(KeySelectedFileIndex, func(obs *observable.Observable, key string) {
		selectionChangeCalled = true
	})
	defer session.ClearSubscriptions(selSubs)

	// Select file at index 1 (different from initial 0)
	session.SetSelectedFile(1)
	assert.True(t, selectionChangeCalled, "Selection change callback should be called")
}

func TestSubscriptions(t *testing.T) {
	session := createTestSession(t, nil)

	// Subscribe to filter mode changes using OnKeyChange
	filterChangeCalled := false
	var changedKey string
	subs := session.OnKeyChange(KeyFilterMode, func(obs *observable.Observable, key string) {
		filterChangeCalled = true
		changedKey = key
	})

	session.SetFilterMode(FilterModeWithComments)
	assert.True(t, filterChangeCalled, "filter change callback should be called")
	assert.Equals(t, changedKey, KeyFilterMode, "changed key should match")

	// Unsubscribe using ClearSubscriptions
	filterChangeCalled = false
	session.ClearSubscriptions(subs)

	session.SetFilterMode(FilterModeWithUnresolved)
	assert.False(t, filterChangeCalled, "callback should not be called after unsubscribe")
}

func TestDeletedFileSelection(t *testing.T) {
	session := createTestSession(t, nil)

	// Set diff with a deleted file
	diff := &types.Diff{
		Files: []*types.FileDiff{
			{NewPath: "file1.go", OldPath: "file1.go"},
			{NewPath: "", OldPath: "deleted.go", IsDeleted: true},
		},
	}
	session.SetDiff(diff)

	// Select deleted file
	session.SetSelectedFile(1)
	assert.Equals(t, session.GetSelectedFilePath(), "deleted.go", "should use OldPath for deleted files")
}

func TestFilterModeString(t *testing.T) {
	assert.Equals(t, FilterModeNone.String(), "All", "None should be 'All'")
	assert.Equals(t, FilterModeWithComments.String(), "With Comments", "WithComments should be 'With Comments'")
	assert.Equals(t, FilterModeWithUnresolved.String(), "Unresolved Only", "WithUnresolved should be 'Unresolved Only'")
}

func TestGitWatcher(t *testing.T) {
	session := createTestSession(t, nil)
	watcher := NewGitWatcher(session)

	// Set bases
	watcher.SetBases([]string{"main", "HEAD"})

	// Set callbacks (callback is registered but not called in this test)
	watcher.OnBasesChanged(func() {
		// Would be called when bases change
	})

	// Check IsRunning
	assert.False(t, watcher.IsRunning(), "watcher should not be running initially")

	// Note: We can't fully test Start() without a git repo, but we can test the structure
	assert.NotNil(t, watcher, "watcher should not be nil")
}

func TestDBWatcher(t *testing.T) {
	// Create a temp dir for testing
	tempDir := t.TempDir()

	// Initialize the database schema (creates _db_version table and triggers)
	db, err := messagedb.New(tempDir)
	assert.NoError(t, err, "should create messagedb")
	defer db.Close()

	watcher, err := NewDBWatcher(tempDir, func() {
		// Would be called when DB changes
	})
	assert.NoError(t, err, "should create watcher")
	assert.NotNil(t, watcher, "watcher should not be nil")

	// Set debounce
	watcher.SetDebounceMs(50)

	// Check IsRunning
	assert.False(t, watcher.IsRunning(), "watcher should not be running initially")

	// Start and stop
	err = watcher.Start()
	assert.NoError(t, err, "should start watcher")
	assert.True(t, watcher.IsRunning(), "watcher should be running")

	watcher.Stop()
	time.Sleep(10 * time.Millisecond) // Give it time to stop
	assert.False(t, watcher.IsRunning(), "watcher should not be running after stop")
}

func TestDBWatcherWithTriggers(t *testing.T) {
	// Create a temp dir for testing
	tempDir := t.TempDir()
	dbPath := tempDir + "/.critic.db"

	// Initialize the database schema (creates _db_version table and triggers)
	msgDB, err := messagedb.New(tempDir)
	assert.NoError(t, err, "should create messagedb")
	defer msgDB.Close()

	// Track callback invocations
	var mu sync.Mutex
	callCount := 0
	watcher, err := NewDBWatcher(tempDir, func() {
		mu.Lock()
		callCount++
		mu.Unlock()
	})
	assert.NoError(t, err, "should create watcher")

	// Set fast poll interval for testing
	watcher.SetPollInterval(50 * time.Millisecond)

	// Start watcher (schema with _db_version table and triggers already exists from messagedb)
	err = watcher.Start()
	assert.NoError(t, err, "should start watcher")
	defer watcher.Stop()

	// Open a separate connection to insert data (triggers will fire on insert)
	db, err := sql.Open("sqlite3", dbPath)
	assert.NoError(t, err, "should open db")
	defer db.Close()

	// Insert a message (triggers are already set up by messagedb schema)
	_, err = db.Exec(`
		INSERT INTO messages (id, author, status, read_status, message, file_path, lineno, conversation_id, sha1, created_at, updated_at)
		VALUES ('1', 'human', 'new', 'unread', 'hello', 'test.go', 1, '1', 'abc123', datetime('now'), datetime('now'))
	`)
	assert.NoError(t, err, "should insert message")

	// Wait for poll to detect change
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	count := callCount
	mu.Unlock()

	assert.True(t, count >= 1, "callback should be called at least once, got %d", count)

	// Insert another message
	_, err = db.Exec(`
		INSERT INTO messages (id, author, status, read_status, message, file_path, lineno, conversation_id, sha1, created_at, updated_at)
		VALUES ('2', 'human', 'new', 'unread', 'world', 'test.go', 2, '2', 'def456', datetime('now'), datetime('now'))
	`)
	assert.NoError(t, err, "should insert another message")

	// Wait for poll
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	count = callCount
	mu.Unlock()

	assert.True(t, count >= 2, "callback should be called at least twice, got %d", count)
}

func TestDiffProcessor(t *testing.T) {
	session := createTestSession(t, nil)
	processor := NewDiffProcessor(session)

	assert.NotNil(t, processor, "processor should not be nil")
	assert.False(t, processor.IsLoading(), "processor should not be loading initially")

	// Test callbacks (registered but not called in this test since we don't have a git repo)
	processor.OnDiffLoaded(func(diff *types.Diff, err error) {
		// Would be called when diff loads
	})

	processor.OnFileLoaded(func(file *types.FileDiff, err error) {
		// Would be called when file loads
	})

	// Note: We can't fully test LoadDiff without a git repo
	assert.NotNil(t, processor, "processor should still be valid")
}

func TestStartWatchers(t *testing.T) {
	tempDir := t.TempDir()

	// Initialize the database schema (creates _db_version table and triggers)
	msgDB, err := messagedb.New(tempDir)
	assert.NoError(t, err, "should create messagedb")
	defer msgDB.Close()

	session, err := NewSession(tempDir, nil, DiffArgs{})
	assert.NoError(t, err, "should create session")

	// Start watchers (in goroutines)
	err = session.StartWatchers()
	assert.NoError(t, err, "should start watchers")

	// Give them time to start
	time.Sleep(50 * time.Millisecond)

	// Stop and close
	session.Close()
}

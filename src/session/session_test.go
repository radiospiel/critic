package session

import (
	"database/sql"
	"sync"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/radiospiel/critic/simple-go/assert"
	"github.com/radiospiel/critic/src/messagedb"
	"github.com/radiospiel/critic/src/pkg/critic"
	"github.com/radiospiel/critic/src/pkg/types"
)

// createTestSession creates a Session for testing with a temp directory.
// If messaging is nil, it defaults to DummyMessaging.
func createTestSession(t *testing.T, messaging critic.Messaging) *Session {
	if messaging == nil {
		messaging = &critic.DummyMessaging{}
	}
	tempDir := t.TempDir()
	session, err := NewSession(tempDir, messaging, DiffArgs{})
	assert.NoError(t, err, "should create session")
	return session
}

func TestNewSession(t *testing.T) {
	tempDir := t.TempDir()
	session, err := NewSession(tempDir, &critic.DummyMessaging{}, DiffArgs{})
	assert.NoError(t, err, "should create session")
	assert.NotNil(t, session, "session should not be nil")
}

func TestDiffArgs(t *testing.T) {
	session := createTestSession(t, nil)

	// Set fileDiffs args
	args := DiffArgs{
		Bases:       []string{"main", "origin/main", "HEAD"},
		CurrentBase: 1,
		Paths:       []string{"internal/"},
		Extensions:  []string{"go"},
	}
	session.SetDiffArgs(args)

	// Get fileDiffs args
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
	assert.Nil(t, session.GetDiff(), "initial fileDiffs should be nil")
	assert.Equals(t, session.GetFileCount(), 0, "initial file count should be 0")

	// Set fileDiffs
	diff := []*types.FileDiff{
		{NewPath: "file1.go", OldPath: "file1.go"},
		{NewPath: "file2.go", OldPath: "file2.go"},
	}
	session.SetDiff(diff)

	assert.NotNil(t, session.GetDiff(), "fileDiffs should be set")
	assert.Equals(t, session.GetFileCount(), 2, "file count should be 2")
}

func TestSelection(t *testing.T) {
	session := createTestSession(t, nil)

	// Set fileDiffs first
	diff := []*types.FileDiff{
		{NewPath: "file1.go", OldPath: "file1.go"},
		{NewPath: "file2.go", OldPath: "file2.go"},
		{NewPath: "file3.go", OldPath: "file3.go"},
	}
	session.SetDiff(diff)

	// Initial selection is empty (no file selected yet)
	assert.Equals(t, getSelectedFileIndex(session), -1, "initial index should be -1 (no selection)")

	// Select first file
	session.SetSelectedFile("file1.go")
	assert.Equals(t, getSelectedFileIndex(session), 0, "index should be 0")

	// Select by path
	session.SetSelectedFile("file2.go")
	assert.Equals(t, getSelectedFileIndex(session), 1, "index should be 1")
	assert.Equals(t, session.GetSelectedFilePath(), "file2.go", "path should be file2.go")

	selected := session.GetSelectedFile()
	assert.NotNil(t, selected, "selected file should not be nil")
	assert.Equals(t, selected.NewPath, "file2.go", "selected file should be file2.go")

	// Select next
	ok := session.SelectNextFile()
	assert.True(t, ok, "should select next")
	assert.Equals(t, getSelectedFileIndex(session), 2, "index should be 2")

	// Select next at end
	ok = session.SelectNextFile()
	assert.False(t, ok, "should not select next at end")
	assert.Equals(t, getSelectedFileIndex(session), 2, "index should still be 2")

	// Select prev
	ok = session.SelectPrevFile()
	assert.True(t, ok, "should select prev")
	assert.Equals(t, getSelectedFileIndex(session), 1, "index should be 1")

	// Select by path
	session.SetSelectedFilePath("file1.go")
	assert.Equals(t, getSelectedFileIndex(session), 0, "index should be 0")
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
	messaging := critic.NewDummyMessaging()
	messaging.Conversations["file1.go"] = []*critic.Conversation{
		{UUID: "conv-1", FilePath: "file1.go", LineNumber: 10},
		{UUID: "conv-2", FilePath: "file1.go", LineNumber: 20},
	}
	messaging.Summaries["file1.go"] = &critic.FileConversationSummary{
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


func TestDeletedFileSelection(t *testing.T) {
	session := createTestSession(t, nil)

	// Set fileDiffs with a deleted file
	diff := []*types.FileDiff{
		{NewPath: "file1.go", OldPath: "file1.go"},
		{NewPath: "", OldPath: "deleted.go", FileStatus: types.FileStatusDeleted},
	}
	session.SetDiff(diff)

	// Select deleted file
	session.SetSelectedFile("deleted.go")
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
	dbPath := tempDir + "/.critic/critic.db"

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

	// Set fast debounce interval for testing
	watcher.SetDebounceMs(50)

	// Start watcher (schema with _db_version table and triggers already exists from messagedb)
	err = watcher.Start()
	assert.NoError(t, err, "should start watcher")
	defer watcher.Stop()

	// Give the poll loop goroutine time to start
	time.Sleep(10 * time.Millisecond)

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

	// Wait for poll to detect change (5x poll interval for reliability under load)
	time.Sleep(250 * time.Millisecond)

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

	// Wait for poll (5x poll interval for reliability under load)
	time.Sleep(250 * time.Millisecond)

	mu.Lock()
	count = callCount
	mu.Unlock()

	assert.True(t, count >= 2, "callback should be called at least twice, got %d", count)
}

func TestStartWatchers(t *testing.T) {
	tempDir := t.TempDir()

	// Initialize the database schema (creates _db_version table and triggers)
	msgDB, err := messagedb.New(tempDir)
	assert.NoError(t, err, "should create messagedb")
	defer msgDB.Close()

	session, err := NewSession(tempDir, &critic.DummyMessaging{}, DiffArgs{})
	assert.NoError(t, err, "should create session")

	// Start watchers (in goroutines)
	err = session.StartWatchers()
	assert.NoError(t, err, "should start watchers")

	// Give them time to start
	time.Sleep(50 * time.Millisecond)

	// Stop and close
	session.Close()
}


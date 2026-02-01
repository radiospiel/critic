package git

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/radiospiel/critic/simple-go/assert"
)

func TestNewGitWatcher(t *testing.T) {
	// Create a temporary directory with a .git subdirectory
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	err := os.MkdirAll(gitDir, 0755)
	assert.NoError(t, err)

	watcher, err := NewGitWatcher(gitDir, 50)
	assert.NoError(t, err)
	assert.NotNil(t, watcher, "watcher should not be nil")

	defer watcher.Close()
}

func TestGitWatcherDetectsFileChange(t *testing.T) {
	// Create a temporary directory with a .git subdirectory
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	err := os.MkdirAll(gitDir, 0755)
	assert.NoError(t, err)

	watcher, err := NewGitWatcher(gitDir, 50)
	assert.NoError(t, err)
	defer watcher.Close()

	// Create a file in .git directory (simulates a git operation)
	testFile := filepath.Join(gitDir, "HEAD")
	err = os.WriteFile(testFile, []byte("ref: refs/heads/main\n"), 0644)
	assert.NoError(t, err)

	// Wait for change notification with timeout
	select {
	case <-watcher.Changes():
		// Success - we received a change notification
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected change notification but got none")
	}
}

func TestGitWatcherDetectsSubdirectoryChange(t *testing.T) {
	// Create a temporary directory with a .git subdirectory and refs
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	refsDir := filepath.Join(gitDir, "refs", "heads")
	err := os.MkdirAll(refsDir, 0755)
	assert.NoError(t, err)

	watcher, err := NewGitWatcher(gitDir, 50)
	assert.NoError(t, err)
	defer watcher.Close()

	// Create a ref file (simulates a commit)
	refFile := filepath.Join(refsDir, "main")
	err = os.WriteFile(refFile, []byte("abc123\n"), 0644)
	assert.NoError(t, err)

	// Wait for change notification with timeout
	select {
	case <-watcher.Changes():
		// Success - we received a change notification
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected change notification but got none")
	}
}

func TestGitWatcherDebouncing(t *testing.T) {
	// Create a temporary directory with a .git subdirectory
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	err := os.MkdirAll(gitDir, 0755)
	assert.NoError(t, err)

	watcher, err := NewGitWatcher(gitDir, 100) // 100ms debounce
	assert.NoError(t, err)
	defer watcher.Close()

	testFile := filepath.Join(gitDir, "index")

	// Write multiple times rapidly
	for i := 0; i < 5; i++ {
		err = os.WriteFile(testFile, []byte{byte(i)}, 0644)
		assert.NoError(t, err)
		time.Sleep(10 * time.Millisecond)
	}

	// Should receive exactly one debounced notification
	select {
	case <-watcher.Changes():
		// Got first notification
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected at least one change notification")
	}

	// Should NOT receive another notification immediately
	select {
	case <-watcher.Changes():
		t.Fatal("expected only one debounced notification")
	case <-time.After(200 * time.Millisecond):
		// Success - no additional notification
	}
}

func TestGitWatcherLastChangeTime(t *testing.T) {
	// Create a temporary directory with a .git subdirectory
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	err := os.MkdirAll(gitDir, 0755)
	assert.NoError(t, err)

	watcher, err := NewGitWatcher(gitDir, 50)
	assert.NoError(t, err)
	defer watcher.Close()

	// Initial last change time should be the creation time
	initialTime := watcher.LastChangeTime()
	assert.True(t, initialTime > 0, "initial time should be positive")

	// Create a file to trigger a change
	testFile := filepath.Join(gitDir, "HEAD")
	err = os.WriteFile(testFile, []byte("ref: refs/heads/main\n"), 0644)
	assert.NoError(t, err)

	// Wait for change
	select {
	case <-watcher.Changes():
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected change notification")
	}

	// Last change time should be updated
	newTime := watcher.LastChangeTime()
	assert.True(t, newTime >= initialTime, "time should be updated after change")
}

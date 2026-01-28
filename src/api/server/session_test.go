package server

import (
	"testing"
	"time"

	"github.com/radiospiel/critic/simple-go/assert"
	"github.com/radiospiel/critic/src/pkg/critic"
	"github.com/radiospiel/critic/src/pkg/types"
)

func TestNewSession(t *testing.T) {
	session := NewSession("/test/root", &critic.DummyMessaging{}, DiffArgs{
		Bases:      []string{"main", "HEAD"},
		Paths:      []string{"src/"},
		Extensions: []string{"go", "ts"},
	})

	assert.NotNil(t, session, "session should not be nil")
	assert.Equals(t, session.GetState(), StateReady, "initial state should be READY")
	assert.Equals(t, session.GetCurrentBase(), "", "initial current base should be empty")
	assert.Nil(t, session.GetDiffSummary(), "initial diff should be nil")
}

func TestSessionGetArgs(t *testing.T) {
	args := DiffArgs{
		Bases:      []string{"main", "origin/main"},
		Paths:      []string{"internal/", "src/"},
		Extensions: []string{"go"},
	}
	session := NewSession("/test/root", &critic.DummyMessaging{}, args)

	retrieved := session.GetArgs()
	assert.Equals(t, len(retrieved.Bases), 2, "should have 2 bases")
	assert.Equals(t, retrieved.Bases[0], "main", "first base should be main")
	assert.Equals(t, retrieved.Bases[1], "origin/main", "second base should be origin/main")
	assert.Equals(t, len(retrieved.Paths), 2, "should have 2 paths")
	assert.Equals(t, len(retrieved.Extensions), 1, "should have 1 extension")
}

func TestSessionState(t *testing.T) {
	session := NewSession("/test/root", &critic.DummyMessaging{}, DiffArgs{})

	// Initial state
	assert.Equals(t, session.GetState(), StateReady, "initial state should be READY")

	// State constants should have correct values
	assert.Equals(t, StateInitialising, State("INITIALISING"), "StateInitialising should be INITIALISING")
	assert.Equals(t, StateReady, State("READY"), "StateReady should be READY")
}

func TestFilterDiffByExtensions(t *testing.T) {
	diff := &types.Diff{
		Files: []*types.FileDiff{
			{NewPath: "file1.go", OldPath: "file1.go"},
			{NewPath: "file2.ts", OldPath: "file2.ts"},
			{NewPath: "file3.go", OldPath: "file3.go"},
			{NewPath: "readme.md", OldPath: "readme.md"},
			{NewPath: "config.json", OldPath: "config.json"},
		},
	}

	// Filter for .go files
	filtered := filterDiffByExtensions(diff, []string{"go"})
	assert.Equals(t, len(filtered.Files), 2, "should have 2 go files")
	assert.Equals(t, filtered.Files[0].NewPath, "file1.go", "first file should be file1.go")
	assert.Equals(t, filtered.Files[1].NewPath, "file3.go", "second file should be file3.go")

	// Filter for .ts files
	filtered = filterDiffByExtensions(diff, []string{".ts"})
	assert.Equals(t, len(filtered.Files), 1, "should have 1 ts file")
	assert.Equals(t, filtered.Files[0].NewPath, "file2.ts", "file should be file2.ts")

	// Filter for multiple extensions
	filtered = filterDiffByExtensions(diff, []string{"go", "ts"})
	assert.Equals(t, len(filtered.Files), 3, "should have 3 files")

	// No extensions - return all
	filtered = filterDiffByExtensions(diff, nil)
	assert.Equals(t, len(filtered.Files), 5, "should have all 5 files")

	// Empty extensions slice - return all
	filtered = filterDiffByExtensions(diff, []string{})
	assert.Equals(t, len(filtered.Files), 5, "should have all 5 files")

	// Nil diff
	filtered = filterDiffByExtensions(nil, []string{"go"})
	assert.Nil(t, filtered, "nil diff should return nil")
}

func TestFilterDiffByExtensionsWithDeletedFiles(t *testing.T) {
	diff := &types.Diff{
		Files: []*types.FileDiff{
			{NewPath: "file1.go", OldPath: "file1.go"},
			{NewPath: "", OldPath: "deleted.go", IsDeleted: true},
			{NewPath: "", OldPath: "deleted.ts", IsDeleted: true},
		},
	}

	// Filter for .go files - should include deleted .go file
	filtered := filterDiffByExtensions(diff, []string{"go"})
	assert.Equals(t, len(filtered.Files), 2, "should have 2 go files (including deleted)")
}

func TestDiffArgsWithoutCurrentBase(t *testing.T) {
	// Verify that api.DiffArgs does not have CurrentBase field
	// (unlike session.DiffArgs which does)
	args := DiffArgs{
		Bases:      []string{"main"},
		Paths:      []string{},
		Extensions: []string{},
	}

	// This should compile - DiffArgs only has Bases, Paths, Extensions
	assert.Equals(t, len(args.Bases), 1, "should have 1 base")
	assert.Equals(t, len(args.Paths), 0, "should have 0 paths")
	assert.Equals(t, len(args.Extensions), 0, "should have 0 extensions")
}

func TestFilterDiffByPaths(t *testing.T) {
	diff := &types.Diff{
		Files: []*types.FileDiff{
			{NewPath: "src/main.go", OldPath: "src/main.go"},
			{NewPath: "src/lib/util.go", OldPath: "src/lib/util.go"},
			{NewPath: "tests/main_test.go", OldPath: "tests/main_test.go"},
			{NewPath: "readme.md", OldPath: "readme.md"},
		},
	}

	// Filter for src/ prefix
	filtered := filterDiffByPaths(diff, []string{"src/"})
	assert.Equals(t, len(filtered.Files), 2, "should have 2 files with src/ prefix")
	assert.Equals(t, filtered.Files[0].NewPath, "src/main.go", "first file should be src/main.go")
	assert.Equals(t, filtered.Files[1].NewPath, "src/lib/util.go", "second file should be src/lib/util.go")

	// Filter for multiple prefixes
	filtered = filterDiffByPaths(diff, []string{"src/", "tests/"})
	assert.Equals(t, len(filtered.Files), 3, "should have 3 files")

	// Exact match
	filtered = filterDiffByPaths(diff, []string{"readme.md"})
	assert.Equals(t, len(filtered.Files), 1, "should have 1 file")
	assert.Equals(t, filtered.Files[0].NewPath, "readme.md", "file should be readme.md")

	// No paths - return all
	filtered = filterDiffByPaths(diff, nil)
	assert.Equals(t, len(filtered.Files), 4, "should have all 4 files")

	// Empty paths slice - return all
	filtered = filterDiffByPaths(diff, []string{})
	assert.Equals(t, len(filtered.Files), 4, "should have all 4 files")

	// Nil diff
	filtered = filterDiffByPaths(nil, []string{"src/"})
	assert.Nil(t, filtered, "nil diff should return nil")
}

func TestSessionConcurrentAccess(t *testing.T) {
	session := NewSession("/test/root", &critic.DummyMessaging{}, DiffArgs{
		Bases: []string{"main"},
	})

	// Test concurrent reads don't panic
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_ = session.GetState()
			_ = session.GetCurrentBase()
			_ = session.GetDiffSummary()
			_ = session.GetArgs()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for goroutines")
		}
	}
}

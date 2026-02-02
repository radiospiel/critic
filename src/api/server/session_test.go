package server

import (
	"testing"

	"github.com/radiospiel/critic/simple-go/assert"
	"github.com/radiospiel/critic/src/pkg/critic"
	"github.com/radiospiel/critic/src/pkg/types"
)

func TestNewSession(t *testing.T) {
	session := NewSession("/test/root", &critic.DummyMessaging{}, []string{"src/"}, nil)

	assert.NotNil(t, session, "session should not be nil")
	assert.Equals(t, session.GetState(), StateReady, "initial state should be READY")
	assert.Equals(t, session.GetCurrentBase(), "", "initial current base should be empty")
	assert.Nil(t, session.GetDiffSummary(), "initial diff should be nil")
}

func TestSessionState(t *testing.T) {
	session := NewSession("/test/root", &critic.DummyMessaging{}, []string{}, nil)

	// Initial state
	assert.Equals(t, session.GetState(), StateReady, "initial state should be READY")

	// State constants should have correct values
	assert.Equals(t, StateInitialising, State("INITIALISING"), "StateInitialising should be INITIALISING")
	assert.Equals(t, StateReady, State("READY"), "StateReady should be READY")
}

func TestFilterDiffByExtensions(t *testing.T) {
	files := []*types.FileDiff{
		{NewPath: "file1.go", OldPath: "file1.go"},
		{NewPath: "file2.ts", OldPath: "file2.ts"},
		{NewPath: "file3.go", OldPath: "file3.go"},
		{NewPath: "readme.md", OldPath: "readme.md"},
		{NewPath: "config.json", OldPath: "config.json"},
	}

	// Filter for .go files
	filtered := filterDiffByExtensions(files, []string{"go"})
	assert.Equals(t, len(filtered), 2, "should have 2 go files")
	assert.Equals(t, filtered[0].NewPath, "file1.go", "first file should be file1.go")
	assert.Equals(t, filtered[1].NewPath, "file3.go", "second file should be file3.go")

	// Filter for .ts files
	filtered = filterDiffByExtensions(files, []string{".ts"})
	assert.Equals(t, len(filtered), 1, "should have 1 ts file")
	assert.Equals(t, filtered[0].NewPath, "file2.ts", "file should be file2.ts")

	// Filter for multiple extensions
	filtered = filterDiffByExtensions(files, []string{"go", "ts"})
	assert.Equals(t, len(filtered), 3, "should have 3 files")

	// No extensions - return all
	filtered = filterDiffByExtensions(files, nil)
	assert.Equals(t, len(filtered), 5, "should have all 5 files")

	// Empty extensions slice - return all
	filtered = filterDiffByExtensions(files, []string{})
	assert.Equals(t, len(filtered), 5, "should have all 5 files")

	// Nil diff
	filtered = filterDiffByExtensions(nil, []string{"go"})
	assert.Nil(t, filtered, "nil diff should return nil")
}

func TestFilterDiffByExtensionsWithDeletedFiles(t *testing.T) {
	files := []*types.FileDiff{
		{NewPath: "file1.go", OldPath: "file1.go"},
		{NewPath: "", OldPath: "deleted.go", IsDeleted: true},
		{NewPath: "", OldPath: "deleted.ts", IsDeleted: true},
	}

	// Filter for .go files - should include deleted .go file
	filtered := filterDiffByExtensions(files, []string{"go"})
	assert.Equals(t, len(filtered), 2, "should have 2 go files (including deleted)")
}

func TestFilterDiffByPaths(t *testing.T) {
	files := []*types.FileDiff{
		{NewPath: "src/main.go", OldPath: "src/main.go"},
		{NewPath: "src/lib/util.go", OldPath: "src/lib/util.go"},
		{NewPath: "tests/main_test.go", OldPath: "tests/main_test.go"},
		{NewPath: "readme.md", OldPath: "readme.md"},
	}

	// Filter for src/ prefix
	filtered := filterDiffByPaths(files, []string{"src/"})
	assert.Equals(t, len(filtered), 2, "should have 2 files with src/ prefix")
	assert.Equals(t, filtered[0].NewPath, "src/main.go", "first file should be src/main.go")
	assert.Equals(t, filtered[1].NewPath, "src/lib/util.go", "second file should be src/lib/util.go")

	// Filter for multiple prefixes
	filtered = filterDiffByPaths(files, []string{"src/", "tests/"})
	assert.Equals(t, len(filtered), 3, "should have 3 files")

	// Exact match
	filtered = filterDiffByPaths(files, []string{"readme.md"})
	assert.Equals(t, len(filtered), 1, "should have 1 file")
	assert.Equals(t, filtered[0].NewPath, "readme.md", "file should be readme.md")

	// No paths - return all
	filtered = filterDiffByPaths(files, nil)
	assert.Equals(t, len(filtered), 4, "should have all 4 files")

	// Empty paths slice - return all
	filtered = filterDiffByPaths(files, []string{})
	assert.Equals(t, len(filtered), 4, "should have all 4 files")

	// Nil diff
	filtered = filterDiffByPaths(nil, []string{"src/"})
	assert.Nil(t, filtered, "nil diff should return nil")
}

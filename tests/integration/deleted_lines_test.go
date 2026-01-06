package critic_integration

import (
	"testing"

	"github.com/samber/lo"
	"git.15b.it/eno/critic/internal/assert"
	"git.15b.it/eno/critic/internal/git"
	"git.15b.it/eno/critic/internal/must"
	ctypes "git.15b.it/eno/critic/pkg/types"
)

// TestDeletedLinesContent verifies that deleted lines show the correct content
// in "Last Commit" mode. This test reproduces the bug where deleted lines were
// showing content from the new version instead of the old version.
func TestDeletedLinesContent(t *testing.T) {
	SetupGitRepo(t)

	// Create initial file with a comment and function
	initialContent := `package test

import (
	"testing"
)

// compareDiff compares actual and expected diffs using JSON serialization

func TestParseDiff_Empty(t *testing.T) {
	actual, err := ParseDiff("")
	if err != nil {
		t.Fatalf("ParseDiff() error = %v", err)
	}
}
`
	must.WriteFile("parser_test.go", initialContent)
	CommitFile(t, "parser_test.go")

	// Remove the comment lines (simulating the bug scenario)
	modifiedContent := `package test

import (
	"testing"
)

func TestParseDiff_Empty(t *testing.T) {
	actual, err := ParseDiff("")
	if err != nil {
		t.Fatalf("ParseDiff() error = %v", err)
	}
}
`
	must.WriteFile("parser_test.go", modifiedContent)
	CommitFile(t, "parser_test.go")

	// Get diff for last commit
	diff, err := git.GetDiff([]string{}, git.DiffToLastCommit)
	assert.NoError(t, err)

	// Find parser_test.go in the diff
	file, found := lo.Find(diff.Files, func(f *ctypes.FileDiff) bool {
		return f.NewPath == "parser_test.go"
	})
	assert.True(t, found, "Expected parser_test.go in diff")
	assert.NotNil(t, file, "Expected parser_test.go in diff")

	// Should have hunks with deleted lines
	assert.True(t, len(file.Hunks) > 0, "Expected at least one hunk")

	// Find all deleted lines using lo
	allLines := lo.FlatMap(file.Hunks, func(hunk *ctypes.Hunk, _ int) []*ctypes.Line {
		return hunk.Lines
	})
	deletedLines := lo.Filter(allLines, func(line *ctypes.Line, _ int) bool {
		return line.Type == ctypes.LineDeleted
	})

	// Should have 2 deleted lines: the comment and the blank line
	assert.Equals(t, len(deletedLines), 2, "Expected 2 deleted lines")

	// First deleted line should be the comment
	assert.Contains(t, deletedLines[0].Content, "compareDiff compares actual and expected",
		"First deleted line should contain the comment text")

	// Second deleted line should be blank
	assert.Equals(t, deletedLines[1].Content, "",
		"Second deleted line should be blank")

	// Verify line numbers are correct
	// The comment should be at old line 7 (after package, import, and blank lines)
	assert.Equals(t, deletedLines[0].OldNum, 7,
		"Comment should be at old line 7")

	// The blank line after comment should be at old line 8
	assert.Equals(t, deletedLines[1].OldNum, 8,
		"Blank line should be at old line 8")

	// Deleted lines should have NewNum = 0
	assert.Equals(t, deletedLines[0].NewNum, 0,
		"Deleted lines should have NewNum = 0")
	assert.Equals(t, deletedLines[1].NewNum, 0,
		"Deleted lines should have NewNum = 0")

	// Find the context line after the deletions (the function declaration)
	funcDeclLine, found := lo.Find(allLines, func(line *ctypes.Line) bool {
		return line.Type == ctypes.LineContext && contains(line.Content, "func TestParseDiff_Empty")
	})
	assert.True(t, found, "Expected to find function declaration as context line")

	// Function declaration should be at old line 9, new line 7
	// (moved up by 2 lines after deleting the comment and blank line)
	assert.Equals(t, funcDeclLine.OldNum, 9,
		"Function declaration should be at old line 9")
	assert.Equals(t, funcDeclLine.NewNum, 7,
		"Function declaration should be at new line 7")
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

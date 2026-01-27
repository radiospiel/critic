package critic_integration

import (
	"fmt"
	"os/exec"
	"testing"

	"github.com/samber/lo"
	"git.15b.it/eno/critic/src/git"
	"git.15b.it/eno/critic/simple-go/assert"
	"git.15b.it/eno/critic/simple-go/must"
	ctypes "git.15b.it/eno/critic/src/pkg/types"
)

// commitFileForDiff creates and commits a file

// modifyFileForDiff modifies an existing file without committing

func TestGetDiff_UnstagedMode(t *testing.T) {
	SetupGitRepo(t)

	// Create and commit initial file
	must.WriteFile("test.go", "package main\n\nfunc main() {}\n")
	CommitFile(t, "test.go")

	// Modify the file (unstaged)
	must.WriteFile("test.go", "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n")

	// Get unstaged diff
	diff, err := git.GetDiff([]string{}, git.DiffUnstaged)
	assert.NoError(t, err)
	assert.NotNil(t, diff)
	assert.Equals(t, len(diff.Files), 1, "expected 1 file in diff")

	file := diff.Files[0]
	assert.Equals(t, file.NewPath, "test.go")
	assert.True(t, len(file.Hunks) > 0, "Expected at least one hunk")

	// Verify we have added lines using lo
	allLines := lo.FlatMap(file.Hunks, func(hunk *ctypes.Hunk, _ int) []*ctypes.Line {
		return hunk.Lines
	})
	hasAddedLines := lo.SomeBy(allLines, func(line *ctypes.Line) bool {
		return line.Type == ctypes.LineAdded
	})

	assert.True(t, hasAddedLines, "Expected to find added lines in diff")
}

func TestGetDiff_LastCommitMode(t *testing.T) {
	SetupGitRepo(t)

	// Create initial commit
	must.WriteFile("initial.go", "package main\n")
	CommitFile(t, "initial.go")

	// Create second commit with changes
	must.WriteFile("test.go", "package main\n\nfunc test() {}\n")
	CommitFile(t, "test.go")

	// Get last commit diff
	diff, err := git.GetDiff([]string{}, git.DiffToLastCommit)
	assert.NoError(t, err)
	assert.NotNil(t, diff)
	assert.True(t, len(diff.Files) > 0, "Expected files in last commit diff")

	// Find test.go in the diff
	file, found := lo.Find(diff.Files, func(f *ctypes.FileDiff) bool {
		return f.NewPath == "test.go"
	})

	assert.True(t, found, "Expected test.go in last commit diff")
	assert.True(t, file.IsNew, "Expected test.go to be marked as new file")
}

func TestGetDiff_MergeBaseMode(t *testing.T) {
	SetupGitRepo(t)

	// Create initial commit on main branch
	must.WriteFile("initial.go", "package main\n")
	CommitFile(t, "initial.go")

	// Ensure we're on "main" branch
	exec.Command("git", "branch", "-M", "main").Run() // Ignore error - might already be "main"

	// Create a feature branch
	must.Exec("git", "checkout", "-b", "feature")

	// Make changes on feature branch
	must.WriteFile("feature.go", "package feature\n\nfunc New() {}\n")
	CommitFile(t, "feature.go")
	must.WriteFile("initial.go", "package main\n\n// Modified\n")

	// Get diff from merge base
	diff, err := git.GetDiff([]string{}, git.DiffToMergeBase)
	assert.NoError(t, err)
	assert.NotNil(t, diff)
	assert.True(t, len(diff.Files) >= 1, "Expected at least 1 file in merge base diff")

	// Should include the new file
	foundNew := lo.ContainsBy(diff.Files, func(f *ctypes.FileDiff) bool {
		return f.NewPath == "feature.go"
	})
	foundModified := lo.ContainsBy(diff.Files, func(f *ctypes.FileDiff) bool {
		return f.NewPath == "initial.go"
	})

	assert.True(t, foundNew, "Expected feature.go in merge base diff")
	assert.True(t, foundModified, "Expected initial.go (modified) in merge base diff")
}

func TestGetDiff_EmptyDiff(t *testing.T) {
	SetupGitRepo(t)

	// Create and commit a file
	must.WriteFile("test.go", "package main\n")
	CommitFile(t, "test.go")

	// Get unstaged diff (should be empty)
	diff, err := git.GetDiff([]string{}, git.DiffUnstaged)
	assert.NoError(t, err)
	assert.Equals(t, len(diff.Files), 0, "Expected no files in diff")
}

func TestGetDiff_PathFiltering(t *testing.T) {
	SetupGitRepo(t)

	// Create and commit multiple files
	must.WriteFile("file1.go", "package main\n")
	CommitFile(t, "file1.go")
	must.WriteFile("file2.go", "package test\n")
	CommitFile(t, "file2.go")
	must.WriteFile("file3.go", "package other\n")
	CommitFile(t, "file3.go")

	// Modify all files
	must.WriteFile("file1.go", "package main\n\n// Modified\n")
	must.WriteFile("file2.go", "package test\n\n// Modified\n")
	must.WriteFile("file3.go", "package other\n\n// Modified\n")

	// Get diff for only file1.go
	diff, err := git.GetDiff([]string{"file1.go"}, git.DiffUnstaged)
	assert.NoError(t, err)
	assert.Equals(t, len(diff.Files), 1, "Expected 1 file in diff")
	assert.Equals(t, diff.Files[0].NewPath, "file1.go")
}

func TestGetDiff_MultipleFiles(t *testing.T) {
	SetupGitRepo(t)

	// Create and commit initial files
	must.WriteFile("file1.go", "package main\n")
	CommitFile(t, "file1.go")
	must.WriteFile("file2.go", "package test\n")
	CommitFile(t, "file2.go")

	// Modify both files
	must.WriteFile("file1.go", "package main\n\nimport \"fmt\"\n")
	must.WriteFile("file2.go", "package test\n\nimport \"testing\"\n")

	// Get unstaged diff
	diff, err := git.GetDiff([]string{}, git.DiffUnstaged)
	assert.NoError(t, err)
	assert.Equals(t, len(diff.Files), 2, "Expected 2 files in diff")

	// Verify both files are in diff
	hasFile1 := lo.ContainsBy(diff.Files, func(f *ctypes.FileDiff) bool {
		return f.NewPath == "file1.go"
	})
	hasFile2 := lo.ContainsBy(diff.Files, func(f *ctypes.FileDiff) bool {
		return f.NewPath == "file2.go"
	})

	assert.True(t, hasFile1, "Expected file1.go in diff")
	assert.True(t, hasFile2, "Expected file2.go in diff")
}

func TestGetDiff_NewFile(t *testing.T) {
	SetupGitRepo(t)

	// Create initial commit
	must.WriteFile("existing.go", "package main\n")
	CommitFile(t, "existing.go")

	// Add a new file (staged)
	must.WriteFile("new.go", "package new\n\nfunc New() {}\n")
	must.Exec("git", "add", "new.go")

	// Get diff in merge base mode (includes staged changes)
	// Need to create a branch first
	exec.Command("git", "branch", "-M", "main").Run()

	diff, err := git.GetDiff([]string{}, git.DiffToMergeBase)
	assert.NoError(t, err)

	// Should show the new file
	file, found := lo.Find(diff.Files, func(f *ctypes.FileDiff) bool {
		return f.NewPath == "new.go"
	})

	assert.True(t, found, "Expected new.go in diff")
	assert.True(t, file.IsNew, "Expected new.go to be marked as new file")
}

func TestGetDiff_DeletedFile(t *testing.T) {
	SetupGitRepo(t)

	// Create and commit files
	must.WriteFile("keep.go", "package keep\n")
	CommitFile(t, "keep.go")
	must.WriteFile("delete.go", "package delete\n")
	CommitFile(t, "delete.go")

	// Delete one file
	must.Remove("delete.go")

	// Get unstaged diff
	diff, err := git.GetDiff([]string{}, git.DiffUnstaged)
	assert.NoError(t, err)

	// Should show the deleted file
	file, found := lo.Find(diff.Files, func(f *ctypes.FileDiff) bool {
		return f.OldPath == "delete.go"
	})

	assert.True(t, found, "Expected delete.go in diff")
	assert.True(t, file.IsDeleted, "Expected delete.go to be marked as deleted")
}

func TestGetDiff_FileInSubdirectory(t *testing.T) {
	SetupGitRepo(t)

	// Create subdirectory structure
	must.MkdirAll("src/pkg", 0755)

	// Create and commit file in subdirectory
	filePath := "src/pkg/module.go"
	must.WriteFile(filePath, "package pkg\n")
	must.Exec("git", "add", "src/pkg/module.go")
	must.Exec("git", "commit", "-m", "add module")

	// Modify the file
	must.WriteFile(filePath, "package pkg\n\n// Modified\n")

	// Get diff
	diff, err := git.GetDiff([]string{}, git.DiffUnstaged)
	assert.NoError(t, err)
	assert.Equals(t, len(diff.Files), 1, "Expected 1 file in diff")
	assert.Equals(t, diff.Files[0].NewPath, "src/pkg/module.go")
}

func TestGetDiff_MultiplePathsFilter(t *testing.T) {
	SetupGitRepo(t)

	// Create and commit multiple files
	must.WriteFile("file1.go", "package main\n")
	CommitFile(t, "file1.go")
	must.WriteFile("file2.go", "package test\n")
	CommitFile(t, "file2.go")
	must.WriteFile("file3.go", "package other\n")
	CommitFile(t, "file3.go")

	// Modify all files
	must.WriteFile("file1.go", "package main\n\n// Modified\n")
	must.WriteFile("file2.go", "package test\n\n// Modified\n")
	must.WriteFile("file3.go", "package other\n\n// Modified\n")

	// Get diff for file1.go and file2.go only
	diff, err := git.GetDiff([]string{"file1.go", "file2.go"}, git.DiffUnstaged)
	assert.NoError(t, err)
	assert.Equals(t, len(diff.Files), 2, "Expected 2 files in diff")

	// Verify file3.go is not included
	hasFile3 := lo.ContainsBy(diff.Files, func(f *ctypes.FileDiff) bool {
		return f.NewPath == "file3.go"
	})
	assert.False(t, hasFile3, "file3.go should not be in filtered diff")
}

func TestGetDiff_LargeFile(t *testing.T) {
	SetupGitRepo(t)

	// Create a file with many lines
	largeContent := "package main\n\n"
	for i := 0; i < 100; i++ {
		largeContent += "// Line number " + fmt.Sprintf("%d", i) + "\n"
	}
	must.WriteFile("large.go", largeContent)
	CommitFile(t, "large.go")

	// Modify it by changing a line in the middle
	modifiedContent := "package main\n\n"
	for i := 0; i < 100; i++ {
		if i == 50 {
			modifiedContent += "// MODIFIED LINE\n"
		} else {
			modifiedContent += "// Line number " + fmt.Sprintf("%d", i) + "\n"
		}
	}
	must.WriteFile("large.go", modifiedContent)

	// Get diff
	diff, err := git.GetDiff([]string{}, git.DiffUnstaged)
	assert.NoError(t, err)
	assert.Equals(t, len(diff.Files), 1, "Expected 1 file in diff")
	assert.True(t, len(diff.Files[0].Hunks) > 0, "Expected hunks in large file diff")
}

func TestDiffMode_String(t *testing.T) {
	tests := []struct {
		mode     git.DiffMode
		expected string
	}{
		{git.DiffToMergeBase, "Merge Base"},
		{git.DiffToLastCommit, "Last Commit"},
		{git.DiffUnstaged, "Unstaged"},
		{git.DiffMode(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equals(t, tt.mode.String(), tt.expected)
		})
	}
}

func TestGetDiff_WithContextLines(t *testing.T) {
	SetupGitRepo(t)

	// Create a file with multiple lines
	initial := "line 1\nline 2\nline 3\nline 4\nline 5\nline 6\nline 7\n"
	must.WriteFile("context.txt", initial)
	CommitFile(t, "context.txt")

	// Modify middle line
	modified := "line 1\nline 2\nline 3\nMODIFIED\nline 5\nline 6\nline 7\n"
	must.WriteFile("context.txt", modified)

	// Get diff
	diff, err := git.GetDiff([]string{}, git.DiffUnstaged)
	assert.NoError(t, err)
	assert.Equals(t, len(diff.Files), 1, "Expected 1 file in diff")

	hunk := diff.Files[0].Hunks[0]

	// Should have context lines, deleted line, and added line
	hasContext := lo.SomeBy(hunk.Lines, func(line *ctypes.Line) bool {
		return line.Type == ctypes.LineContext
	})
	hasDeleted := lo.SomeBy(hunk.Lines, func(line *ctypes.Line) bool {
		return line.Type == ctypes.LineDeleted
	})
	hasAdded := lo.SomeBy(hunk.Lines, func(line *ctypes.Line) bool {
		return line.Type == ctypes.LineAdded
	})

	assert.True(t, hasContext, "Expected context lines in diff")
	assert.True(t, hasDeleted, "Expected deleted line in diff")
	assert.True(t, hasAdded, "Expected added line in diff")
}

// TestGetDiff_AmbiguousFilename tests that files with names that look like git refs
// (like "HEAD", "master", etc.) are handled correctly using the "--" separator
func TestGetDiff_AmbiguousFilename(t *testing.T) {
	SetupGitRepo(t)

	// Create and commit a file with an ambiguous name
	must.WriteFile("HEAD", "original content\n")
	CommitFile(t, "HEAD")

	// Modify the file (unstaged)
	must.WriteFile("HEAD", "modified content\n")

	// Get diff for the specific file
	// This should work correctly with "--" separator
	diff, err := git.GetDiff([]string{"HEAD"}, git.DiffUnstaged)
	assert.NoError(t, err)
	assert.True(t, len(diff.Files) > 0, "Expected diff for file 'HEAD', got empty diff")

	// Verify the diff is for the file "HEAD", not a git reference
	foundFile := lo.ContainsBy(diff.Files, func(f *ctypes.FileDiff) bool {
		return f.NewPath == "HEAD" || f.OldPath == "HEAD"
	})

	assert.True(t, foundFile, "Expected diff to contain file 'HEAD'")
}

// TestGitDiff_ErrorWithEmptyOutput tests whether git diff can return
// an error with empty output. This helps verify if the error handling
// at diff.go:75-79 is actually needed.
func TestGitDiff_ErrorWithEmptyOutput(t *testing.T) {
	SetupGitRepo(t)

	// Try various scenarios that might produce error with empty output

	// Scenario 1: Non-existent file path (should return empty diff, not error)
	diff, err := git.GetDiff([]string{"nonexistent.txt"}, git.DiffUnstaged)
	t.Logf("Scenario 1 - Nonexistent file: err=%v, files=%d", err, len(diff.Files))

	// Scenario 2: Empty diff (no changes)
	must.WriteFile("test.txt", "content\n")
	CommitFile(t, "test.txt")
	diff, err = git.GetDiff([]string{}, git.DiffUnstaged)
	t.Logf("Scenario 2 - No changes: err=%v, files=%d", err, len(diff.Files))

	// Conclusion: In normal operation, git diff doesn't return an error with empty output.
	// The error handling at diff.go:75-79 appears to be defensive programming but may not
	// be reachable in practice.
}

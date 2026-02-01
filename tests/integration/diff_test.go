package critic_integration

import (
	"fmt"
	"testing"

	"github.com/samber/lo"
	"github.com/radiospiel/critic/src/git"
	"github.com/radiospiel/critic/simple-go/assert"
	"github.com/radiospiel/critic/simple-go/must"
	ctypes "github.com/radiospiel/critic/src/pkg/types"
)

func TestGetDiff_Basic(t *testing.T) {
	SetupGitRepo(t)

	// Create and commit initial file
	must.WriteFile("test.go", "package main\n\nfunc main() {}\n")
	CommitFile(t, "test.go")

	// Modify the file (unstaged)
	must.WriteFile("test.go", "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n")

	// Get diff from HEAD to working directory
	headSHA := git.ResolveRef("HEAD")
	diff, err := git.GetDiff(headSHA, []string{}, 3)
	assert.NoError(t, err)
	assert.NotNil(t, diff)
	assert.Equals(t, len(diff.Files), 1, "expected 1 file in diff")

	file := diff.Files[0]
	assert.Equals(t, file.NewPath, "test.go")
	assert.True(t, len(file.Hunks) > 0, "Expected at least one hunk")

	// Verify we have added lines
	allLines := lo.FlatMap(file.Hunks, func(hunk *ctypes.Hunk, _ int) []*ctypes.Line {
		return hunk.Lines
	})
	hasAddedLines := lo.SomeBy(allLines, func(line *ctypes.Line) bool {
		return line.Type == ctypes.LineAdded
	})

	assert.True(t, hasAddedLines, "Expected to find added lines in diff")
}

func TestGetDiff_EmptyDiff(t *testing.T) {
	SetupGitRepo(t)

	// Create and commit a file
	must.WriteFile("test.go", "package main\n")
	CommitFile(t, "test.go")

	// Get diff (should be empty since no changes)
	headSHA := git.ResolveRef("HEAD")
	diff, err := git.GetDiff(headSHA, []string{}, 3)
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
	headSHA := git.ResolveRef("HEAD")
	diff, err := git.GetDiff(headSHA, []string{"file1.go"}, 3)
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

	// Get diff
	headSHA := git.ResolveRef("HEAD")
	diff, err := git.GetDiff(headSHA, []string{}, 3)
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

	// Add a new file (not staged yet, just in working directory)
	must.WriteFile("new.go", "package new\n\nfunc New() {}\n")
	must.Exec("git", "add", "new.go")

	// Get diff
	headSHA := git.ResolveRef("HEAD")
	diff, err := git.GetDiff(headSHA, []string{}, 3)
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

	// Get diff
	headSHA := git.ResolveRef("HEAD")
	diff, err := git.GetDiff(headSHA, []string{}, 3)
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
	headSHA := git.ResolveRef("HEAD")
	diff, err := git.GetDiff(headSHA, []string{}, 3)
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
	headSHA := git.ResolveRef("HEAD")
	diff, err := git.GetDiff(headSHA, []string{"file1.go", "file2.go"}, 3)
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
	headSHA := git.ResolveRef("HEAD")
	diff, err := git.GetDiff(headSHA, []string{}, 3)
	assert.NoError(t, err)
	assert.Equals(t, len(diff.Files), 1, "Expected 1 file in diff")
	assert.True(t, len(diff.Files[0].Hunks) > 0, "Expected hunks in large file diff")
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
	headSHA := git.ResolveRef("HEAD")
	diff, err := git.GetDiff(headSHA, []string{}, 3)
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

func TestGetDiff_AmbiguousFilename(t *testing.T) {
	SetupGitRepo(t)

	// Create and commit a file with an ambiguous name
	must.WriteFile("HEAD", "original content\n")
	CommitFile(t, "HEAD")

	// Modify the file
	must.WriteFile("HEAD", "modified content\n")

	// Get diff for the specific file
	headSHA := git.ResolveRef("HEAD")
	diff, err := git.GetDiff(headSHA, []string{"HEAD"}, 3)
	assert.NoError(t, err)
	assert.True(t, len(diff.Files) > 0, "Expected diff for file 'HEAD', got empty diff")

	// Verify the diff is for the file "HEAD", not a git reference
	foundFile := lo.ContainsBy(diff.Files, func(f *ctypes.FileDiff) bool {
		return f.NewPath == "HEAD" || f.OldPath == "HEAD"
	})

	assert.True(t, foundFile, "Expected diff to contain file 'HEAD'")
}

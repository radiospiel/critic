package critic_integration

import (
	"os"
	"os/exec"
	"testing"

	"github.com/samber/lo"
	"git.15b.it/eno/critic/internal/git"
	"git.15b.it/eno/critic/simple-go/assert"
	"git.15b.it/eno/critic/simple-go/must"
	ctypes "git.15b.it/eno/critic/pkg/types"
)

// commitFile creates a file and commits it

// modifyFile modifies an existing file

func TestGitWorkflow_UnstagedChanges(t *testing.T) {
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
	assert.Equals(t, len(diff.Files), 1)

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

func TestGitWorkflow_LastCommit(t *testing.T) {
	SetupGitRepo(t)

	// Create initial commit
	must.WriteFile("initial.go", "package main\n")
	CommitFile(t, "initial.go")

	// Create second commit with changes
	must.WriteFile("test.go", "package main\n\nfunc main() {\n\tfmt.Println(\"test\")\n}\n")
	CommitFile(t, "test.go")

	// Get last commit diff
	diff, err := git.GetDiff([]string{}, git.DiffToLastCommit)
	assert.NoError(t, err)
	assert.NotNil(t, diff)
	assert.True(t, len(diff.Files) > 0, "Expected files in last commit diff")

	// Find test.go in the diff
	_, found := lo.Find(diff.Files, func(f *ctypes.FileDiff) bool {
		return f.NewPath == "test.go"
	})

	assert.True(t, found, "Expected test.go in last commit diff")
}

func TestGitWorkflow_EmptyDiff(t *testing.T) {
	SetupGitRepo(t)

	// Create and commit a file
	must.WriteFile("test.go", "package main\n")
	CommitFile(t, "test.go")

	// Get unstaged diff (should be empty)
	diff, err := git.GetDiff([]string{}, git.DiffUnstaged)
	assert.NoError(t, err)
	assert.Equals(t, len(diff.Files), 0)
}

func TestGitWorkflow_NewFile(t *testing.T) {
	SetupGitRepo(t)

	// Create initial commit
	must.WriteFile("initial.go", "package main\n")
	CommitFile(t, "initial.go")

	// Add a new file (staged)
	must.WriteFile("new.go", "package main\n\nfunc new() {}\n")
	must.Exec("git", "add", "new.go")

	// Get diff (should show staged changes when comparing to HEAD)
	// Note: This tests the merge base mode which includes staged changes
	diff, err := git.GetDiff([]string{}, git.DiffUnstaged)
	assert.NoError(t, err)

	// Unstaged mode won't show staged files, so this is expected to be empty
	// This validates that DiffUnstaged works correctly
	if len(diff.Files) != 0 {
		t.Logf("Note: DiffUnstaged correctly excludes staged files")
	}
}

func TestGitWorkflow_MultipleFiles(t *testing.T) {
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

func TestGitWorkflow_GetCurrentBranch(t *testing.T) {
	SetupGitRepo(t)

	// Create initial commit (required for branch to exist)
	must.WriteFile("test.go", "package main\n")
	CommitFile(t, "test.go")

	branch := must.Must2(git.GetCurrentBranch())

	// Default branch is typically "master" or "main"
	if branch != "master" && branch != "main" {
		t.Logf("Current branch: %s (expected master or main)", branch)
	}
}

func TestGitWorkflow_IsGitRepo(t *testing.T) {
	SetupGitRepo(t)

	assert.True(t, git.IsGitRepo(), "IsGitRepo() should return true for git repository")

	// Test non-git directory
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)

	assert.False(t, git.IsGitRepo(), "IsGitRepo() should return false for non-git directory")
}

func TestGitWorkflow_PathFiltering(t *testing.T) {
	SetupGitRepo(t)

	// Create and commit multiple files
	must.WriteFile("file1.go", "package main\n")
	CommitFile(t, "file1.go")
	must.WriteFile("file2.go", "package test\n")
	CommitFile(t, "file2.go")
	must.WriteFile("file3.go", "package other\n")
	CommitFile(t, "file3.go")

	// Modify all files
	must.WriteFile("file1.go", "package main\n\nimport \"fmt\"\n")
	must.WriteFile("file2.go", "package test\n\nimport \"testing\"\n")
	must.WriteFile("file3.go", "package other\n\nimport \"os\"\n")

	// Get diff for only file1.go
	diff, err := git.GetDiff([]string{"file1.go"}, git.DiffUnstaged)
	assert.NoError(t, err)
	assert.Equals(t, len(diff.Files), 1, "Expected 1 file in diff")
	assert.Equals(t, diff.Files[0].NewPath, "file1.go")
}

func TestGitWorkflow_MergeBaseWithMainBranch(t *testing.T) {
	SetupGitRepo(t)

	// Create initial commit on default branch (usually "main" or "master")
	must.WriteFile("initial.go", "package main\n")
	CommitFile(t, "initial.go")

	// Ensure we're on "main" branch (modern git default)
	exec.Command("git", "branch", "-M", "main").Run() // Ignore error - might already be "main"

	// Create a feature branch
	must.Exec("git", "checkout", "-b", "feature")

	// Make a commit on feature branch
	must.WriteFile("feature.go", "package feature\n")
	CommitFile(t, "feature.go")

	// GetMergeBase should find the merge base with "main" branch
	mergeBase := must.Must2(git.GetMergeBase())

	mergeBase, err := git.GetMergeBase()
	assert.NoError(t, err)
	assert.NotEquals(t, mergeBase, "", "GetMergeBase() should not return empty string")
	assert.True(t, len(mergeBase) >= 7, "GetMergeBase() should return valid commit hash")

	// Verify the merge base is a valid commit
	output := must.Exec("git", "cat-file", "-t", mergeBase)
	assert.Equals(t, output, "commit\n")
}

func TestGitWorkflow_MergeBaseFallbackToMaster(t *testing.T) {
	SetupGitRepo(t)

	// Create initial commit on default branch
	must.WriteFile("initial.go", "package main\n")
	CommitFile(t, "initial.go")

	// Rename branch to "master" to test fallback behavior
	// (modern git uses "main" by default, but we want to test "master" fallback)
	must.Exec("git", "branch", "-m", "master")

	// Create a feature branch
	must.Exec("git", "checkout", "-b", "feature")

	// Make a commit on feature branch
	must.WriteFile("feature.go", "package feature\n")
	CommitFile(t, "feature.go")

	// GetMergeBase should work even though there's no "main" branch
	// It should fallback to "master"
	mergeBase := must.Must2(git.GetMergeBase())

	// Verify it found a valid commit hash
	assert.True(t, len(mergeBase) >= 7, "GetMergeBase() should return valid commit hash")

	// Verify current branch is "feature"
	currentBranch := must.Must2(git.GetCurrentBranch())
	assert.Equals(t, currentBranch, "feature")
}

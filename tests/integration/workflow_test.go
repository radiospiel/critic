package critic_integration

import (
	"os"
	"os/exec"
	"testing"

	"github.com/samber/lo"
	"github.com/radiospiel/critic/src/git"
	"github.com/radiospiel/critic/simple-go/assert"
	"github.com/radiospiel/critic/simple-go/must"
	ctypes "github.com/radiospiel/critic/src/pkg/types"
)

func TestGitWorkflow_UnstagedChanges(t *testing.T) {
	SetupGitRepo(t)

	// Create and commit initial file
	must.WriteFile("test.go", "package main\n\nfunc main() {}\n")
	CommitFile(t, "test.go")

	// Modify the file (unstaged)
	must.WriteFile("test.go", "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n")

	// Get diff from HEAD to working directory
	headSHA := git.ResolveRef("HEAD")
	file, err := git.GetDiff(headSHA, "test.go", 3)
	assert.NoError(t, err)
	assert.NotNil(t, file)

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

func TestGitWorkflow_EmptyDiff(t *testing.T) {
	SetupGitRepo(t)

	// Create and commit a file
	must.WriteFile("test.go", "package main\n")
	CommitFile(t, "test.go")

	// Get diff (should be empty since no changes)
	headSHA := git.ResolveRef("HEAD")
	file, err := git.GetDiff(headSHA, "test.go", 3)
	assert.NoError(t, err)
	assert.Nil(t, file, "Expected no diff for unchanged file")
}

func TestGitWorkflow_NewFile(t *testing.T) {
	SetupGitRepo(t)

	// Create initial commit
	must.WriteFile("initial.go", "package main\n")
	CommitFile(t, "initial.go")

	// Add a new file (staged)
	must.WriteFile("new.go", "package main\n\nfunc new() {}\n")
	must.Exec("git", "add", "new.go")

	// Get diff from HEAD to working directory (includes staged changes)
	headSHA := git.ResolveRef("HEAD")
	file, err := git.GetDiff(headSHA, "new.go", 3)
	assert.NoError(t, err)
	assert.NotNil(t, file, "Expected new.go in diff")
	assert.Equals(t, file.FileStatus, ctypes.FileStatusNew, "Expected new.go to be marked as new file")
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

	// Get diff names
	headSHA := git.ResolveRef("HEAD")
	files, err := git.GetDiffNames(headSHA, []string{})
	assert.NoError(t, err)
	assert.Equals(t, len(files), 2, "Expected 2 files in diff")

	// Verify both files are in diff
	hasFile1 := lo.ContainsBy(files, func(f *ctypes.FileDiff) bool {
		return f.NewPath == "file1.go"
	})
	hasFile2 := lo.ContainsBy(files, func(f *ctypes.FileDiff) bool {
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

	branch := git.GetCurrentBranch()

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
	headSHA := git.ResolveRef("HEAD")
	file, err := git.GetDiff(headSHA, "file1.go", 3)
	assert.NoError(t, err)
	assert.NotNil(t, file, "Expected diff for file1.go")
	assert.Equals(t, file.NewPath, "file1.go")
}

func TestGitWorkflow_FeatureBranch(t *testing.T) {
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

	// Verify current branch is "feature"
	currentBranch := git.GetCurrentBranch()
	assert.Equals(t, currentBranch, "feature")

	// Make uncommitted changes
	must.WriteFile("feature.go", "package feature\n\n// modified\n")

	// Get diff from HEAD to working directory
	headSHA := git.ResolveRef("HEAD")
	file, err := git.GetDiff(headSHA, "feature.go", 3)
	assert.NoError(t, err)
	assert.NotNil(t, file, "Expected diff for feature.go")
}

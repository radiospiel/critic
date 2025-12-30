package critic_integration

import (
	"fmt"
	"os/exec"
	"testing"

	"git.15b.it/eno/critic/internal/git"
	"git.15b.it/eno/critic/internal/must"
	ctypes "git.15b.it/eno/critic/pkg/types"
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
	if err != nil {
		t.Fatalf("git.GetDiff() error = %v", err)
	}

	if diff == nil {
		t.Fatal("git.GetDiff() returned nil")
	}

	if len(diff.Files) != 1 {
		t.Fatalf("Expected 1 file in diff, got %d", len(diff.Files))
	}

	file := diff.Files[0]
	if file.NewPath != "test.go" {
		t.Errorf("File path = %q, want test.go", file.NewPath)
	}

	if len(file.Hunks) == 0 {
		t.Fatal("Expected at least one hunk")
	}

	// Verify we have added lines
	hasAddedLines := false
	for _, hunk := range file.Hunks {
		for _, line := range hunk.Lines {
			if line.Type == ctypes.LineAdded {
				hasAddedLines = true
				break
			}
		}
	}

	if !hasAddedLines {
		t.Error("Expected to find added lines in diff")
	}
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
	if err != nil {
		t.Fatalf("git.GetDiff() error = %v", err)
	}

	if diff == nil {
		t.Fatal("git.GetDiff() returned nil")
	}

	// Should show the file added in last commit
	if len(diff.Files) == 0 {
		t.Fatal("Expected files in last commit diff")
	}

	// Find test.go in the diff
	found := false
	for _, file := range diff.Files {
		if file.NewPath == "test.go" {
			found = true
			if !file.IsNew {
				t.Error("Expected test.go to be marked as new file")
			}
			break
		}
	}

	if !found {
		t.Error("Expected test.go in last commit diff")
	}
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
	if err != nil {
		t.Fatalf("git.GetDiff() error = %v", err)
	}

	if diff == nil {
		t.Fatal("git.GetDiff() returned nil")
	}

	// Should show both committed and uncommitted changes
	if len(diff.Files) < 1 {
		t.Fatal("Expected at least 1 file in merge base diff")
	}

	// Should include the new file
	foundNew := false
	foundModified := false
	for _, file := range diff.Files {
		if file.NewPath == "feature.go" {
			foundNew = true
		}
		if file.NewPath == "initial.go" {
			foundModified = true
		}
	}

	if !foundNew {
		t.Error("Expected feature.go in merge base diff")
	}
	if !foundModified {
		t.Error("Expected initial.go (modified) in merge base diff")
	}
}

func TestGetDiff_EmptyDiff(t *testing.T) {
	SetupGitRepo(t)

	// Create and commit a file
	must.WriteFile("test.go", "package main\n")
	CommitFile(t, "test.go")


	// Get unstaged diff (should be empty)
	diff, err := git.GetDiff([]string{}, git.DiffUnstaged)
	if err != nil {
		t.Fatalf("git.GetDiff() error = %v", err)
	}

	if len(diff.Files) != 0 {
		t.Errorf("Expected no files in diff, got %d", len(diff.Files))
	}
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
	if err != nil {
		t.Fatalf("git.GetDiff() error = %v", err)
	}

	if len(diff.Files) != 1 {
		t.Fatalf("Expected 1 file in diff, got %d", len(diff.Files))
	}

	if diff.Files[0].NewPath != "file1.go" {
		t.Errorf("Expected file1.go, got %s", diff.Files[0].NewPath)
	}
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
	if err != nil {
		t.Fatalf("git.GetDiff() error = %v", err)
	}

	if len(diff.Files) != 2 {
		t.Errorf("Expected 2 files in diff, got %d", len(diff.Files))
	}

	// Verify both files are in diff
	fileNames := make(map[string]bool)
	for _, file := range diff.Files {
		fileNames[file.NewPath] = true
	}

	if !fileNames["file1.go"] {
		t.Error("Expected file1.go in diff")
	}
	if !fileNames["file2.go"] {
		t.Error("Expected file2.go in diff")
	}
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
	if err != nil {
		t.Fatalf("git.GetDiff() error = %v", err)
	}

	// Should show the new file
	found := false
	for _, file := range diff.Files {
		if file.NewPath == "new.go" {
			found = true
			if !file.IsNew {
				t.Error("Expected new.go to be marked as new file")
			}
			break
		}
	}

	if !found {
		t.Error("Expected new.go in diff")
	}
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
	if err != nil {
		t.Fatalf("git.GetDiff() error = %v", err)
	}

	// Should show the deleted file
	found := false
	for _, file := range diff.Files {
		if file.OldPath == "delete.go" {
			found = true
			if !file.IsDeleted {
				t.Error("Expected delete.go to be marked as deleted")
			}
			break
		}
	}

	if !found {
		t.Error("Expected delete.go in diff")
	}
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
	if err != nil {
		t.Fatalf("git.GetDiff() error = %v", err)
	}

	if len(diff.Files) != 1 {
		t.Fatalf("Expected 1 file in diff, got %d", len(diff.Files))
	}

	if diff.Files[0].NewPath != "src/pkg/module.go" {
		t.Errorf("Expected src/pkg/module.go, got %s", diff.Files[0].NewPath)
	}
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
	if err != nil {
		t.Fatalf("git.GetDiff() error = %v", err)
	}

	if len(diff.Files) != 2 {
		t.Fatalf("Expected 2 files in diff, got %d", len(diff.Files))
	}

	// Verify file3.go is not included
	for _, file := range diff.Files {
		if file.NewPath == "file3.go" {
			t.Error("file3.go should not be in filtered diff")
		}
	}
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
	if err != nil {
		t.Fatalf("git.GetDiff() error = %v", err)
	}

	if len(diff.Files) != 1 {
		t.Fatalf("Expected 1 file in diff, got %d", len(diff.Files))
	}

	// Should have hunks
	if len(diff.Files[0].Hunks) == 0 {
		t.Error("Expected hunks in large file diff")
	}
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
			actual := tt.mode.String()
			if actual != tt.expected {
				t.Errorf("DiffMode.String() = %q, want %q", actual, tt.expected)
			}
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
	if err != nil {
		t.Fatalf("git.GetDiff() error = %v", err)
	}

	if len(diff.Files) != 1 {
		t.Fatalf("Expected 1 file in diff, got %d", len(diff.Files))
	}

	hunk := diff.Files[0].Hunks[0]

	// Should have context lines, deleted line, and added line
	hasContext := false
	hasDeleted := false
	hasAdded := false

	for _, line := range hunk.Lines {
		switch line.Type {
		case ctypes.LineContext:
			hasContext = true
		case ctypes.LineDeleted:
			hasDeleted = true
		case ctypes.LineAdded:
			hasAdded = true
		}
	}

	if !hasContext {
		t.Error("Expected context lines in diff")
	}
	if !hasDeleted {
		t.Error("Expected deleted line in diff")
	}
	if !hasAdded {
		t.Error("Expected added line in diff")
	}
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
	if err != nil {
		t.Fatalf("GetDiff failed: %v", err)
	}

	// Verify we got a diff for the file
	if len(diff.Files) == 0 {
		t.Fatal("Expected diff for file 'HEAD', got empty diff")
	}

	// Verify the diff is for the file "HEAD", not a git reference
	foundFile := false
	for _, file := range diff.Files {
		if file.NewPath == "HEAD" || file.OldPath == "HEAD" {
			foundFile = true
			break
		}
	}

	if !foundFile {
		t.Error("Expected diff to contain file 'HEAD'")
	}
}

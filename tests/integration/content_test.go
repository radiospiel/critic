package critic_integration

import (
	"testing"

	"git.15b.it/eno/critic/internal/git"
	"git.15b.it/eno/critic/internal/must"
	tu "git.15b.it/eno/critic/internal/testutils"
)

func TestGetFileContent_FromGit(t *testing.T) {
	SetupGitRepo(t)

	// Create and commit a file
	testContent := "package main\n\nfunc main() {}\n"
	must.WriteFile("test.go", testContent)
	CommitFile(t, "test.go")

	// Get content from HEAD
	content, err := git.GetFileContent("test.go", "HEAD")
	tu.AssertNoError(t, err)
	tu.AssertEquals(t, content, testContent)
}

func TestGetFileContent_FromWorkingDirectory(t *testing.T) {
	SetupGitRepo(t)

	// Create and commit initial version
	initialContent := "version 1\n"
	must.WriteFile("file.txt", initialContent)
	CommitFile(t, "file.txt")

	// Modify file in working directory (don't commit)
	modifiedContent := "version 2 - modified\n"
	must.WriteFile("file.txt", modifiedContent)

	// Get content from working directory (empty revision)
	content, err := git.GetFileContent("file.txt", "")
	tu.AssertNoError(t, err)
	tu.AssertEquals(t, content, modifiedContent)

	// Verify we can still get original from git
	contentFromGit, err := git.GetFileContent("file.txt", "HEAD")
	tu.AssertNoError(t, err)
	tu.AssertEquals(t, contentFromGit, initialContent)
}

func TestGetFileContent_DifferentRevisions(t *testing.T) {
	SetupGitRepo(t)

	// Create first commit
	content1 := "version 1\n"
	must.WriteFile("history.txt", content1)
	CommitFile(t, "history.txt")

	// Create second commit
	content2 := "version 2\n"
	must.WriteFile("history.txt", content2)
	CommitFile(t, "history.txt")

	// Create third commit
	content3 := "version 3\n"
	must.WriteFile("history.txt", content3)
	CommitFile(t, "history.txt")

	// Test HEAD (should be version 3)
	content, err := git.GetFileContent("history.txt", "HEAD")
	if err != nil {
		t.Fatalf("git.GetFileContent(HEAD) error = %v", err)
	}
	if content != content3 {
		t.Errorf("git.GetFileContent(HEAD) = %q, want %q", content, content3)
	}

	// Test HEAD~1 (should be version 2)
	content, err = git.GetFileContent("history.txt", "HEAD~1")
	if err != nil {
		t.Fatalf("git.GetFileContent(HEAD~1) error = %v", err)
	}
	if content != content2 {
		t.Errorf("git.GetFileContent(HEAD~1) = %q, want %q", content, content2)
	}

	// Test HEAD~2 (should be version 1)
	content, err = git.GetFileContent("history.txt", "HEAD~2")
	if err != nil {
		t.Fatalf("git.GetFileContent(HEAD~2) error = %v", err)
	}
	if content != content1 {
		t.Errorf("git.GetFileContent(HEAD~2) = %q, want %q", content, content1)
	}
}

func TestGetFileContent_SpecificCommitHash(t *testing.T) {
	SetupGitRepo(t)

	// Create and commit a file
	testContent := "content at specific commit\n"
	must.WriteFile("file.txt", testContent)
	CommitFile(t, "file.txt")

	// Get the commit hash
	output := must.Run("git", "rev-parse", "HEAD")
	commitHash := string(output[:7]) // Use short hash

	// Modify the file
	must.WriteFile("file.txt", "modified content\n")
	CommitFile(t, "file.txt")

	// Get content from specific commit
	content, err := git.GetFileContent("file.txt", commitHash)
	if err != nil {
		t.Fatalf("git.GetFileContent(%s) error = %v", commitHash, err)
	}

	if content != testContent {
		t.Errorf("git.GetFileContent(%s) = %q, want %q", commitHash, content, testContent)
	}
}

func TestGetFileContent_NonexistentFileInGit(t *testing.T) {
	SetupGitRepo(t)

	// Create a commit (so HEAD exists)
	must.WriteFile("exists.txt", "content\n")
	CommitFile(t, "exists.txt")

	// Try to get non-existent file from git
	_, err := git.GetFileContent("does-not-exist.txt", "HEAD")
	if err == nil {
		t.Error("git.GetFileContent() should return error for non-existent file in git")
	}
}

func TestGetFileContent_NonexistentFileOnDisk(t *testing.T) {
	SetupGitRepo(t)

	// Try to get non-existent file from disk
	_, err := git.GetFileContent("does-not-exist.txt", "")
	if err == nil {
		t.Error("git.GetFileContent() should return error for non-existent file on disk")
	}
}

func TestGetFileContent_InvalidRevision(t *testing.T) {
	SetupGitRepo(t)

	// Create a commit
	must.WriteFile("file.txt", "content\n")
	CommitFile(t, "file.txt")

	// Try to get file from invalid revision
	_, err := git.GetFileContent("file.txt", "invalid-revision-xyz")
	if err == nil {
		t.Error("git.GetFileContent() should return error for invalid revision")
	}
}

func TestGetFileContent_FileInSubdirectory(t *testing.T) {
	SetupGitRepo(t)

	// Create subdirectory structure
	must.MkdirAll("src/pkg", 0755)

	// Create file in subdirectory
	testContent := "package pkg\n"
	filePath := "src/pkg/module.go"
	must.WriteFile(filePath, testContent)

	// Commit the file
	must.Exec("git", "add", "src/pkg/module.go")
	must.Exec("git", "commit", "-m", "add module")

	// Get content from git
	content, err := git.GetFileContent("src/pkg/module.go", "HEAD")
	if err != nil {
		t.Fatalf("git.GetFileContent() error = %v", err)
	}

	if content != testContent {
		t.Errorf("git.GetFileContent() = %q, want %q", content, testContent)
	}
}

func TestGetFileContent_BinaryFile(t *testing.T) {
	SetupGitRepo(t)

	// Create a binary file (with null bytes)
	binaryContent := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE}
	must.WriteFile("binary.dat", binaryContent)

	// Commit the binary file
	must.Exec("git", "add", "binary.dat")
	must.Exec("git", "commit", "-m", "add binary")

	// Get content from git (should work, even though it's binary)
	content, err := git.GetFileContent("binary.dat", "HEAD")
	if err != nil {
		t.Fatalf("git.GetFileContent() error = %v", err)
	}

	// Verify binary content is preserved
	if string(content) != string(binaryContent) {
		t.Errorf("git.GetFileContent() binary content mismatch")
	}
}

func TestGetFileContent_EmptyFile(t *testing.T) {
	SetupGitRepo(t)

	// Create and commit an empty file
	must.WriteFile("empty.txt", "")
	CommitFile(t, "empty.txt")

	// Get empty file content
	content, err := git.GetFileContent("empty.txt", "HEAD")
	if err != nil {
		t.Fatalf("git.GetFileContent() error = %v", err)
	}

	if content != "" {
		t.Errorf("git.GetFileContent() = %q, want empty string", content)
	}
}

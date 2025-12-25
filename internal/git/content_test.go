package git

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestGetFileContentWithExecutor_FromGit(t *testing.T) {
	mock := &MockExecutor{
		Output: []byte("file content from git\n"),
		Err:    nil,
	}

	content, err := GetFileContentWithExecutor("path/to/file.go", "HEAD", mock)
	if err != nil {
		t.Fatalf("GetFileContentWithExecutor() error = %v", err)
	}

	if content != "file content from git\n" {
		t.Errorf("GetFileContentWithExecutor() = %q, want %q", content, "file content from git\n")
	}

	// Verify correct git command was executed
	if len(mock.Commands) != 1 {
		t.Fatalf("Expected 1 command, got %d", len(mock.Commands))
	}
	cmd := mock.Commands[0]
	if len(cmd) < 3 || cmd[0] != "git" || cmd[1] != "show" || cmd[2] != "HEAD:path/to/file.go" {
		t.Errorf("Expected 'git show HEAD:path/to/file.go', got %v", cmd)
	}
}

func TestGetFileContentWithExecutor_GitError(t *testing.T) {
	mock := &MockExecutor{
		Output: nil,
		Err:    fmt.Errorf("fatal: Path does not exist"),
	}

	_, err := GetFileContentWithExecutor("nonexistent.go", "HEAD", mock)
	if err == nil {
		t.Error("GetFileContentWithExecutor() should return error for missing file")
	}
}

func TestGetFileContentWithExecutor_FromWorkingDirectory(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	testContent := "test content from disk\n"
	err := os.WriteFile(tmpFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Mock executor should not be called when revision is empty
	mock := &MockExecutor{}

	content, err := GetFileContentWithExecutor(tmpFile, "", mock)
	if err != nil {
		t.Fatalf("GetFileContentWithExecutor() error = %v", err)
	}

	if content != testContent {
		t.Errorf("GetFileContentWithExecutor() = %q, want %q", content, testContent)
	}

	// Verify mock was not called (file read from disk)
	if len(mock.Commands) != 0 {
		t.Errorf("Expected 0 git commands when reading from working directory, got %d", len(mock.Commands))
	}
}

func TestGetFileContentWithExecutor_DiskFileNotFound(t *testing.T) {
	mock := &MockExecutor{}

	_, err := GetFileContentWithExecutor("/nonexistent/path/file.txt", "", mock)
	if err == nil {
		t.Error("GetFileContentWithExecutor() should return error for nonexistent file")
	}

	// Verify mock was not called
	if len(mock.Commands) != 0 {
		t.Errorf("Expected 0 git commands, got %d", len(mock.Commands))
	}
}

func TestGetFileContentWithExecutor_DifferentRevisions(t *testing.T) {
	tests := []struct {
		name     string
		revision string
		wantArg  string
	}{
		{
			name:     "HEAD",
			revision: "HEAD",
			wantArg:  "HEAD:file.go",
		},
		{
			name:     "Specific commit",
			revision: "abc123",
			wantArg:  "abc123:file.go",
		},
		{
			name:     "HEAD~1",
			revision: "HEAD~1",
			wantArg:  "HEAD~1:file.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockExecutor{
				Output: []byte("content"),
				Err:    nil,
			}

			_, err := GetFileContentWithExecutor("file.go", tt.revision, mock)
			if err != nil {
				t.Fatalf("GetFileContentWithExecutor() error = %v", err)
			}

			if len(mock.Commands) != 1 {
				t.Fatalf("Expected 1 command, got %d", len(mock.Commands))
			}
			cmd := mock.Commands[0]
			if len(cmd) < 3 || cmd[2] != tt.wantArg {
				t.Errorf("Expected argument %q, got %v", tt.wantArg, cmd)
			}
		})
	}
}

func TestGetFileContent_UsesDefaultExecutor(t *testing.T) {
	// This test verifies that the public API works
	// We can't easily test it uses defaultExecutor without mocking,
	// but we can verify it doesn't crash and handles errors

	// Try to get content for a file that doesn't exist in git
	_, err := GetFileContent("definitely-nonexistent-file-xyz.txt", "HEAD")
	// We expect an error since we're likely not in a git repo or file doesn't exist
	// The important thing is it doesn't panic
	_ = err
}

package comments

import (
	"os"
	"path/filepath"
	"testing"

	"git.15b.it/eno/critic/pkg/types"
)

func TestFileManager_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	fm := NewFileManager(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.go")
	content := []string{"line 1", "line 2", "line 3"}
	if err := os.WriteFile(testFile, []byte("line 1\nline 2\nline 3\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a critic file with comments
	criticFile := &types.CriticFile{
		FilePath:      testFile,
		OriginalLines: content,
		Comments: map[int]*types.CriticBlock{
			1: {
				LineNumber: 1,
				Lines:      []string{"This is a comment", "on multiple lines"},
			},
		},
	}

	// Save comments
	if err := fm.SaveComments(criticFile); err != nil {
		t.Fatalf("Failed to save comments: %v", err)
	}

	// Verify .critic.md file was created
	criticPath := fm.GetCriticFilePath(testFile)
	if _, err := os.Stat(criticPath); os.IsNotExist(err) {
		t.Error("Expected .critic.md file to be created")
	}

	// Verify .critic.original file was created
	originalPath := fm.GetCriticOriginalPath(testFile)
	if _, err := os.Stat(originalPath); os.IsNotExist(err) {
		t.Error("Expected .critic.original file to be created")
	}

	// Load comments back
	loadedFile, err := fm.LoadComments(testFile)
	if err != nil {
		t.Fatalf("Failed to load comments: %v", err)
	}

	// Verify loaded content matches
	if len(loadedFile.OriginalLines) != len(content) {
		t.Errorf("Expected %d original lines, got %d", len(content), len(loadedFile.OriginalLines))
	}

	if len(loadedFile.Comments) != 1 {
		t.Errorf("Expected 1 comment, got %d", len(loadedFile.Comments))
	}

	// Check comment content
	if comment, exists := loadedFile.Comments[1]; exists {
		if len(comment.Lines) != 2 {
			t.Errorf("Expected 2 comment lines, got %d", len(comment.Lines))
		}
		if comment.Lines[0] != "This is a comment" {
			t.Errorf("Expected first line to be 'This is a comment', got %q", comment.Lines[0])
		}
	} else {
		t.Error("Expected comment at line 1")
	}
}

func TestFileManager_HasComments(t *testing.T) {
	tmpDir := t.TempDir()
	fm := NewFileManager(tmpDir)

	testFile := filepath.Join(tmpDir, "test.go")

	// Should return false for non-existent file
	if fm.HasComments(testFile) {
		t.Error("Expected HasComments to return false for non-existent file")
	}

	// Create a critic file
	criticPath := fm.GetCriticFilePath(testFile)
	if err := os.WriteFile(criticPath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create critic file: %v", err)
	}

	// Should return true now
	if !fm.HasComments(testFile) {
		t.Error("Expected HasComments to return true after creating critic file")
	}
}

func TestFileManager_DeleteComments(t *testing.T) {
	tmpDir := t.TempDir()
	fm := NewFileManager(tmpDir)

	testFile := filepath.Join(tmpDir, "test.go")

	// Create critic files
	criticPath := fm.GetCriticFilePath(testFile)
	originalPath := fm.GetCriticOriginalPath(testFile)

	if err := os.WriteFile(criticPath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create critic file: %v", err)
	}
	if err := os.WriteFile(originalPath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create original file: %v", err)
	}

	// Delete comments
	if err := fm.DeleteComments(testFile); err != nil {
		t.Fatalf("Failed to delete comments: %v", err)
	}

	// Verify files are deleted
	if _, err := os.Stat(criticPath); !os.IsNotExist(err) {
		t.Error("Expected .critic.md file to be deleted")
	}
	if _, err := os.Stat(originalPath); !os.IsNotExist(err) {
		t.Error("Expected .critic.original file to be deleted")
	}
}

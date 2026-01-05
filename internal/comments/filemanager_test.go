package comments

import (
	"os"
	"path/filepath"
	"testing"

	"git.15b.it/eno/critic/pkg/types"
)

func TestFileManager_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	fm := NewFileManager(tmpDir, "current")

	// Create a test file in the tmpDir
	testFile := "test.go"
	testFilePath := filepath.Join(tmpDir, testFile)
	content := []string{"line 1", "line 2", "line 3"}
	if err := os.WriteFile(testFilePath, []byte("line 1\nline 2\nline 3\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a critic file with comments (using relative path)
	criticFile := &types.CriticFile{
		FilePath:      testFile,
		OriginalLines: content,
		Comments: map[int]*types.CriticBlock{
			1: {
				LineNumber: 1,
				Lines:      []string{"This is a comment", "on multiple lines"},
				UUID:       "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			},
		},
	}

	// Save comments
	if err := fm.SaveComments(criticFile); err != nil {
		t.Fatalf("Failed to save comments: %v", err)
	}

	// Verify .comments file was created
	commentsPath := fm.GetCriticCommentsPath(testFile)
	if _, err := os.Stat(commentsPath); os.IsNotExist(err) {
		t.Errorf("Expected .comments file to be created at %s", commentsPath)
	}

	// Verify original copy file was created
	originalPath := fm.GetCriticOriginalPath(testFile)
	if _, err := os.Stat(originalPath); os.IsNotExist(err) {
		t.Errorf("Expected original copy to be created at %s", originalPath)
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
	fm := NewFileManager(tmpDir, "current")

	testFile := "test.go"
	testFilePath := filepath.Join(tmpDir, testFile)

	// Create test file
	if err := os.WriteFile(testFilePath, []byte("line 1\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Should return false for file without comments
	if fm.HasComments(testFile) {
		t.Error("Expected HasComments to return false for file without comments")
	}

	// Create a critic file with comments
	criticFile := &types.CriticFile{
		FilePath:      testFile,
		OriginalLines: []string{"line 1"},
		Comments: map[int]*types.CriticBlock{
			0: {
				LineNumber: 0,
				Lines:      []string{"A comment"},
				UUID:       "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			},
		},
	}

	if err := fm.SaveComments(criticFile); err != nil {
		t.Fatalf("Failed to save comments: %v", err)
	}

	// Should return true now
	if !fm.HasComments(testFile) {
		t.Error("Expected HasComments to return true after saving comments")
	}
}

func TestFileManager_DeleteComments(t *testing.T) {
	tmpDir := t.TempDir()
	fm := NewFileManager(tmpDir, "current")

	testFile := "test.go"
	testFilePath := filepath.Join(tmpDir, testFile)

	// Create test file
	if err := os.WriteFile(testFilePath, []byte("line 1\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create critic files
	criticFile := &types.CriticFile{
		FilePath:      testFile,
		OriginalLines: []string{"line 1"},
		Comments: map[int]*types.CriticBlock{
			0: {
				LineNumber: 0,
				Lines:      []string{"A comment"},
			},
		},
	}

	if err := fm.SaveComments(criticFile); err != nil {
		t.Fatalf("Failed to save comments: %v", err)
	}

	// Verify files exist
	commentsPath := fm.GetCriticCommentsPath(testFile)
	originalPath := fm.GetCriticOriginalPath(testFile)

	if _, err := os.Stat(commentsPath); os.IsNotExist(err) {
		t.Error("Expected comments file to exist before deletion")
	}
	if _, err := os.Stat(originalPath); os.IsNotExist(err) {
		t.Error("Expected original copy to exist before deletion")
	}

	// Delete comments
	if err := fm.DeleteComments(testFile); err != nil {
		t.Fatalf("Failed to delete comments: %v", err)
	}

	// Verify files are deleted
	if _, err := os.Stat(commentsPath); !os.IsNotExist(err) {
		t.Error("Expected comments file to be deleted")
	}
	if _, err := os.Stat(originalPath); !os.IsNotExist(err) {
		t.Error("Expected original copy to be deleted")
	}
}

func TestFileManager_CriticDir(t *testing.T) {
	tmpDir := t.TempDir()
	fm := NewFileManager(tmpDir, "current")

	expectedDir := filepath.Join(tmpDir, ".critic", "current")
	if fm.GetCriticDir() != expectedDir {
		t.Errorf("Expected critic dir %s, got %s", expectedDir, fm.GetCriticDir())
	}
}

func TestFileManager_PathConstruction(t *testing.T) {
	tmpDir := t.TempDir()
	fm := NewFileManager(tmpDir, "main")

	testFile := "src/foo/bar.go"

	expectedComments := filepath.Join(tmpDir, ".critic", "main", "src/foo/bar.go.comments")
	if fm.GetCriticCommentsPath(testFile) != expectedComments {
		t.Errorf("Expected comments path %s, got %s", expectedComments, fm.GetCriticCommentsPath(testFile))
	}

	expectedOriginal := filepath.Join(tmpDir, ".critic", "main", "src/foo/bar.go")
	if fm.GetCriticOriginalPath(testFile) != expectedOriginal {
		t.Errorf("Expected original path %s, got %s", expectedOriginal, fm.GetCriticOriginalPath(testFile))
	}
}

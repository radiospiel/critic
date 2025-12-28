package git

import (
	"os"
	"path/filepath"
	"testing"

	ctypes "git.15b.it/eno/critic/pkg/types"
)

func TestGetUntrackedFiles(t *testing.T) {
	// This test requires a git repository
	// We'll test with the current repo
	files, err := GetUntrackedFiles([]string{"."}, []string{"go"})
	if err != nil {
		t.Fatalf("GetUntrackedFiles() error = %v", err)
	}

	// We can't assert specific files, but we can check the structure
	for _, file := range files {
		if !filepath.IsAbs(file) && filepath.Ext(file) != ".go" {
			t.Errorf("GetUntrackedFiles() returned non-.go file: %s", file)
		}
	}
}

func TestGetUntrackedFiles_NoExtensionFilter(t *testing.T) {
	// Test with no extension filtering
	files, err := GetUntrackedFiles([]string{"."}, nil)
	if err != nil {
		t.Fatalf("GetUntrackedFiles() error = %v", err)
	}

	// Should return all untracked files
	// Can't assert specific count, but should not error
	_ = files
}

func TestCreateUntrackedFileDiff(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	content := "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n"

	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	fileDiff, err := createUntrackedFileDiff(testFile)
	if err != nil {
		t.Fatalf("createUntrackedFileDiff() error = %v", err)
	}

	// Verify basic structure
	if fileDiff.OldPath != "" {
		t.Errorf("createUntrackedFileDiff() OldPath = %q, want empty", fileDiff.OldPath)
	}
	if fileDiff.NewPath != testFile {
		t.Errorf("createUntrackedFileDiff() NewPath = %q, want %q", fileDiff.NewPath, testFile)
	}
	if !fileDiff.IsNew {
		t.Error("createUntrackedFileDiff() IsNew = false, want true")
	}
	if fileDiff.IsDeleted {
		t.Error("createUntrackedFileDiff() IsDeleted = true, want false")
	}

	// Verify hunk structure
	if len(fileDiff.Hunks) != 1 {
		t.Fatalf("createUntrackedFileDiff() got %d hunks, want 1", len(fileDiff.Hunks))
	}

	hunk := fileDiff.Hunks[0]
	if hunk.OldStart != 0 || hunk.OldLines != 0 {
		t.Errorf("createUntrackedFileDiff() hunk old range = %d,%d, want 0,0", hunk.OldStart, hunk.OldLines)
	}
	if hunk.NewStart != 1 {
		t.Errorf("createUntrackedFileDiff() hunk NewStart = %d, want 1", hunk.NewStart)
	}

	// Verify all lines are additions
	for i, line := range hunk.Lines {
		if line.Type != ctypes.LineAdded {
			t.Errorf("createUntrackedFileDiff() line %d type = %v, want LineAdded", i, line.Type)
		}
		if line.OldNum != 0 {
			t.Errorf("createUntrackedFileDiff() line %d OldNum = %d, want 0", i, line.OldNum)
		}
		if line.NewNum != i+1 {
			t.Errorf("createUntrackedFileDiff() line %d NewNum = %d, want %d", i, line.NewNum, i+1)
		}
	}

	// Verify content is preserved
	expectedLines := []string{"package main", "", "func main() {", "\tprintln(\"hello\")", "}", ""}
	if len(hunk.Lines) != len(expectedLines) {
		t.Errorf("createUntrackedFileDiff() got %d lines, want %d", len(hunk.Lines), len(expectedLines))
	} else {
		for i, expected := range expectedLines {
			if hunk.Lines[i].Content != expected {
				t.Errorf("createUntrackedFileDiff() line %d content = %q, want %q", i, hunk.Lines[i].Content, expected)
			}
		}
	}
}

func TestGetUntrackedDiff(t *testing.T) {
	// Test with current directory
	diff, err := GetUntrackedDiff([]string{"."}, []string{"go"})
	if err != nil {
		t.Fatalf("GetUntrackedDiff() error = %v", err)
	}

	// Verify structure
	if diff == nil {
		t.Fatal("GetUntrackedDiff() returned nil diff")
	}
	if diff.Files == nil {
		t.Fatal("GetUntrackedDiff() returned diff with nil Files")
	}

	// Verify all files are marked as new
	for _, file := range diff.Files {
		if !file.IsNew {
			t.Errorf("GetUntrackedDiff() file %s IsNew = false, want true", file.NewPath)
		}
		if file.IsDeleted {
			t.Errorf("GetUntrackedDiff() file %s IsDeleted = true, want false", file.NewPath)
		}
	}
}

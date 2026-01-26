package git

import (
	"os"
	"path/filepath"
	"testing"

	"git.15b.it/eno/critic/simple-go/assert"
	ctypes "git.15b.it/eno/critic/src/pkg/types"
)

func TestGetUntrackedFiles(t *testing.T) {
	// This test requires a git repository
	// We'll test with the current repo
	files, err := GetUntrackedFiles([]string{"."}, []string{"go"})
	assert.NoError(t, err)

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
	assert.NoError(t, err)

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
	assert.NoError(t, err, "Failed to create test file")

	fileDiff, err := createUntrackedFileDiff(testFile)
	assert.NoError(t, err)

	// Verify basic structure
	assert.Equals(t, fileDiff.OldPath, "", "OldPath should be empty")
	assert.Equals(t, fileDiff.NewPath, testFile)
	assert.True(t, fileDiff.IsNew, "IsNew should be true")
	assert.False(t, fileDiff.IsDeleted, "IsDeleted should be false")

	// Verify hunk structure
	assert.Equals(t, len(fileDiff.Hunks), 1, "expected 1 hunk")

	hunk := fileDiff.Hunks[0]
	assert.Equals(t, hunk.OldStart, 0)
	assert.Equals(t, hunk.OldLines, 0)
	assert.Equals(t, hunk.NewStart, 1)

	// Verify all lines are additions
	for i, line := range hunk.Lines {
		assert.Equals(t, line.Type, ctypes.LineAdded, "line %d should be LineAdded", i)
		assert.Equals(t, line.OldNum, 0, "line %d OldNum should be 0", i)
		assert.Equals(t, line.NewNum, i+1, "line %d NewNum should be %d", i, i+1)
	}

	// Verify content is preserved
	expectedLines := []string{"package main", "", "func main() {", "\tprintln(\"hello\")", "}", ""}
	assert.Equals(t, len(hunk.Lines), len(expectedLines), "line count mismatch")
	for i, expected := range expectedLines {
		assert.Equals(t, hunk.Lines[i].Content, expected, "line %d content mismatch", i)
	}
}

func TestGetUntrackedDiff(t *testing.T) {
	// Test with current directory
	diff, err := GetUntrackedDiff([]string{"."}, []string{"go"})
	assert.NoError(t, err)

	// Verify structure
	assert.NotNil(t, diff, "diff should not be nil")
	assert.NotNil(t, diff.Files, "diff.Files should not be nil")

	// Verify all files are marked as new
	for _, file := range diff.Files {
		assert.True(t, file.IsNew, "file %s IsNew should be true", file.NewPath)
		assert.False(t, file.IsDeleted, "file %s IsDeleted should be false", file.NewPath)
	}
}

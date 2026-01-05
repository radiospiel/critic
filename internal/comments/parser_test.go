package comments

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.15b.it/eno/critic/pkg/types"
)

func TestParseCriticFile(t *testing.T) {
	// Create a temporary test file
	content := `line 1
line 2
--- CRITIC 2 LINES a1b2c3d4-e5f6-7890-abcd-ef1234567890 ---
This is a comment
on two lines
--- CRITIC END ---
line 3
line 4
--- CRITIC 1 LINES b2c3d4e5-f6a7-8901-bcde-f12345678901 ---
Single line comment
--- CRITIC END ---
line 5
`

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go.critic.md")
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Parse the file
	criticFile, err := ParseCriticFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse critic file: %v", err)
	}

	// Verify original lines
	expectedLines := []string{"line 1", "line 2", "line 3", "line 4", "line 5"}
	if len(criticFile.OriginalLines) != len(expectedLines) {
		t.Errorf("Expected %d original lines, got %d", len(expectedLines), len(criticFile.OriginalLines))
	}

	for i, expected := range expectedLines {
		if i >= len(criticFile.OriginalLines) || criticFile.OriginalLines[i] != expected {
			t.Errorf("Line %d: expected %q, got %q", i, expected, criticFile.OriginalLines[i])
		}
	}

	// Verify comments
	if len(criticFile.Comments) != 2 {
		t.Errorf("Expected 2 comments, got %d", len(criticFile.Comments))
	}

	// Check first comment at line 2
	if comment, exists := criticFile.Comments[2]; exists {
		if len(comment.Lines) != 2 {
			t.Errorf("Expected 2 comment lines, got %d", len(comment.Lines))
		}
		if comment.Lines[0] != "This is a comment" {
			t.Errorf("Expected first comment line to be 'This is a comment', got %q", comment.Lines[0])
		}
	} else {
		t.Error("Expected comment at line 2")
	}

	// Check second comment at line 4
	if comment, exists := criticFile.Comments[4]; exists {
		if len(comment.Lines) != 1 {
			t.Errorf("Expected 1 comment line, got %d", len(comment.Lines))
		}
		if comment.Lines[0] != "Single line comment" {
			t.Errorf("Expected comment to be 'Single line comment', got %q", comment.Lines[0])
		}
	} else {
		t.Error("Expected comment at line 4")
	}
}

func TestFormatCriticFile(t *testing.T) {
	criticFile := &types.CriticFile{
		FilePath:      "test.go",
		OriginalLines: []string{"line 1", "line 2", "line 3"},
		Comments: map[int]*types.CriticBlock{
			1: {
				LineNumber: 1,
				Lines:      []string{"Comment for line 2"},
				UUID:       "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			},
		},
	}

	formatted := FormatCriticFile(criticFile)

	// Verify the formatted output contains the comment
	if !strings.Contains(formatted, "--- CRITIC 1 LINES a1b2c3d4-e5f6-7890-abcd-ef1234567890 ---") {
		t.Error("Formatted output should contain CRITIC opening fence")
	}
	if !strings.Contains(formatted, "Comment for line 2") {
		t.Error("Formatted output should contain comment text")
	}
	if !strings.Contains(formatted, "--- CRITIC END ---") {
		t.Error("Formatted output should contain CRITIC closing fence")
	}
}

func TestValidateCriticFile(t *testing.T) {
	criticFile := &types.CriticFile{
		FilePath:      "test.go",
		OriginalLines: []string{"line 1", "line 2", "line 3"},
		Comments: map[int]*types.CriticBlock{
			1: {
				LineNumber: 1,
				Lines:      []string{"Comment"},
			},
		},
	}

	// Should validate successfully
	if err := ValidateCriticFile(criticFile, []string{"line 1", "line 2", "line 3"}); err != nil {
		t.Errorf("Expected validation to succeed, got error: %v", err)
	}

	// Should fail with different content
	if err := ValidateCriticFile(criticFile, []string{"different", "content"}); err == nil {
		t.Error("Expected validation to fail with different content")
	}

	// Should fail with invalid line number
	criticFile.Comments[999] = &types.CriticBlock{
		LineNumber: 999,
		Lines:      []string{"Invalid"},
	}
	if err := ValidateCriticFile(criticFile, []string{"line 1", "line 2", "line 3"}); err == nil {
		t.Error("Expected validation to fail with invalid line number")
	}
}

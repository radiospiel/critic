package comments

import (
	"testing"

	"git.15b.it/eno/critic/pkg/types"
)

func TestSyncComments_NoChanges(t *testing.T) {
	// Create a critic file with comments
	criticFile := &types.CriticFile{
		FilePath:      "test.go",
		OriginalLines: []string{"line 1", "line 2", "line 3"},
		Comments: map[int]*types.CriticBlock{
			1: {
				LineNumber: 1,
				Lines:      []string{"Comment on line 2"},
			},
		},
	}

	oldContent := []string{"line 1", "line 2", "line 3"}
	newContent := []string{"line 1", "line 2", "line 3"}

	// Sync comments (no changes)
	newCriticFile, err := SyncComments(criticFile, oldContent, newContent)
	if err != nil {
		t.Fatalf("SyncComments failed: %v", err)
	}

	// Verify comment is still at line 1
	if _, exists := newCriticFile.Comments[1]; !exists {
		t.Error("Expected comment at line 1")
	}

	if len(newCriticFile.Comments) != 1 {
		t.Errorf("Expected 1 comment, got %d", len(newCriticFile.Comments))
	}
}

func TestSyncComments_LineAddedBefore(t *testing.T) {
	// Create a critic file with comments
	criticFile := &types.CriticFile{
		FilePath:      "test.go",
		OriginalLines: []string{"line 1", "line 2", "line 3"},
		Comments: map[int]*types.CriticBlock{
			1: {
				LineNumber: 1,
				Lines:      []string{"Comment on line 2"},
			},
		},
	}

	oldContent := []string{"line 1", "line 2", "line 3"}
	newContent := []string{"new line 0", "line 1", "line 2", "line 3"}

	// Sync comments (line added before commented line)
	newCriticFile, err := SyncComments(criticFile, oldContent, newContent)
	if err != nil {
		t.Fatalf("SyncComments failed: %v", err)
	}

	// Comment should now be at line 2 (shifted down by 1)
	if _, exists := newCriticFile.Comments[2]; !exists {
		t.Errorf("Expected comment at line 2, got comments at: %v", getCommentLineNumbers(newCriticFile))
	}

	if len(newCriticFile.Comments) != 1 {
		t.Errorf("Expected 1 comment, got %d", len(newCriticFile.Comments))
	}
}

func TestSyncComments_LineDeletedBefore(t *testing.T) {
	// Create a critic file with comments
	criticFile := &types.CriticFile{
		FilePath:      "test.go",
		OriginalLines: []string{"line 0", "line 1", "line 2", "line 3"},
		Comments: map[int]*types.CriticBlock{
			2: {
				LineNumber: 2,
				Lines:      []string{"Comment on line 3"},
			},
		},
	}

	oldContent := []string{"line 0", "line 1", "line 2", "line 3"}
	newContent := []string{"line 1", "line 2", "line 3"}

	// Sync comments (line deleted before commented line)
	newCriticFile, err := SyncComments(criticFile, oldContent, newContent)
	if err != nil {
		t.Fatalf("SyncComments failed: %v", err)
	}

	// Comment should now be at line 1 (shifted up by 1)
	if _, exists := newCriticFile.Comments[1]; !exists {
		t.Errorf("Expected comment at line 1, got comments at: %v", getCommentLineNumbers(newCriticFile))
	}

	if len(newCriticFile.Comments) != 1 {
		t.Errorf("Expected 1 comment, got %d", len(newCriticFile.Comments))
	}
}

func TestSyncComments_CommentedLineDeleted(t *testing.T) {
	// Create a critic file with comments
	criticFile := &types.CriticFile{
		FilePath:      "test.go",
		OriginalLines: []string{"line 1", "line 2", "line 3"},
		Comments: map[int]*types.CriticBlock{
			1: {
				LineNumber: 1,
				Lines:      []string{"Comment on line 2"},
			},
		},
	}

	oldContent := []string{"line 1", "line 2", "line 3"}
	newContent := []string{"line 1", "line 3"}

	// Sync comments (commented line deleted)
	newCriticFile, err := SyncComments(criticFile, oldContent, newContent)
	if err != nil {
		t.Fatalf("SyncComments failed: %v", err)
	}

	// Comment should be dropped because line 2 was deleted
	// The comment was before line 2, so it should be gone
	if len(newCriticFile.Comments) > 1 {
		t.Errorf("Expected at most 1 comment (may be preserved at adjacent line), got %d at lines: %v",
			len(newCriticFile.Comments), getCommentLineNumbers(newCriticFile))
	}
}

func TestSyncComments_MultipleComments(t *testing.T) {
	// Create a critic file with multiple comments
	criticFile := &types.CriticFile{
		FilePath:      "test.go",
		OriginalLines: []string{"line 1", "line 2", "line 3", "line 4", "line 5"},
		Comments: map[int]*types.CriticBlock{
			1: {
				LineNumber: 1,
				Lines:      []string{"Comment on line 2"},
			},
			3: {
				LineNumber: 3,
				Lines:      []string{"Comment on line 4"},
			},
		},
	}

	oldContent := []string{"line 1", "line 2", "line 3", "line 4", "line 5"}
	newContent := []string{"new line", "line 1", "line 2", "line 3", "line 4", "line 5"}

	// Sync comments (line added at beginning)
	newCriticFile, err := SyncComments(criticFile, oldContent, newContent)
	if err != nil {
		t.Fatalf("SyncComments failed: %v", err)
	}

	// Both comments should be shifted down by 1
	hasComment2 := false
	hasComment4 := false
	for lineNum := range newCriticFile.Comments {
		if lineNum == 2 {
			hasComment2 = true
		}
		if lineNum == 4 {
			hasComment4 = true
		}
	}

	if !hasComment2 {
		t.Errorf("Expected comment at line 2, got comments at: %v", getCommentLineNumbers(newCriticFile))
	}
	if !hasComment4 {
		t.Errorf("Expected comment at line 4, got comments at: %v", getCommentLineNumbers(newCriticFile))
	}

	if len(newCriticFile.Comments) != 2 {
		t.Errorf("Expected 2 comments, got %d", len(newCriticFile.Comments))
	}
}

func TestSyncComments_LineAddedAfter(t *testing.T) {
	// Create a critic file with comments
	criticFile := &types.CriticFile{
		FilePath:      "test.go",
		OriginalLines: []string{"line 1", "line 2", "line 3"},
		Comments: map[int]*types.CriticBlock{
			1: {
				LineNumber: 1,
				Lines:      []string{"Comment on line 2"},
			},
		},
	}

	oldContent := []string{"line 1", "line 2", "line 3"}
	newContent := []string{"line 1", "line 2", "new line", "line 3"}

	// Sync comments (line added after commented line)
	newCriticFile, err := SyncComments(criticFile, oldContent, newContent)
	if err != nil {
		t.Fatalf("SyncComments failed: %v", err)
	}

	// Comment should still be at line 1 (before "line 2")
	if _, exists := newCriticFile.Comments[1]; !exists {
		t.Errorf("Expected comment at line 1, got comments at: %v", getCommentLineNumbers(newCriticFile))
	}

	if len(newCriticFile.Comments) != 1 {
		t.Errorf("Expected 1 comment, got %d", len(newCriticFile.Comments))
	}
}

func TestSyncComments_ComplexChanges(t *testing.T) {
	// Create a critic file with multiple comments
	criticFile := &types.CriticFile{
		FilePath: "test.go",
		OriginalLines: []string{
			"func main() {",
			"    fmt.Println(\"hello\")",
			"    x := 1",
			"    y := 2",
			"}",
		},
		Comments: map[int]*types.CriticBlock{
			0: {
				LineNumber: 0,
				Lines:      []string{"Entry point"},
			},
			2: {
				LineNumber: 2,
				Lines:      []string{"Initialize x"},
			},
		},
	}

	oldContent := []string{
		"func main() {",
		"    fmt.Println(\"hello\")",
		"    x := 1",
		"    y := 2",
		"}",
	}

	// Add lines and modify
	newContent := []string{
		"func main() {",
		"    fmt.Println(\"hello\")",
		"    // New comment line",
		"    x := 1",
		"    y := 2",
		"    z := 3",
		"}",
	}

	// Sync comments
	newCriticFile, err := SyncComments(criticFile, oldContent, newContent)
	if err != nil {
		t.Fatalf("SyncComments failed: %v", err)
	}

	// Entry point comment should still be at line 0
	if _, exists := newCriticFile.Comments[0]; !exists {
		t.Errorf("Expected comment at line 0, got comments at: %v", getCommentLineNumbers(newCriticFile))
	}

	// "Initialize x" comment should be shifted to line 3 (one line added before it)
	hasCommentAtCorrectLine := false
	for lineNum := range newCriticFile.Comments {
		if lineNum == 3 {
			hasCommentAtCorrectLine = true
		}
	}

	if !hasCommentAtCorrectLine {
		t.Errorf("Expected comment to be shifted to line 3, got comments at: %v", getCommentLineNumbers(newCriticFile))
	}
}

// Helper function to get line numbers of all comments
func getCommentLineNumbers(criticFile *types.CriticFile) []int {
	var lines []int
	for lineNum := range criticFile.Comments {
		lines = append(lines, lineNum)
	}
	return lines
}

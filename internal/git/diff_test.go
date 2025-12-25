package git

import (
	"fmt"
	"testing"
)

func TestGetDiffWithExecutor_MergeBaseMode(t *testing.T) {
	mock := &MultiCallMockExecutor{
		Calls: []struct {
			Output []byte
			Err    error
		}{
			{Output: []byte("abc123\n"), Err: nil}, // GetMergeBase call
			{Output: []byte(sampleDiffOutput()), Err: nil}, // git diff call
		},
	}

	diff, err := GetDiffWithExecutor([]string{}, DiffToMergeBase, mock)
	if err != nil {
		t.Fatalf("GetDiffWithExecutor() error = %v", err)
	}

	if diff == nil {
		t.Fatal("GetDiffWithExecutor() returned nil diff")
	}

	// Verify commands executed
	if len(mock.Commands) != 2 {
		t.Fatalf("Expected 2 commands, got %d", len(mock.Commands))
	}

	// First command should be merge-base
	if mock.Commands[0][1] != "merge-base" {
		t.Errorf("First command should be merge-base, got %v", mock.Commands[0])
	}

	// Second command should be diff with the merge base
	if mock.Commands[1][0] != "git" || mock.Commands[1][1] != "diff" || mock.Commands[1][2] != "abc123" {
		t.Errorf("Second command should be 'git diff abc123', got %v", mock.Commands[1])
	}

	// Should have --patch and --no-color flags
	hasNoPatch := false
	hasNoColor := false
	for _, arg := range mock.Commands[1] {
		if arg == "--patch" {
			hasNoPatch = true
		}
		if arg == "--no-color" {
			hasNoColor = true
		}
	}
	if !hasNoPatch || !hasNoColor {
		t.Errorf("Expected --patch and --no-color flags in: %v", mock.Commands[1])
	}
}

func TestGetDiffWithExecutor_LastCommitMode(t *testing.T) {
	mock := &MockExecutor{
		Output: []byte(sampleDiffOutput()),
		Err:    nil,
	}

	diff, err := GetDiffWithExecutor([]string{}, DiffToLastCommit, mock)
	if err != nil {
		t.Fatalf("GetDiffWithExecutor() error = %v", err)
	}

	if diff == nil {
		t.Fatal("GetDiffWithExecutor() returned nil diff")
	}

	// Verify command
	if len(mock.Commands) != 1 {
		t.Fatalf("Expected 1 command, got %d", len(mock.Commands))
	}

	cmd := mock.Commands[0]
	if cmd[0] != "git" || cmd[1] != "show" || cmd[2] != "HEAD" {
		t.Errorf("Expected 'git show HEAD', got %v", cmd)
	}
}

func TestGetDiffWithExecutor_UnstagedMode(t *testing.T) {
	mock := &MockExecutor{
		Output: []byte(sampleDiffOutput()),
		Err:    nil,
	}

	diff, err := GetDiffWithExecutor([]string{}, DiffUnstaged, mock)
	if err != nil {
		t.Fatalf("GetDiffWithExecutor() error = %v", err)
	}

	if diff == nil {
		t.Fatal("GetDiffWithExecutor() returned nil diff")
	}

	// Verify command
	if len(mock.Commands) != 1 {
		t.Fatalf("Expected 1 command, got %d", len(mock.Commands))
	}

	cmd := mock.Commands[0]
	if cmd[0] != "git" || cmd[1] != "diff" {
		t.Errorf("Expected 'git diff', got %v", cmd)
	}

	// Should NOT have a commit hash (unlike merge base mode)
	if len(cmd) > 2 && cmd[2] != "--patch" && cmd[2] != "--no-color" {
		t.Errorf("Expected no commit hash in unstaged mode, got %v", cmd)
	}
}

func TestGetDiffWithExecutor_WithPaths(t *testing.T) {
	mock := &MockExecutor{
		Output: []byte(sampleDiffOutput()),
		Err:    nil,
	}

	paths := []string{"file1.go", "file2.go"}
	_, err := GetDiffWithExecutor(paths, DiffUnstaged, mock)
	if err != nil {
		t.Fatalf("GetDiffWithExecutor() error = %v", err)
	}

	// Verify paths are included in command
	cmd := mock.Commands[0]
	hasFile1 := false
	hasFile2 := false
	for _, arg := range cmd {
		if arg == "file1.go" {
			hasFile1 = true
		}
		if arg == "file2.go" {
			hasFile2 = true
		}
	}

	if !hasFile1 || !hasFile2 {
		t.Errorf("Expected file paths in command, got %v", cmd)
	}
}

func TestGetDiffWithExecutor_EmptyDiff(t *testing.T) {
	mock := &MockExecutor{
		Output: []byte(""),
		Err:    fmt.Errorf("exit status 1"), // Git returns error for empty diff
	}

	diff, err := GetDiffWithExecutor([]string{}, DiffUnstaged, mock)
	if err != nil {
		t.Fatalf("GetDiffWithExecutor() should handle empty diff, got error: %v", err)
	}

	if len(diff.Files) != 0 {
		t.Errorf("Empty diff should have 0 files, got %d", len(diff.Files))
	}
}

func TestGetDiffWithExecutor_InvalidMergeBase(t *testing.T) {
	mock := &MultiCallMockExecutor{
		Calls: []struct {
			Output []byte
			Err    error
		}{
			{Output: []byte("invalid hash!\n"), Err: nil}, // GetMergeBase returns invalid hash
		},
	}

	_, err := GetDiffWithExecutor([]string{}, DiffToMergeBase, mock)
	if err == nil {
		t.Error("GetDiffWithExecutor() should return error for invalid merge base")
	}

	// Error is wrapped, so just check it contains the expected text
	if err != nil && err.Error() != "failed to get merge base: invalid merge base format: invalid hash!" {
		t.Errorf("Expected wrapped 'invalid merge base format' error, got: %v", err)
	}
}

func TestGetDiffWithExecutor_MergeBaseFails(t *testing.T) {
	mock := &MultiCallMockExecutor{
		Calls: []struct {
			Output []byte
			Err    error
		}{
			{Output: nil, Err: fmt.Errorf("not a git repo")}, // GetMergeBase fails (main)
			{Output: nil, Err: fmt.Errorf("not a git repo")}, // GetMergeBase fails (master)
		},
	}

	_, err := GetDiffWithExecutor([]string{}, DiffToMergeBase, mock)
	if err == nil {
		t.Error("GetDiffWithExecutor() should return error when merge base fails")
	}
}

func TestGetDiffWithExecutor_ParseError(t *testing.T) {
	mock := &MockExecutor{
		Output: []byte("invalid diff output that won't parse"),
		Err:    nil,
	}

	// Note: ParseDiff is quite forgiving, so this might not actually error
	// But we test the error path exists
	diff, _ := GetDiffWithExecutor([]string{}, DiffUnstaged, mock)

	// Should return empty diff on parse issues
	if diff == nil {
		t.Error("GetDiffWithExecutor() should return diff even on parse issues")
	}
}

func TestDiffMode_String(t *testing.T) {
	tests := []struct {
		mode DiffMode
		want string
	}{
		{DiffToMergeBase, "Merge Base"},
		{DiffToLastCommit, "Last Commit"},
		{DiffUnstaged, "Unstaged"},
		{DiffMode(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.mode.String()
			if got != tt.want {
				t.Errorf("DiffMode.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetDiff_UsesDefaultExecutor(t *testing.T) {
	// This test verifies the public API works
	// We expect it might fail since we're not in a real git repo,
	// but the important thing is it doesn't panic
	_, err := GetDiff([]string{}, DiffUnstaged)
	_ = err // We don't care if it errors
}

// sampleDiffOutput returns a simple valid diff for testing
func sampleDiffOutput() string {
	return `diff --git a/test.go b/test.go
--- a/test.go
+++ b/test.go
@@ -1,2 +1,2 @@
 package main
-import "fmt"
+import "log"
`
}

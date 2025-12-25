package git

import (
	"fmt"
	"testing"
)

func TestGetMergeBaseWithExecutor_Main(t *testing.T) {
	mock := &MockExecutor{
		Output: []byte("abc123def456\n"),
		Err:    nil,
	}

	base, err := GetMergeBaseWithExecutor(mock)
	if err != nil {
		t.Fatalf("GetMergeBaseWithExecutor() error = %v", err)
	}

	if base != "abc123def456" {
		t.Errorf("GetMergeBaseWithExecutor() = %q, want %q", base, "abc123def456")
	}

	// Verify it tried "main" branch first
	if len(mock.Commands) != 1 {
		t.Fatalf("Expected 1 command, got %d", len(mock.Commands))
	}
	cmd := mock.Commands[0]
	if len(cmd) < 4 || cmd[0] != "git" || cmd[1] != "merge-base" || cmd[2] != "main" {
		t.Errorf("Expected 'git merge-base main HEAD', got %v", cmd)
	}
}

// MultiCallMockExecutor returns different results for successive calls
type MultiCallMockExecutor struct {
	Commands [][]string
	Calls    []struct {
		Output []byte
		Err    error
	}
	callIndex int
}

func (m *MultiCallMockExecutor) Run(name string, args ...string) ([]byte, error) {
	cmd := append([]string{name}, args...)
	m.Commands = append(m.Commands, cmd)

	if m.callIndex >= len(m.Calls) {
		return nil, fmt.Errorf("unexpected call %d", m.callIndex)
	}

	result := m.Calls[m.callIndex]
	m.callIndex++
	return result.Output, result.Err
}

func TestGetMergeBaseWithExecutor_FallbackToMaster(t *testing.T) {
	mock := &MultiCallMockExecutor{
		Calls: []struct {
			Output []byte
			Err    error
		}{
			{Output: nil, Err: fmt.Errorf("main branch not found")}, // First call (main) fails
			{Output: []byte("def456abc123\n"), Err: nil},            // Second call (master) succeeds
		},
	}

	base, err := GetMergeBaseWithExecutor(mock)
	if err != nil {
		t.Fatalf("GetMergeBaseWithExecutor() error = %v", err)
	}

	if base != "def456abc123" {
		t.Errorf("GetMergeBaseWithExecutor() = %q, want %q", base, "def456abc123")
	}

	// Verify it tried both "main" and "master"
	if len(mock.Commands) != 2 {
		t.Fatalf("Expected 2 commands, got %d", len(mock.Commands))
	}

	// First command should be for "main"
	if mock.Commands[0][2] != "main" {
		t.Errorf("First command should use 'main', got %v", mock.Commands[0])
	}

	// Second command should be for "master"
	if mock.Commands[1][2] != "master" {
		t.Errorf("Second command should use 'master', got %v", mock.Commands[1])
	}
}

func TestGetMergeBaseWithExecutor_BothFail(t *testing.T) {
	mock := &MockExecutor{
		Output: nil,
		Err:    fmt.Errorf("branch not found"),
	}

	_, err := GetMergeBaseWithExecutor(mock)
	if err == nil {
		t.Error("GetMergeBaseWithExecutor() should return error when both branches fail")
	}

	// Should have tried both branches
	if len(mock.Commands) != 2 {
		t.Errorf("Expected 2 commands, got %d", len(mock.Commands))
	}
}

func TestGetMergeBaseWithExecutor_InvalidHash(t *testing.T) {
	mock := &MockExecutor{
		Output: []byte("invalid hash with spaces!\n"),
		Err:    nil,
	}

	_, err := GetMergeBaseWithExecutor(mock)
	if err == nil {
		t.Error("GetMergeBaseWithExecutor() should return error for invalid hash format")
	}

	if err != nil && err.Error() != "invalid merge base format: invalid hash with spaces!" {
		t.Errorf("Expected 'invalid merge base format' error, got: %v", err)
	}
}

func TestGetMergeBaseWithExecutor_EmptyHash(t *testing.T) {
	mock := &MockExecutor{
		Output: []byte("\n"),
		Err:    nil,
	}

	_, err := GetMergeBaseWithExecutor(mock)
	if err == nil {
		t.Error("GetMergeBaseWithExecutor() should return error for empty merge base")
	}
}

func TestGetCurrentBranchWithExecutor(t *testing.T) {
	mock := &MockExecutor{
		Output: []byte("feature/my-branch\n"),
		Err:    nil,
	}

	branch, err := GetCurrentBranchWithExecutor(mock)
	if err != nil {
		t.Fatalf("GetCurrentBranchWithExecutor() error = %v", err)
	}

	if branch != "feature/my-branch" {
		t.Errorf("GetCurrentBranchWithExecutor() = %q, want %q", branch, "feature/my-branch")
	}

	// Verify correct command was executed
	if len(mock.Commands) != 1 {
		t.Fatalf("Expected 1 command, got %d", len(mock.Commands))
	}
	cmd := mock.Commands[0]
	if len(cmd) < 3 || cmd[0] != "git" || cmd[1] != "rev-parse" || cmd[2] != "--abbrev-ref" {
		t.Errorf("Expected 'git rev-parse --abbrev-ref HEAD', got %v", cmd)
	}
}

func TestGetCurrentBranchWithExecutor_Error(t *testing.T) {
	mock := &MockExecutor{
		Output: nil,
		Err:    fmt.Errorf("not a git repository"),
	}

	_, err := GetCurrentBranchWithExecutor(mock)
	if err == nil {
		t.Error("GetCurrentBranchWithExecutor() should return error when command fails")
	}
}

func TestIsGitRepoWithExecutor_True(t *testing.T) {
	mock := &MockExecutor{
		Output: []byte(".git\n"),
		Err:    nil,
	}

	isRepo := IsGitRepoWithExecutor(mock)
	if !isRepo {
		t.Error("IsGitRepoWithExecutor() = false, want true")
	}

	// Verify correct command was executed
	if len(mock.Commands) != 1 {
		t.Fatalf("Expected 1 command, got %d", len(mock.Commands))
	}
	cmd := mock.Commands[0]
	if len(cmd) < 2 || cmd[0] != "git" || cmd[1] != "rev-parse" {
		t.Errorf("Expected 'git rev-parse --git-dir', got %v", cmd)
	}
}

func TestIsGitRepoWithExecutor_False(t *testing.T) {
	mock := &MockExecutor{
		Output: nil,
		Err:    fmt.Errorf("not a git repository"),
	}

	isRepo := IsGitRepoWithExecutor(mock)
	if isRepo {
		t.Error("IsGitRepoWithExecutor() = true, want false")
	}
}

func TestGetMergeBaseWithExecutor_ValidatesSHA(t *testing.T) {
	tests := []struct {
		name      string
		hash      string
		wantError bool
	}{
		{
			name:      "Valid full SHA-1",
			hash:      "abc123def456789012345678901234567890abcd",
			wantError: false,
		},
		{
			name:      "Valid short SHA",
			hash:      "abc123",
			wantError: false,
		},
		{
			name:      "Invalid - too short",
			hash:      "abc12",
			wantError: true,
		},
		{
			name:      "Invalid - contains spaces",
			hash:      "abc123 def456",
			wantError: true,
		},
		{
			name:      "Invalid - contains uppercase",
			hash:      "ABC123",
			wantError: true,
		},
		{
			name:      "Invalid - contains special chars",
			hash:      "abc123;rm -rf /",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockExecutor{
				Output: []byte(tt.hash + "\n"),
				Err:    nil,
			}

			_, err := GetMergeBaseWithExecutor(mock)
			if tt.wantError && err == nil {
				t.Errorf("GetMergeBaseWithExecutor() should reject invalid hash %q", tt.hash)
			}
			if !tt.wantError && err != nil {
				t.Errorf("GetMergeBaseWithExecutor() should accept valid hash %q, got error: %v", tt.hash, err)
			}
		})
	}
}

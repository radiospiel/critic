package git

import (
	"testing"
)

func TestGetDiffBetween(t *testing.T) {
	// Test with HEAD and working directory
	headSHA := ResolveRef("HEAD")

	diff, err := GetDiffBetween(headSHA, "current", []string{"."})
	if err != nil {
		t.Fatalf("GetDiffBetween() error = %v", err)
	}

	if diff == nil {
		t.Fatal("GetDiffBetween() returned nil diff")
	}

	// Should have Files field (even if empty)
	if diff.Files == nil {
		t.Error("GetDiffBetween() diff.Files is nil")
	}
}

func TestGetDiffBetween_InvalidBase(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("GetDiffBetween() should panic on invalid base ref")
		}
	}()
	GetDiffBetween("invalid", "current", []string{"."})
}

func TestGetDiffBetween_InvalidTarget(t *testing.T) {
	headSHA := ResolveRef("HEAD")

	_, err := GetDiffBetween(headSHA, "invalid", []string{"."})
	if err == nil {
		t.Error("GetDiffBetween() should error on invalid target commit")
	}
}

func TestGetDiffBetween_CommitToCommit(t *testing.T) {
	// Get HEAD and HEAD~1 SHAs
	headSHA := ResolveRef("HEAD")

	// Check if HEAD~1 exists
	if !HasRef("HEAD~1") {
		// Might not have HEAD~1 in a new repo, skip test
		t.Skipf("Skipping test: HEAD~1 not available")
	}
	head1SHA := ResolveRef("HEAD~1")

	diff, err := GetDiffBetween(head1SHA, headSHA, []string{"."})
	if err != nil {
		t.Fatalf("GetDiffBetween() error = %v", err)
	}

	if diff == nil {
		t.Fatal("GetDiffBetween() returned nil diff")
	}
}

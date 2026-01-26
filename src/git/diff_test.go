package git

import (
	"testing"
)

func TestGetDiffBetween(t *testing.T) {
	// Test with HEAD and working directory
	headSHA, err := ResolveRef("HEAD")
	if err != nil {
		t.Fatalf("Failed to resolve HEAD: %v", err)
	}

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
	_, err := GetDiffBetween("invalid", "current", []string{"."})
	if err == nil {
		t.Error("GetDiffBetween() should error on invalid base commit")
	}
}

func TestGetDiffBetween_InvalidTarget(t *testing.T) {
	headSHA, err := ResolveRef("HEAD")
	if err != nil {
		t.Fatalf("Failed to resolve HEAD: %v", err)
	}

	_, err = GetDiffBetween(headSHA, "invalid", []string{"."})
	if err == nil {
		t.Error("GetDiffBetween() should error on invalid target commit")
	}
}

func TestGetDiffBetween_CommitToCommit(t *testing.T) {
	// Get HEAD and HEAD~1 SHAs
	headSHA, err := ResolveRef("HEAD")
	if err != nil {
		t.Fatalf("Failed to resolve HEAD: %v", err)
	}

	head1SHA, err := ResolveRef("HEAD~1")
	if err != nil {
		// Might not have HEAD~1 in a new repo, skip test
		t.Skipf("Skipping test: HEAD~1 not available")
	}

	diff, err := GetDiffBetween(head1SHA, headSHA, []string{"."})
	if err != nil {
		t.Fatalf("GetDiffBetween() error = %v", err)
	}

	if diff == nil {
		t.Fatal("GetDiffBetween() returned nil diff")
	}
}

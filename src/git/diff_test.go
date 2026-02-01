package git

import (
	"testing"
)

func TestGetDiff(t *testing.T) {
	// Test with HEAD and working directory
	headSHA := ResolveRef("HEAD")

	diff, err := GetDiff(headSHA, []string{"."}, 0)
	if err != nil {
		t.Fatalf("GetDiff() error = %v", err)
	}

	if diff == nil {
		t.Fatal("GetDiff() returned nil diff")
	}

	// Should have Files field (even if empty)
	if diff.Files == nil {
		t.Error("GetDiff() diff.Files is nil")
	}
}

func TestGetDiff_InvalidBase(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("GetDiff() should panic on invalid base ref")
		}
	}()
	GetDiff("invalid", []string{"."}, 0)
}

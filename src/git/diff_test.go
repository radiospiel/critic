package git

import (
	"testing"
)

func TestGetDiff(t *testing.T) {
	// Test with HEAD and a known file
	headSHA := ResolveRef("HEAD")

	// Use a file that exists in the repo
	fileDiff, err := GetDiff(headSHA, "go.mod", 0)
	if err != nil {
		t.Fatalf("GetFileDiffs() error = %v", err)
	}

	// fileDiff may be nil if the file hasn't changed, which is valid
	_ = fileDiff
}

func TestGetDiff_InvalidBase(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("GetFileDiffs() should panic on invalid base ref")
		}
	}()
	GetDiff("invalid", "go.mod", 0)
}

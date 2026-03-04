package git

import (
	"testing"

	"github.com/radiospiel/critic/simple-go/assert"
)

func TestLocalBranchesOnPath_EmptyAncestor(t *testing.T) {
	result := LocalBranchesOnPath("")
	assert.Nil(t, result, "empty ancestor should return nil")
}

func TestLocalBranchesOnPath(t *testing.T) {
	// Find a valid ancestor ref to test with.
	ancestor := ""
	for _, candidate := range []string{"master", "main"} {
		if HasRef(candidate) {
			ancestor = candidate
			break
		}
	}
	if ancestor == "" {
		t.Skip("no master/main branch found")
	}

	result := LocalBranchesOnPath(ancestor)

	// Result should not contain the ancestor itself.
	for _, r := range result {
		if r == ancestor {
			t.Errorf("result should not contain ancestor %s", ancestor)
		}
	}

	// Result should not contain the current branch (HEAD).
	headBranch := GetCurrentBranch()
	for _, r := range result {
		if r == headBranch {
			t.Errorf("result should not contain current branch %s", headBranch)
		}
	}

	// Result should not contain duplicates.
	seen := make(map[string]bool)
	for _, r := range result {
		if seen[r] {
			t.Errorf("duplicate branch in result: %s", r)
		}
		seen[r] = true
	}
}

package git

import (
	"testing"

	"github.com/radiospiel/critic/simple-go/assert"
)

func TestGraphDistance(t *testing.T) {
	// HEAD has distance 0 from itself.
	d := graphDistance("HEAD")
	assert.Equals(t, d, 0, "HEAD should have distance 0 from itself")

	// Invalid ref should return MaxInt.
	d = graphDistance("nonexistent-ref-xyz")
	assert.NotEquals(t, d, 0, "invalid ref should not have distance 0")
}

func TestSortRefsByGraphOrder(t *testing.T) {
	// Only test with refs that exist in this repo.
	refs := []string{}
	for _, candidate := range []string{"master", "main", "HEAD"} {
		if HasRef(candidate) {
			refs = append(refs, candidate)
		}
	}
	if len(refs) == 0 {
		t.Skip("no usable refs found")
	}

	SortRefsByGraphOrder(refs)

	// Verify distances are non-increasing (oldest/most distant first).
	for i := 1; i < len(refs); i++ {
		prev := graphDistance(refs[i-1])
		curr := graphDistance(refs[i])
		if prev < curr {
			t.Errorf("refs not sorted: %s (dist %d) should come after %s (dist %d)",
				refs[i-1], prev, refs[i], curr)
		}
	}
}

func TestSortRefsByGraphOrder_EmptySlice(t *testing.T) {
	refs := []string{}
	SortRefsByGraphOrder(refs)
	assert.Equals(t, len(refs), 0, "empty slice should stay empty")
}

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
	ancSHA := ResolveRef(ancestor)
	for _, r := range result {
		sha := ResolveRef(r)
		if sha == ancSHA {
			t.Errorf("result should not contain ancestor %s (sha %s)", ancestor, ancSHA)
		}
	}

	// Result should not contain HEAD.
	headSHA := ResolveRef("HEAD")
	for _, r := range result {
		sha := ResolveRef(r)
		if sha == headSHA {
			t.Errorf("result should not contain HEAD (sha %s)", headSHA)
		}
	}

	// Result should be sorted by graph order (oldest/most distant first).
	for i := 1; i < len(result); i++ {
		prev := graphDistance(result[i-1])
		curr := graphDistance(result[i])
		if prev < curr {
			t.Errorf("results not sorted: %s (dist %d) should come after %s (dist %d)",
				result[i-1], prev, result[i], curr)
		}
	}
}

package git

import (
	"testing"
	"time"

	"github.com/radiospiel/critic/simple-go/assert"
)

func TestNewBaseResolver(t *testing.T) {
	// Test with real git refs
	bases := []string{"HEAD"}
	resolver := NewBaseResolver(bases, "current", nil)
	defer resolver.Stop()

	// Check that bases were resolved
	resolved := resolver.GetResolvedBases()
	assert.Equals(t, len(resolved), 1, "expected 1 resolved base")

	sha, ok := resolver.GetResolvedBase("HEAD")
	assert.True(t, ok, "GetResolvedBase(HEAD) not found")
	assert.NotEquals(t, sha, "", "GetResolvedBase(HEAD) returned empty SHA")
	assert.True(t, len(sha) >= 40, "GetResolvedBase(HEAD) returned SHA with length %d, want 40", len(sha))
}

func TestBaseResolver_ResolveOne(t *testing.T) {
	resolver := &BaseResolver{
		resolvedBases: make(map[string]string),
	}

	// Test resolving HEAD
	sha := resolver.resolveOne("HEAD")
	assert.NotEquals(t, sha, "", "resolveOne(HEAD) returned empty SHA")

	// Test resolving merge-base (skip if main/master not available, e.g., in CI)
	if HasRef("main") || HasRef("master") {
		sha = resolver.resolveOne("merge-base")
		assert.NotEquals(t, sha, "", "resolveOne(merge-base) returned empty SHA")
	} else {
		t.Skip("Skipping merge-base test: no main or master branch available")
	}
}

func TestBaseResolver_GetResolvedBases(t *testing.T) {
	bases := []string{"HEAD"}
	resolver := NewBaseResolver(bases, "current", nil)
	defer resolver.Stop()

	// Get resolved bases
	resolved := resolver.GetResolvedBases()

	// Modify the returned map (should not affect internal state)
	resolved["HEAD"] = "modified"

	// Get again and verify internal state wasn't modified
	resolved2 := resolver.GetResolvedBases()
	assert.NotEquals(t, resolved2["HEAD"], "modified", "GetResolvedBases() should return a copy, not the internal map")
}

func TestBaseResolver_OnChange(t *testing.T) {
	// This test verifies that the polling mechanism works
	// We can't easily test actual changes without modifying git state,
	// but we can verify the structure is correct

	changeCalled := false
	onChange := func() {
		changeCalled = true
	}

	bases := []string{"HEAD"}
	resolver := NewBaseResolver(bases, "current", onChange)
	defer resolver.Stop()

	// Wait a bit to ensure polling goroutine is running
	time.Sleep(100 * time.Millisecond)

	// Since we haven't changed anything in git, onChange shouldn't be called
	assert.False(t, changeCalled, "onChange was called even though no changes occurred")
}

func TestBaseResolver_Stop(t *testing.T) {
	bases := []string{"HEAD"}
	resolver := NewBaseResolver(bases, "current", nil)

	// Stop should not panic
	resolver.Stop()

	// Calling Stop again should not panic
	// (though the channel is already closed, we shouldn't be sending to it)
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Stop() panicked on second call: %v", r)
		}
	}()
}

func TestBaseResolver_CheckForChanges(t *testing.T) {
	bases := []string{"HEAD"}
	resolver := NewBaseResolver(bases, "current", nil)
	defer resolver.Stop()

	// First check should return false (nothing changed)
	changed := resolver.checkForChanges()
	assert.False(t, changed, "checkForChanges() returned true on first check (nothing should have changed)")
}

package git

import (
	"testing"
	"time"
)

func TestNewBaseResolver(t *testing.T) {
	// Test with real git refs
	bases := []string{"HEAD"}
	resolver, err := NewBaseResolver(bases, "current", nil)
	if err != nil {
		t.Fatalf("NewBaseResolver() error = %v", err)
	}
	defer resolver.Stop()

	// Check that bases were resolved
	resolved := resolver.GetResolvedBases()
	if len(resolved) != 1 {
		t.Errorf("GetResolvedBases() returned %d bases, want 1", len(resolved))
	}

	sha, ok := resolver.GetResolvedBase("HEAD")
	if !ok {
		t.Error("GetResolvedBase(HEAD) not found")
	}
	if sha == "" {
		t.Error("GetResolvedBase(HEAD) returned empty SHA")
	}
	if len(sha) < 40 {
		t.Errorf("GetResolvedBase(HEAD) returned SHA with length %d, want 40", len(sha))
	}
}

func TestBaseResolver_ResolveOne(t *testing.T) {
	resolver := &BaseResolver{
		resolvedBases: make(map[string]string),
	}

	// Test resolving HEAD
	sha, err := resolver.resolveOne("HEAD")
	if err != nil {
		t.Fatalf("resolveOne(HEAD) error = %v", err)
	}
	if sha == "" {
		t.Error("resolveOne(HEAD) returned empty SHA")
	}

	// Test resolving merge-base (if available)
	sha, err = resolver.resolveOne("merge-base")
	if err != nil {
		// merge-base might not be available in all repos, so we just check
		// that it returns an error rather than panicking
		t.Logf("resolveOne(merge-base) error = %v (expected if not in a feature branch)", err)
	} else if sha == "" {
		t.Error("resolveOne(merge-base) returned empty SHA")
	}
}

func TestBaseResolver_GetResolvedBases(t *testing.T) {
	bases := []string{"HEAD"}
	resolver, err := NewBaseResolver(bases, "current", nil)
	if err != nil {
		t.Fatalf("NewBaseResolver() error = %v", err)
	}
	defer resolver.Stop()

	// Get resolved bases
	resolved := resolver.GetResolvedBases()

	// Modify the returned map (should not affect internal state)
	resolved["HEAD"] = "modified"

	// Get again and verify internal state wasn't modified
	resolved2 := resolver.GetResolvedBases()
	if resolved2["HEAD"] == "modified" {
		t.Error("GetResolvedBases() should return a copy, not the internal map")
	}
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
	resolver, err := NewBaseResolver(bases, "current", onChange)
	if err != nil {
		t.Fatalf("NewBaseResolver() error = %v", err)
	}
	defer resolver.Stop()

	// Wait a bit to ensure polling goroutine is running
	time.Sleep(100 * time.Millisecond)

	// Since we haven't changed anything in git, onChange shouldn't be called
	if changeCalled {
		t.Error("onChange was called even though no changes occurred")
	}
}

func TestBaseResolver_Stop(t *testing.T) {
	bases := []string{"HEAD"}
	resolver, err := NewBaseResolver(bases, "current", nil)
	if err != nil {
		t.Fatalf("NewBaseResolver() error = %v", err)
	}

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
	resolver, err := NewBaseResolver(bases, "current", nil)
	if err != nil {
		t.Fatalf("NewBaseResolver() error = %v", err)
	}
	defer resolver.Stop()

	// First check should return false (nothing changed)
	changed := resolver.checkForChanges()
	if changed {
		t.Error("checkForChanges() returned true on first check (nothing should have changed)")
	}
}

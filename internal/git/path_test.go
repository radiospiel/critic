package git

import (
	"path/filepath"
	"testing"
)

func TestAbsPathToGitPath(t *testing.T) {
	// Force initialization
	initPathCache()

	// Get the git root for testing
	gitRoot := gitRootCache

	// Test converting absolute path to git-relative
	absPath := filepath.Join(gitRoot, "docs", "CLI.md")
	gitPath := AbsPathToGitPath(absPath)

	expected := "docs/CLI.md"
	if gitPath != expected {
		t.Errorf("AbsPathToGitPath(%q) = %q, want %q", absPath, gitPath, expected)
	}

	t.Logf("Git root: %s", gitRoot)
	t.Logf("Abs path: %s", absPath)
	t.Logf("Git path: %s", gitPath)
}

func TestGitPathToDisplayPath(t *testing.T) {
	// Force initialization
	initPathCache()

	gitPath := "docs/CLI.md"
	displayPath := GitPathToDisplayPath(gitPath)

	t.Logf("Git root: %s", gitRootCache)
	t.Logf("Cwd: %s", cwdCache)
	t.Logf("Git path: %s", gitPath)
	t.Logf("Display path: %s", displayPath)
}

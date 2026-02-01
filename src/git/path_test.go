package git

import (
	"path/filepath"
	"testing"

	"github.com/radiospiel/critic/simple-go/assert"
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
	assert.Equals(t, gitPath, expected, "AbsPathToGitPath(%q)", absPath)

	t.Logf("Git root: %s", gitRoot)
	t.Logf("Abs path: %s", absPath)
	t.Logf("Git path: %s", gitPath)
}

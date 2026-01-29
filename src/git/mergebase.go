package git

import (
	"fmt"
	"strings"

	"github.com/radiospiel/critic/simple-go/must"
)

// GetMergeBase returns the merge base commit between HEAD and the main branch.
// It tries "main" first, then falls back to "master".
func GetMergeBase() string {
	// Try main first
	base, ok := TryGetMergeBaseWithBranch("main")
	if !ok {
		// Fall back to master
		base, ok = TryGetMergeBaseWithBranch("master")
		if !ok {
			panic("failed to find merge base with main or master")
		}
	}

	// Sanity check: git should always return valid commit hashes.
	if !validCommitHash.MatchString(base) {
		panic(fmt.Sprintf("git returned invalid merge base format: %s", base))
	}

	return base
}

// TryGetMergeBaseWithBranch gets the merge base between HEAD and the specified branch.
// Returns (base, true) on success, ("", false) if the branch doesn't exist or has no common ancestor.
func TryGetMergeBaseWithBranch(branch string) (string, bool) {
	output, err := must.TryExec("git", "merge-base", branch, "HEAD")
	if err != nil {
		return "", false
	}

	base := strings.TrimSpace(string(output))
	if base == "" {
		return "", false
	}

	return base, true
}

// GetMergeBaseBetween returns the merge base commit between two refs
func GetMergeBaseBetween(ref1, ref2 string) string {
	output := must.Exec("git", "merge-base", ref1, ref2)

	base := strings.TrimSpace(string(output))
	if base == "" {
		panic(fmt.Sprintf("empty merge base between %s and %s", ref1, ref2))
	}

	// Sanity check: git should always return valid commit hashes.
	if !validCommitHash.MatchString(base) {
		panic(fmt.Sprintf("git returned invalid merge base format: %s", base))
	}

	return base
}

// GetCurrentBranch returns the name of the current branch
func GetCurrentBranch() string {
	output := must.Exec("git", "rev-parse", "--abbrev-ref", "HEAD")
	return strings.TrimSpace(string(output))
}

// IsGitRepo checks if the current directory is inside a git repository
func IsGitRepo() bool {
	_, err := must.TryExec("git", "rev-parse", "--git-dir")
	return err == nil
}

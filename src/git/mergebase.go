package git

import (
	"strings"

	"github.com/radiospiel/critic/simple-go/must"
)

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

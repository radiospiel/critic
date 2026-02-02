package git

import (
	"strings"
)

// GetCurrentBranch returns the name of the current branch
func GetCurrentBranch() string {
	output := git("rev-parse", "--abbrev-ref", "HEAD")
	return strings.TrimSpace(string(output))
}

// IsGitRepo checks if the current directory is inside a git repository
func IsGitRepo() bool {
	_, err := tryGit("rev-parse", "--git-dir")
	return err == nil
}

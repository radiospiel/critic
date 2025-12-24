package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetMergeBase returns the merge base commit between HEAD and the main branch.
// It tries "main" first, then falls back to "master".
func GetMergeBase() (string, error) {
	// Try main first
	base, err := getMergeBaseWithBranch("main")
	if err == nil {
		return base, nil
	}

	// Fall back to master
	base, err = getMergeBaseWithBranch("master")
	if err != nil {
		return "", fmt.Errorf("failed to find merge base with main or master: %w", err)
	}

	return base, nil
}

// getMergeBaseWithBranch gets the merge base between HEAD and the specified branch
func getMergeBaseWithBranch(branch string) (string, error) {
	cmd := exec.Command("git", "merge-base", branch, "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	base := strings.TrimSpace(string(output))
	if base == "" {
		return "", fmt.Errorf("empty merge base")
	}

	return base, nil
}

// GetCurrentBranch returns the name of the current branch
func GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	branch := strings.TrimSpace(string(output))
	return branch, nil
}

// IsGitRepo checks if the current directory is inside a git repository
func IsGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	err := cmd.Run()
	return err == nil
}

package git

import (
	"fmt"
	"strings"
)

// GetMergeBase returns the merge base commit between HEAD and the main branch.
// It tries "main" first, then falls back to "master".
func GetMergeBase() (string, error) {
	return GetMergeBaseWithExecutor(defaultExecutor)
}

// GetMergeBaseWithExecutor returns the merge base using the provided executor.
func GetMergeBaseWithExecutor(executor CommandExecutor) (string, error) {
	// Try main first
	base, err := getMergeBaseWithBranch("main", executor)
	if err == nil {
		// Validate merge base to prevent command injection
		if !validCommitHash.MatchString(base) {
			return "", fmt.Errorf("invalid merge base format: %s", base)
		}
		return base, nil
	}

	// Fall back to master
	base, err = getMergeBaseWithBranch("master", executor)
	if err != nil {
		return "", fmt.Errorf("failed to find merge base with main or master: %w", err)
	}

	// Validate merge base to prevent command injection
	if !validCommitHash.MatchString(base) {
		return "", fmt.Errorf("invalid merge base format: %s", base)
	}

	return base, nil
}

// getMergeBaseWithBranch gets the merge base between HEAD and the specified branch
func getMergeBaseWithBranch(branch string, executor CommandExecutor) (string, error) {
	output, err := executor.Run("git", "merge-base", branch, "HEAD")
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
	return GetCurrentBranchWithExecutor(defaultExecutor)
}

// GetCurrentBranchWithExecutor returns the current branch using the provided executor.
func GetCurrentBranchWithExecutor(executor CommandExecutor) (string, error) {
	output, err := executor.Run("git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	branch := strings.TrimSpace(string(output))
	return branch, nil
}

// IsGitRepo checks if the current directory is inside a git repository
func IsGitRepo() bool {
	return IsGitRepoWithExecutor(defaultExecutor)
}

// IsGitRepoWithExecutor checks if current directory is a git repo using the provided executor.
func IsGitRepoWithExecutor(executor CommandExecutor) bool {
	_, err := executor.Run("git", "rev-parse", "--git-dir")
	return err == nil
}

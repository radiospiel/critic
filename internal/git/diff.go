package git

import (
	"fmt"
	"os/exec"
	"regexp"

	ctypes "git.15b.it/eno/critic/pkg/types"
)

// validCommitHash checks if a string is a valid git commit hash (SHA-1 or short form)
var validCommitHash = regexp.MustCompile(`^[a-f0-9]{6,40}$`)

// GetDiff returns the diff between the merge base and HEAD for the specified paths.
// If paths is empty, returns diff for all changed files.
func GetDiff(paths []string) (*ctypes.Diff, error) {
	// Get merge base
	base, err := GetMergeBase()
	if err != nil {
		return nil, fmt.Errorf("failed to get merge base: %w", err)
	}

	// Validate merge base to prevent command injection
	if !validCommitHash.MatchString(base) {
		return nil, fmt.Errorf("invalid merge base format: %s", base)
	}

	// Build git diff command
	// Compare merge base to working directory (includes committed, staged, and unstaged changes)
	args := []string{"diff", base, "--patch", "--no-color"}
	args = append(args, paths...)

	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		// Check if it's just an empty diff
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) == 0 {
			// Empty diff is valid
			return &ctypes.Diff{Files: []*ctypes.FileDiff{}}, nil
		}
		return nil, fmt.Errorf("failed to run git diff: %w", err)
	}

	// Parse the diff output
	diff, err := ParseDiff(string(output))
	if err != nil {
		return nil, fmt.Errorf("failed to parse diff: %w", err)
	}

	return diff, nil
}

// GetChangedFiles returns a list of files that have changed between merge base and HEAD
func GetChangedFiles(paths []string) ([]string, error) {
	// Get merge base
	base, err := GetMergeBase()
	if err != nil {
		return nil, fmt.Errorf("failed to get merge base: %w", err)
	}

	// Validate merge base to prevent command injection
	if !validCommitHash.MatchString(base) {
		return nil, fmt.Errorf("invalid merge base format: %s", base)
	}

	// Build git diff command to get file names only
	args := []string{"diff", base, "--name-only"}
	args = append(args, paths...)

	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	if len(output) == 0 {
		return []string{}, nil
	}

	// Split output into lines
	files := []string{}
	for _, line := range splitLines(string(output)) {
		if line != "" {
			files = append(files, line)
		}
	}

	return files, nil
}

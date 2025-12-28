package git

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	ctypes "git.15b.it/eno/critic/pkg/types"
)

// validCommitHash checks if a string is a valid git commit hash (SHA-1 or short form)
var validCommitHash = regexp.MustCompile(`^[a-f0-9]{6,40}$`)

// DiffMode represents the type of diff to show
type DiffMode int

const (
	DiffToMergeBase DiffMode = iota // Diff from merge base to working directory
	DiffToLastCommit                 // Diff of last commit (HEAD~1..HEAD)
	DiffUnstaged                     // Diff of unstaged changes only
)

// String returns the name of the diff mode
func (m DiffMode) String() string {
	switch m {
	case DiffToMergeBase:
		return "Merge Base"
	case DiffToLastCommit:
		return "Last Commit"
	case DiffUnstaged:
		return "Unstaged"
	default:
		return "Unknown"
	}
}

// GetDiff returns the diff for the specified paths and mode.
// If paths is empty, returns diff for all changed files.
func GetDiff(paths []string, mode DiffMode) (*ctypes.Diff, error) {
	// Build git diff command based on mode
	var args []string

	switch mode {
	case DiffToMergeBase:
		// Get merge base
		base, err := GetMergeBase()
		if err != nil {
			return nil, fmt.Errorf("failed to get merge base: %w", err)
		}

		// Sanity check: git should always return valid commit hashes.
		// If this fails, it indicates a catastrophic system failure.
		if !validCommitHash.MatchString(base) {
			panic(fmt.Sprintf("git returned invalid merge base format: %s", base))
		}

		// Compare merge base to working directory (includes committed, staged, and unstaged changes)
		args = []string{"diff", base, "--patch", "--no-color"}
		args = append(args, paths...)

	case DiffToLastCommit:
		// Show the last commit
		args = []string{"show", "HEAD", "--patch", "--no-color"}
		args = append(args, paths...)

	case DiffUnstaged:
		// Show only unstaged changes
		args = []string{"diff", "--patch", "--no-color"}
		args = append(args, paths...)
	}

	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		// If there's an error but output is empty, it might just be an empty diff
		if len(output) == 0 {
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

// GetDiffBetween returns the diff between a base commit and a target.
// base is a commit SHA, target is either "current" for working directory or a commit SHA.
// If paths is empty, returns diff for all changed files.
func GetDiffBetween(base, target string, paths []string) (*ctypes.Diff, error) {
	// Validate base commit
	if !validCommitHash.MatchString(base) {
		return nil, fmt.Errorf("invalid base commit SHA: %s", base)
	}

	// Build git diff command
	var args []string
	if target == "current" {
		// Compare base to working directory (includes committed, staged, and unstaged changes)
		args = []string{"diff", base, "--patch", "--no-color"}
	} else {
		// Validate target commit
		if !validCommitHash.MatchString(target) {
			return nil, fmt.Errorf("invalid target commit SHA: %s", target)
		}
		// Compare base to target commit
		args = []string{"diff", base + ".." + target, "--patch", "--no-color"}
	}
	args = append(args, paths...)

	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		// If there's an error but output is empty, it might just be an empty diff
		if len(output) == 0 {
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

// ResolveRef resolves a git reference (branch, tag, or commit) to a commit SHA
// Returns the resolved SHA or an error if the ref doesn't exist
func ResolveRef(ref string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--verify", ref)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to resolve ref %s: %w", ref, err)
	}

	sha := strings.TrimSpace(string(output))
	if !validCommitHash.MatchString(sha) {
		return "", fmt.Errorf("invalid commit SHA returned for ref %s: %s", ref, sha)
	}

	return sha, nil
}

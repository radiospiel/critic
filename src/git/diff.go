package git

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/radiospiel/critic/simple-go/must"
	"github.com/radiospiel/critic/simple-go/preconditions"
	ctypes "github.com/radiospiel/critic/src/pkg/types"
)

// GetGitRoot returns the root directory of the git repository
func GetGitRoot() string {
	output := must.Exec("git", "rev-parse", "--show-toplevel")
	return strings.TrimSpace(string(output))
}

// validCommitHash checks if a string is a valid git commit hash (SHA-1 or short form)
var validCommitHash = regexp.MustCompile(`^[a-f0-9]{6,40}$`)

// DiffMode represents the type of diff to show
type DiffMode int

const (
	DiffToMergeBase  DiffMode = iota // Diff from merge base to working directory
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

// Whitespace options for git diff. Using -w to ignore all whitespace and
// --ignore-blank-lines to ignore blank line changes.
// Other options available: -b (--ignore-space-change), --ignore-space-at-eol,
// --ignore-cr-at-eol
var diffWhitespaceOpts = []string{"-w", "--ignore-blank-lines"}

// GetDiff returns the diff for the specified paths and mode.
// If paths is empty, returns diff for all changed files.
func GetDiff(paths []string, mode DiffMode) (*ctypes.Diff, error) {
	// Build git diff command based on mode
	var args []string

	switch mode {
	case DiffToMergeBase:
		// Get merge base
		base := GetMergeBase()

		// Compare merge base to working directory (includes committed, staged, and unstaged changes)
		args = []string{"diff", base, "--patch", "--no-color"}
		args = append(args, diffWhitespaceOpts...)
		if len(paths) > 0 {
			args = append(args, "--")
			args = append(args, paths...)
		}

	case DiffToLastCommit:
		// Show the last commit
		args = []string{"show", "HEAD", "--patch", "--no-color"}
		args = append(args, diffWhitespaceOpts...)
		if len(paths) > 0 {
			args = append(args, "--")
			args = append(args, paths...)
		}

	case DiffUnstaged:
		// Show only unstaged changes
		args = []string{"diff", "--patch", "--no-color"}
		args = append(args, diffWhitespaceOpts...)
		if len(paths) > 0 {
			args = append(args, "--")
			args = append(args, paths...)
		}
	}

	output := must.Exec("git", args...)

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
	// if !validCommitHash.MatchString(base) {
	// 	base = ResolveRef(base)
	// 	if !validCommitHash.MatchString(base) {
	// 		return nil, fmt.Errorf("invalid base commit SHA: %s", base)
	// 	}
	// }

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
	args = append(args, diffWhitespaceOpts...)
	if len(paths) > 0 {
		args = append(args, "--")
		for _, path := range paths {
			preconditions.Check(len(path) > 0, "Path cannot be empty")
		}
		args = append(args, paths...)
	}

	output := must.Exec("git", args...)

	// Parse the diff output
	diff, err := ParseDiff(string(output))
	if err != nil {
		return nil, fmt.Errorf("failed to parse diff: %w", err)
	}

	return diff, nil
}

// ResolveRef resolves a git reference (branch, tag, or commit) to a commit SHA
func ResolveRef(ref string) string {
	output := must.Exec("git", "rev-parse", "--verify", ref)
	sha := strings.TrimSpace(string(output))
	if !validCommitHash.MatchString(sha) {
		panic(fmt.Sprintf("invalid commit SHA returned for ref %s: %s", ref, sha))
	}
	return sha
}

// IsCommitSHA checks if a string looks like a commit SHA (hexadecimal, 6-40 chars)
func IsCommitSHA(s string) bool {
	return validCommitHash.MatchString(s)
}

// HasRef checks if a git ref exists
func HasRef(ref string) bool {
	// Try to resolve the ref
	_, err := must.TryExec("git", "rev-parse", "--verify", ref)
	return err == nil
}

// GetDiffNamesBetween returns a diff summary (file metadata only, no hunks) between a base commit and a target.
// base is a commit SHA, target is either "current" for working directory or a commit SHA.
// This is more efficient than GetDiffBetween when you only need to know which files changed.
func GetDiffNamesBetween(base, target string) (*ctypes.Diff, error) {
	// Validate base commit
	// if !validCommitHash.MatchString(base) {
	// 	base = ResolveRef(base)
	// }
	// if !validCommitHash.MatchString(base) {
	// 	return nil, fmt.Errorf("invalid base commit SHA: %s", base)
	// }

	// Build git diff --name-status command
	var args []string
	if target == "current" {
		// Compare base to working directory
		args = []string{"diff", "--name-status", base}
	} else {
		// Validate target commit
		if !validCommitHash.MatchString(target) {
			return nil, fmt.Errorf("invalid target commit SHA: %s", target)
		}
		// Compare base to target commit
		args = []string{"diff", "--name-status", base + ".." + target}
	}
	args = append(args, diffWhitespaceOpts...)

	output := must.Exec("git", args...)

	// Parse the name-status output
	diff, err := ParseDiffNameStatus(string(output))
	if err != nil {
		return nil, fmt.Errorf("failed to parse diff name-status: %w", err)
	}

	return diff, nil
}

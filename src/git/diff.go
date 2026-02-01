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

// Whitespace options for git diff. Using -w to ignore all whitespace and
// --ignore-blank-lines to ignore blank line changes.
// Other options available: -b (--ignore-space-change), --ignore-space-at-eol,
// --ignore-cr-at-eol
var diffWhitespaceOpts = []string{"-w", "--ignore-blank-lines"}

// ResolveRef resolves a git reference (branch, tag, or commit) to a commit SHA
func ResolveRef(ref string) string {
	output := must.Exec("git", "rev-parse", "--verify", ref)
	sha := strings.TrimSpace(string(output))
	if !validCommitHash.MatchString(sha) {
		panic(fmt.Sprintf("invalid commit SHA returned for ref %s: %s", ref, sha))
	}
	return sha
}

// HasRef checks if a git ref exists
func HasRef(ref string) bool {
	// Try to resolve the ref
	_, err := must.TryExec("git", "rev-parse", "--verify", ref)
	return err == nil
}

// GetDiff returns the diff between a base commit and a target.
// base is a commit SHA, target is either "current" for working directory or a commit SHA.
// If paths is empty, returns diff for all changed files.
// contextLines specifies the number of context lines (minimum 3, default 3).
func GetDiff(base string, paths []string, contextLines int) (*ctypes.Diff, error) {
	for _, path := range paths {
		preconditions.Check(len(path) > 0, "Path cannot be empty")
	}

	// Ensure contextLines is at least 3
	if contextLines < 3 {
		contextLines = 3
	}

	// Build git diff command
	var args []string

	// Compare base to working directory (includes committed, staged, and unstaged changes)
	args = []string{"diff", base, "--merge-base", "--patch", "--no-color", fmt.Sprintf("--unified=%d", contextLines)}
	args = append(args, diffWhitespaceOpts...)
	if len(paths) > 0 {
		args = append(args, "--")
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

// GetDiffNames returns a diff summary (file metadata only, no hunks) between a base commit and a target.
// base is a commit SHA, target is either "current" for working directory or a commit SHA.
// This is more efficient than GetDiff when you only need to know which files changed.
func GetDiffNames(base string, paths []string) (*ctypes.Diff, error) {
	for _, path := range paths {
		preconditions.Check(len(path) > 0, "Path cannot be empty")
	}

	// Build git diff --name-status command
	var args []string

	// Compare base to working directory
	args = []string{"diff", "--merge-base", base, "--name-status"}
	args = append(args, diffWhitespaceOpts...)

	output := must.Exec("git", args...)

	// Parse the name-status output
	diff, err := ParseDiffNameStatus(string(output))
	if err != nil {
		return nil, fmt.Errorf("failed to parse diff name-status: %w", err)
	}

	return diff, nil
}

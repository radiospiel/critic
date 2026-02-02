package git

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/radiospiel/critic/simple-go/preconditions"
	ctypes "github.com/radiospiel/critic/src/pkg/types"
)

// GetGitRoot returns the root directory of the git repository
func GetGitRoot() string {
	output := git("rev-parse", "--show-toplevel")
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
	output := git("rev-parse", "--verify", ref)
	sha := strings.TrimSpace(string(output))
	if !validCommitHash.MatchString(sha) {
		panic(fmt.Sprintf("invalid commit SHA returned for ref %s: %s", ref, sha))
	}
	return sha
}

// HasRef checks if a git ref exists
func HasRef(ref string) bool {
	// Try to resolve the ref
	_, err := tryGit("rev-parse", "--verify", ref)
	return err == nil
}

// GetDiff returns the diff for a single file between a base commit and the working directory.
// base is a commit SHA, path is the file path to diff.
// contextLines specifies the number of context lines (minimum 3, default 3).
func GetDiff(base string, path string, contextLines int) (*ctypes.FileDiff, error) {
	preconditions.Check(len(path) > 0, "Path cannot be empty")

	// Ensure contextLines is at least 3
	if contextLines < 3 {
		contextLines = 3
	}

	// Check if file is untracked first
	if isUntracked(path) {
		return readUntrackedFile(path)
	}

	// Build git diff command
	args := []string{"diff", base, "--merge-base", "--patch", "--no-color", fmt.Sprintf("--unified=%d", contextLines)}
	args = append(args, diffWhitespaceOpts...)
	args = append(args, "--", path)

	output := git(args...)

	// Parse the diff output
	diff, err := ParseDiff(string(output))
	if err != nil {
		return nil, fmt.Errorf("failed to parse diff: %w", err)
	}

	if len(diff.Files) == 0 {
		return nil, nil
	}

	return diff.Files[0], nil
}

// isUntracked checks if a file is untracked by git
func isUntracked(path string) bool {
	output := git("ls-files", "--others", "--exclude-standard", "--", path)
	return strings.TrimSpace(string(output)) == path
}

// readUntrackedFile reads an untracked file and returns a FileDiff with all lines as added
func readUntrackedFile(path string) (*ctypes.FileDiff, error) {
	gitRoot := GetGitRoot()
	fullPath := filepath.Join(gitRoot, path)

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read untracked file %s: %w", path, err)
	}

	lines := strings.Split(string(content), "\n")
	// Remove trailing empty line if file ends with newline
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	hunkLines := make([]*ctypes.Line, len(lines))
	for i, line := range lines {
		hunkLines[i] = &ctypes.Line{
			Type:    ctypes.LineAdded,
			Content: line,
			OldNum:  0,
			NewNum:  i + 1,
		}
	}

	return &ctypes.FileDiff{
		OldPath:     "",
		NewPath:     path,
		IsUntracked: true,
		Hunks: []*ctypes.Hunk{
			{
				OldStart: 0,
				OldLines: 0,
				NewStart: 1,
				NewLines: len(lines),
				Header:   fmt.Sprintf("@@ -0,0 +1,%d @@", len(lines)),
				Lines:    hunkLines,
				Stats: ctypes.HunkStats{
					Added:   len(lines),
					Deleted: 0,
				},
			},
		},
	}, nil
}

// GetDiffNames returns a diff summary (file metadata only, no hunks) between a base commit and a target.
// base is a commit SHA, target is either "current" for working directory or a commit SHA.
// This is more efficient than GetDiff when you only need to know which files changed.
// Also includes untracked files (new files not yet added to git).
func GetDiffNames(base string, paths []string) (*ctypes.Diff, error) {
	for _, path := range paths {
		preconditions.Check(len(path) > 0, "Path cannot be empty")
	}

	// Build git diff --name-status command
	var args []string

	// Compare base to working directory
	args = []string{"diff", "--merge-base", base, "--name-status"}
	args = append(args, diffWhitespaceOpts...)

	output := git(args...)

	// Parse the name-status output
	diff, err := ParseDiffNameStatus(string(output))
	if err != nil {
		return nil, fmt.Errorf("failed to parse diff name-status: %w", err)
	}

	// Get untracked files and add them as untracked
	untrackedOutput := git("ls-files", "--others", "--exclude-standard")
	untrackedFiles := strings.Split(strings.TrimSpace(string(untrackedOutput)), "\n")
	for _, path := range untrackedFiles {
		if path == "" {
			continue
		}
		diff.Files = append(diff.Files, &ctypes.FileDiff{
			OldPath:     "",
			NewPath:     path,
			IsUntracked: true,
		})
	}

	return diff, nil
}

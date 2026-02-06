package git

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/samber/lo"

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
	files, err := ParseDiff(string(output))
	if err != nil {
		return nil, fmt.Errorf("failed to parse diff: %w", err)
	}

	if len(files) == 0 {
		return nil, nil
	}

	return files[0], nil
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
		OldPath:    "",
		NewPath:    path,
		FileStatus: ctypes.FileStatusUntracked,
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
//
// When paths is non-empty, only files under those paths are included. The paths are
// passed directly to git as pathspec arguments (after --) to filter at the source,
// avoiding the overhead of listing all changed files and filtering afterwards.
func GetDiffNames(base string, paths []string) ([]*ctypes.FileDiff, error) {
	for _, path := range paths {
		preconditions.Check(len(path) > 0, "Path cannot be empty")
	}

	// Build git diff --name-status command
	args := []string{"diff", "--merge-base", base, "--name-status"}
	args = append(args, diffWhitespaceOpts...)

	// Pass paths as pathspec arguments after -- to let git filter at the source
	if len(paths) > 0 {
		args = append(args, "--")
		args = append(args, paths...)
	}

	output := git(args...)

	// Parse the name-status output
	files, err := ParseDiffNameStatus(string(output))
	if err != nil {
		return nil, fmt.Errorf("failed to parse diff name-status: %w", err)
	}

	// Get untracked files and add them as untracked
	untrackedDiffs := getUntrackedDiffs(paths)
	files = append(files, untrackedDiffs...)

	return files, nil
}

// getUntrackedDiffs returns untracked files as FileDiff entries.
// When paths is non-empty, only untracked files under those paths are included.
func getUntrackedDiffs(paths []string) []*ctypes.FileDiff {
	args := []string{"ls-files", "--others", "--exclude-standard"}

	// Pass paths as pathspec arguments after -- to let git filter at the source
	if len(paths) > 0 {
		args = append(args, "--")
		args = append(args, paths...)
	}

	output := git(args...)
	files := lo.Filter(
		strings.Split(strings.TrimSpace(string(output)), "\n"),
		func(path string, _ int) bool { return path != "" },
	)
	return lo.Map(files, func(path string, _ int) *ctypes.FileDiff {
		return &ctypes.FileDiff{
			OldPath:    "",
			NewPath:    path,
			FileStatus: ctypes.FileStatusUntracked,
		}
	})
}

package git

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"git.15b.it/eno/critic/internal/config"
	ctypes "git.15b.it/eno/critic/pkg/types"
)

// GetUntrackedFiles returns a list of untracked files (respecting .gitignore)
// filtered by the given extensions. If extensions is nil or empty, all untracked
// files are returned.
func GetUntrackedFiles(paths []string, extensions []string) ([]string, error) {
	// Build git ls-files command
	args := []string{"ls-files", "--others", "--exclude-standard"}
	args = append(args, paths...)

	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list untracked files: %w", err)
	}

	// Parse output - one file per line
	var files []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		path := scanner.Text()
		if path == "" {
			continue
		}

		// Apply extension filtering
		if config.HasExtension(path, extensions) {
			files = append(files, path)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse untracked files: %w", err)
	}

	return files, nil
}

// GetUntrackedDiff returns a diff for untracked files, treating them as
// new files with all content as additions (diff against empty state).
func GetUntrackedDiff(paths []string, extensions []string) (*ctypes.Diff, error) {
	files, err := GetUntrackedFiles(paths, extensions)
	if err != nil {
		return nil, err
	}

	diff := &ctypes.Diff{
		Files: make([]*ctypes.FileDiff, 0, len(files)),
	}

	for _, path := range files {
		fileDiff, err := createUntrackedFileDiff(path)
		if err != nil {
			// Skip files we can't read (e.g., permission issues)
			continue
		}
		diff.Files = append(diff.Files, fileDiff)
	}

	return diff, nil
}

// createUntrackedFileDiff creates a FileDiff for an untracked file,
// treating all content as additions.
func createUntrackedFileDiff(path string) (*ctypes.FileDiff, error) {
	// Read file content
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	// Split into lines
	lines := strings.Split(string(content), "\n")

	// Create hunk with all lines as additions
	hunk := &ctypes.Hunk{
		OldStart: 0,
		OldLines: 0,
		NewStart: 1,
		NewLines: len(lines),
		Lines:    make([]*ctypes.Line, 0, len(lines)),
	}

	for i, line := range lines {
		hunk.Lines = append(hunk.Lines, &ctypes.Line{
			Type:    ctypes.LineAdded,
			Content: line,
			NewNum:  i + 1,
			OldNum:  0,
		})
	}

	return &ctypes.FileDiff{
		OldPath: "",
		NewPath: path,
		IsNew:   true,
		IsDeleted: false,
		Hunks:   []*ctypes.Hunk{hunk},
	}, nil
}

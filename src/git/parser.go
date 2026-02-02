package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/radiospiel/critic/simple-go/must"
	ctypes "github.com/radiospiel/critic/src/pkg/types"
)

var (
	// Regex patterns for parsing diff output
	fileHeaderRegex = regexp.MustCompile(`^diff --git a/(.+) b/(.+)$`)
	oldFileRegex    = regexp.MustCompile(`^--- (.+)$`)
	newFileRegex    = regexp.MustCompile(`^\+\+\+ (.+)$`)
	hunkHeaderRegex = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@ ?(.*)$`)
	oldModeRegex    = regexp.MustCompile(`^old mode (.+)$`)
	newModeRegex    = regexp.MustCompile(`^new mode (.+)$`)
	newFileMode     = regexp.MustCompile(`^new file mode (.+)$`)
	deletedFileMode = regexp.MustCompile(`^deleted file mode (.+)$`)
	renameFromRegex = regexp.MustCompile(`^rename from (.+)$`)
	renameToRegex   = regexp.MustCompile(`^rename to (.+)$`)
	binaryRegex     = regexp.MustCompile(`^Binary files (.+) and (.+) differ$`)
)

var (
	gitRootCache  string
	cwdCache      string
	pathCacheOnce sync.Once
)

// initPathCache initializes the cached git root and cwd (never changes during runtime)
func initPathCache() {
	pathCacheOnce.Do(func() {
		// Get git root
		cmd := exec.Command("git", "rev-parse", "--show-toplevel")
		output, err := cmd.Output()
		if err != nil {
			panic(fmt.Sprintf("failed to get git root: %v", err))
		}
		gitRootCache = strings.TrimSpace(string(output))

		// Get current working directory
		cwdCache = must.Must2(filepath.Abs("."))
	})
}

// AbsPathToGitPath converts an absolute filesystem path to a git-relative path
// Used for comparing fsnotify paths with git diff paths
func AbsPathToGitPath(absPath string) string {
	initPathCache()

	if absPath == "" || absPath == "/dev/null" {
		return absPath
	}

	// Make path relative to git root
	relPath, err := filepath.Rel(gitRootCache, absPath)
	if err != nil {
		// If we can't make it relative, return as-is
		return absPath
	}

	return relPath
}

// ParseDiff parses git diff output into a slice of FileDiff objects
func ParseDiff(diffText string) ([]*ctypes.FileDiff, error) {
	lines := splitLines(diffText)

	var files []*ctypes.FileDiff
	var currentFile *ctypes.FileDiff
	var currentHunk *ctypes.Hunk
	var oldLineNum, newLineNum int

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Check for file header
		if matches := fileHeaderRegex.FindStringSubmatch(line); matches != nil {
			// Save previous file if exists
			if currentFile != nil {
				if currentHunk != nil {
					currentFile.Hunks = append(currentFile.Hunks, currentHunk)
					currentHunk = nil
				}
				files = append(files, currentFile)
			}

			// Create new file diff
			currentFile = &ctypes.FileDiff{
				OldPath: matches[1],
				NewPath: matches[2],
				Hunks:   []*ctypes.Hunk{},
			}
			continue
		}

		if currentFile == nil {
			continue
		}

		// Check for file mode changes
		if matches := newFileMode.FindStringSubmatch(line); matches != nil {
			currentFile.FileStatus = ctypes.FileStatusNew
			currentFile.NewMode = matches[1]
			continue
		}

		if matches := deletedFileMode.FindStringSubmatch(line); matches != nil {
			currentFile.FileStatus = ctypes.FileStatusDeleted
			currentFile.OldMode = matches[1]
			continue
		}

		if matches := oldModeRegex.FindStringSubmatch(line); matches != nil {
			currentFile.OldMode = matches[1]
			continue
		}

		if matches := newModeRegex.FindStringSubmatch(line); matches != nil {
			currentFile.NewMode = matches[1]
			continue
		}

		if matches := renameFromRegex.FindStringSubmatch(line); matches != nil {
			currentFile.FileStatus = ctypes.FileStatusRenamed
			currentFile.OldPath = matches[1]
			continue
		}

		if matches := renameToRegex.FindStringSubmatch(line); matches != nil {
			currentFile.FileStatus = ctypes.FileStatusRenamed
			currentFile.NewPath = matches[1]
			continue
		}

		// Check for binary file
		if matches := binaryRegex.FindStringSubmatch(line); matches != nil {
			currentFile.IsBinary = true
			continue
		}

		// Skip --- and +++ lines (old and new file paths)
		if oldFileRegex.MatchString(line) || newFileRegex.MatchString(line) {
			continue
		}

		// Check for hunk header
		if matches := hunkHeaderRegex.FindStringSubmatch(line); matches != nil {
			// Save previous hunk if exists
			if currentHunk != nil {
				currentFile.Hunks = append(currentFile.Hunks, currentHunk)
			}

			// Parse hunk header
			oldStart, _ := strconv.Atoi(matches[1])
			oldLines := 1
			if matches[2] != "" {
				oldLines, _ = strconv.Atoi(matches[2])
			}

			newStart, _ := strconv.Atoi(matches[3])
			newLines := 1
			if matches[4] != "" {
				newLines, _ = strconv.Atoi(matches[4])
			}

			currentHunk = &ctypes.Hunk{
				OldStart: oldStart,
				OldLines: oldLines,
				NewStart: newStart,
				NewLines: newLines,
				Header:   matches[5],
				Lines:    []*ctypes.Line{},
			}

			oldLineNum = oldStart
			newLineNum = newStart
			continue
		}

		// Parse hunk lines
		if currentHunk != nil && len(line) > 0 {
			var lineType ctypes.LineType
			var content string
			var oldNum, newNum int

			switch line[0] {
			case '+':
				lineType = ctypes.LineAdded
				content = line[1:]
				oldNum = 0
				newNum = newLineNum
				newLineNum++
				currentHunk.Stats.Added++
			case '-':
				lineType = ctypes.LineDeleted
				content = line[1:]
				oldNum = oldLineNum
				newNum = 0
				oldLineNum++
				currentHunk.Stats.Deleted++
			case ' ':
				lineType = ctypes.LineContext
				content = line[1:]
				oldNum = oldLineNum
				newNum = newLineNum
				oldLineNum++
				newLineNum++
			default:
				// Skip other lines (like "\ No newline at end of file")
				continue
			}

			currentHunk.Lines = append(currentHunk.Lines, &ctypes.Line{
				Type:    lineType,
				Content: content,
				OldNum:  oldNum,
				NewNum:  newNum,
			})
		}
	}

	// Save last hunk and file
	if currentHunk != nil && currentFile != nil {
		currentFile.Hunks = append(currentFile.Hunks, currentHunk)
	}
	if currentFile != nil {
		files = append(files, currentFile)
	}

	return files, nil
}

// splitLines splits text into lines, handling both \n and \r\n
func splitLines(text string) []string {
	// TODO: rebuild using regexps
	// Replace \r\n with \n, then split on \n
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(text, "\n")

	// Remove empty last line if it exists
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return lines
}

// ParseDiffNameStatus parses `git diff --name-status` output into a slice of FileDiff.
// The output format is: STATUS\tPATH or STATUS\tOLD_PATH\tNEW_PATH for renames.
// Status codes: A=Added, D=Deleted, M=Modified, R=Renamed, C=Copied, T=Type changed
func ParseDiffNameStatus(output string) ([]*ctypes.FileDiff, error) {
	lines := splitLines(output)

	var files []*ctypes.FileDiff

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}

		status := parts[0]
		fileDiff := &ctypes.FileDiff{
			Hunks: []*ctypes.Hunk{}, // Empty hunks for name-status output
		}

		// Handle different status codes
		// Rename status can be R100, R095, etc. (with similarity percentage)
		if len(status) > 0 && (status[0] == 'R' || status[0] == 'C') {
			// Rename or Copy: has old path and new path
			if len(parts) >= 3 {
				if status[0] == 'R' {
					fileDiff.FileStatus = ctypes.FileStatusRenamed
				}
				fileDiff.OldPath = parts[1]
				fileDiff.NewPath = parts[2]
			}
		} else {
			fileDiff.OldPath = parts[1]
			fileDiff.NewPath = parts[1]

			switch status {
			case "A":
				fileDiff.FileStatus = ctypes.FileStatusNew
			case "D":
				fileDiff.FileStatus = ctypes.FileStatusDeleted
			case "M", "T":
				// Modified or Type change - default state (FileStatusModified)
			}
		}

		files = append(files, fileDiff)
	}

	return files, nil
}

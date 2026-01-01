package comments

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"git.15b.it/eno/critic/pkg/types"
)

// SyncComments synchronizes comments when the original file changes
// It applies the diff between oldContent and newContent to the critic file
// using git diff for accurate line mapping
func SyncComments(criticFile *types.CriticFile, oldContent, newContent []string) (*types.CriticFile, error) {
	// Build line mapping using git diff
	lineMapping, err := buildLineMappingWithGit(oldContent, newContent)
	if err != nil {
		return nil, fmt.Errorf("failed to build line mapping: %w", err)
	}

	// Create a new critic file with updated content and adjusted comment positions
	newCriticFile := &types.CriticFile{
		FilePath:      criticFile.FilePath,
		OriginalLines: make([]string, len(newContent)),
		Comments:      make(map[int]*types.CriticBlock),
	}
	copy(newCriticFile.OriginalLines, newContent)

	// Remap comments to their new positions
	for oldLine, comment := range criticFile.Comments {
		if newLine, exists := lineMapping[oldLine]; exists {
			newCriticFile.Comments[newLine] = &types.CriticBlock{
				LineNumber: newLine,
				Lines:      comment.Lines,
			}
		}
		// If the line doesn't exist in the mapping, the comment is dropped
		// (the line was deleted)
	}

	return newCriticFile, nil
}

// buildLineMappingWithGit uses git diff to build a mapping of old line numbers to new line numbers
func buildLineMappingWithGit(oldContent, newContent []string) (map[int]int, error) {
	// Create temporary files
	tmpDir, err := os.MkdirTemp("", "critic-diff-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	oldFile := filepath.Join(tmpDir, "old")
	newFile := filepath.Join(tmpDir, "new")

	// Write content to temp files
	if err := writeLines(oldFile, oldContent); err != nil {
		return nil, fmt.Errorf("failed to write old file: %w", err)
	}
	if err := writeLines(newFile, newContent); err != nil {
		return nil, fmt.Errorf("failed to write new file: %w", err)
	}

	// Run git diff
	cmd := exec.Command("git", "diff", "--no-index", "--unified=0", oldFile, newFile)
	output, _ := cmd.CombinedOutput() // git diff returns exit code 1 when there are differences

	// Parse the unified diff to build line mapping
	return parseUnifiedDiff(string(output), len(oldContent), len(newContent))
}

// Hunk represents a single hunk from a unified diff
type Hunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
}

var hunkHeaderPattern = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

// parseUnifiedDiff parses a unified diff and builds a line mapping
func parseUnifiedDiff(diff string, oldLen, newLen int) (map[int]int, error) {
	// Parse hunks from the diff
	var hunks []Hunk
	scanner := bufio.NewScanner(strings.NewReader(diff))

	for scanner.Scan() {
		line := scanner.Text()
		if matches := hunkHeaderPattern.FindStringSubmatch(line); matches != nil {
			hunk := Hunk{}

			// Parse old start and count
			hunk.OldStart, _ = strconv.Atoi(matches[1])
			if matches[2] != "" {
				hunk.OldCount, _ = strconv.Atoi(matches[2])
			} else {
				hunk.OldCount = 1
			}

			// Parse new start and count
			hunk.NewStart, _ = strconv.Atoi(matches[3])
			if matches[4] != "" {
				hunk.NewCount, _ = strconv.Atoi(matches[4])
			} else {
				hunk.NewCount = 1
			}

			hunks = append(hunks, hunk)
		}
	}

	// Build line mapping from hunks
	mapping := make(map[int]int)

	// If there are no hunks, all lines are unchanged
	if len(hunks) == 0 {
		for i := 0; i < oldLen && i < newLen; i++ {
			mapping[i] = i
		}
		return mapping, nil
	}

	// Process each section between hunks
	oldLine := 0
	newLine := 0

	for _, hunk := range hunks {
		// Map unchanged lines before this hunk
		// Git uses 1-based line numbers, we use 0-based
		hunkOldStart := hunk.OldStart - 1

		// For insertions (OldCount=0), we need to handle differently
		// The line at OldStart is context and should be mapped before the insertion
		if hunk.OldCount == 0 {
			// Map lines up to and including the context line
			for oldLine <= hunkOldStart && oldLine < oldLen {
				mapping[oldLine] = newLine
				oldLine++
				newLine++
			}
			// Now skip the inserted lines in the new file
			newLine += hunk.NewCount
		} else {
			// Map context lines before the hunk (not including OldStart)
			for oldLine < hunkOldStart {
				mapping[oldLine] = newLine
				oldLine++
				newLine++
			}
			// Skip the changed section
			// Lines in the old file that were deleted/changed don't get mapped
			oldLine += hunk.OldCount
			newLine += hunk.NewCount
		}
	}

	// Map any remaining unchanged lines after the last hunk
	for oldLine < oldLen && newLine < newLen {
		mapping[oldLine] = newLine
		oldLine++
		newLine++
	}

	return mapping, nil
}

// writeLines writes lines to a file
func writeLines(path string, lines []string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return err
		}
	}
	return writer.Flush()
}

// ValidateSync validates that the synchronized critic file is consistent
func ValidateSync(criticFile *types.CriticFile, expectedContent []string) error {
	return ValidateCriticFile(criticFile, expectedContent)
}

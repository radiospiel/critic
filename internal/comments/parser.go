package comments

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"git.15b.it/eno/critic/pkg/types"
)

var (
	// Pattern for CRITIC block opening fence: --- CRITIC <number> lines ------------------------------
	criticOpenPattern = regexp.MustCompile(`^---\s+CRITIC\s+(\d+)\s+lines?\s+-+$`)
	// Pattern for CRITIC block closing fence: --- CRITIC END ------------------------------
	criticClosePattern = regexp.MustCompile(`^---\s+CRITIC\s+END\s+-+$`)
)

// ParseCriticFile parses a .critic.md file and extracts the original content and comments
func ParseCriticFile(filePath string) (*types.CriticFile, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open critic file: %w", err)
	}
	defer file.Close()

	result := &types.CriticFile{
		FilePath:      filePath,
		OriginalLines: make([]string, 0),
		Comments:      make(map[int]*types.CriticBlock),
	}

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		line := scanner.Text()

		// Check if this is a CRITIC block opening
		if match := criticOpenPattern.FindStringSubmatch(line); match != nil {
			numLines, _ := strconv.Atoi(match[1])

			// Read the comment lines
			commentLines := make([]string, 0, numLines)
			for i := 0; i < numLines; i++ {
				if !scanner.Scan() {
					return nil, fmt.Errorf("unexpected end of file in CRITIC block at line %d", lineNum)
				}
				commentLines = append(commentLines, scanner.Text())
			}

			// Verify the closing fence
			if !scanner.Scan() {
				return nil, fmt.Errorf("missing CRITIC END fence at line %d", lineNum+numLines+1)
			}
			closeLine := scanner.Text()
			if !criticClosePattern.MatchString(closeLine) {
				return nil, fmt.Errorf("invalid CRITIC END fence at line %d: %s", lineNum+numLines+1, closeLine)
			}

			// Store the comment block
			result.Comments[lineNum] = &types.CriticBlock{
				LineNumber: lineNum,
				Lines:      commentLines,
			}
		} else {
			// This is an original file line
			result.OriginalLines = append(result.OriginalLines, line)
			lineNum++
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return result, nil
}

// FormatCriticFile formats a critic file with embedded comment blocks
func FormatCriticFile(criticFile *types.CriticFile) string {
	var builder strings.Builder

	for i, line := range criticFile.OriginalLines {
		// Check if there's a comment at this line
		if comment, exists := criticFile.Comments[i]; exists {
			// Write the CRITIC block
			numLines := len(comment.Lines)
			lineWord := "line"
			if numLines != 1 {
				lineWord = "lines"
			}
			builder.WriteString(fmt.Sprintf("--- CRITIC %d %s ------------------------------\n", numLines, lineWord))
			for _, commentLine := range comment.Lines {
				builder.WriteString(commentLine)
				builder.WriteString("\n")
			}
			builder.WriteString("--- CRITIC END ------------------------------\n")
		}

		// Write the original line
		builder.WriteString(line)
		builder.WriteString("\n")
	}

	// Check for comment after the last line
	if comment, exists := criticFile.Comments[len(criticFile.OriginalLines)]; exists {
		numLines := len(comment.Lines)
		lineWord := "line"
		if numLines != 1 {
			lineWord = "lines"
		}
		builder.WriteString(fmt.Sprintf("--- CRITIC %d %s ------------------------------\n", numLines, lineWord))
		for _, commentLine := range comment.Lines {
			builder.WriteString(commentLine)
			builder.WriteString("\n")
		}
		builder.WriteString("--- CRITIC END ------------------------------\n")
	}

	return builder.String()
}

// ValidateCriticFile validates that a critic file is well-formed
func ValidateCriticFile(criticFile *types.CriticFile, originalContent []string) error {
	// Check that all original lines are present
	if len(criticFile.OriginalLines) != len(originalContent) {
		return fmt.Errorf("line count mismatch: expected %d, got %d",
			len(originalContent), len(criticFile.OriginalLines))
	}

	for i, line := range originalContent {
		if i >= len(criticFile.OriginalLines) || criticFile.OriginalLines[i] != line {
			return fmt.Errorf("line %d mismatch: expected %q, got %q",
				i+1, line, criticFile.OriginalLines[i])
		}
	}

	// Check that all comment line numbers are valid
	for lineNum := range criticFile.Comments {
		if lineNum < 0 || lineNum > len(criticFile.OriginalLines) {
			return fmt.Errorf("invalid comment line number: %d (file has %d lines)",
				lineNum, len(criticFile.OriginalLines))
		}
	}

	return nil
}

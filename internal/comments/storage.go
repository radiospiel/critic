package comments

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"git.15b.it/eno/critic/internal/git"
	"git.15b.it/eno/critic/internal/logger"
)

const (
	criticStartPrefix = "--- CRITIC "
	criticEnd         = "--- CRITIC END"
)

// Comment represents a single review comment
type Comment struct {
	LineNumber int      // Line number in the file where the comment is attached
	Content    string   // The comment text (markdown)
	Lines      []string // Split content by lines for easier handling
}

// Storage handles reading and writing critic comments
type Storage struct {
	repoRoot string
}

// NewStorage creates a new comment storage
func NewStorage() (*Storage, error) {
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to get repo root: %w", err)
	}
	return &Storage{
		repoRoot: repoRoot,
	}, nil
}

// GetCriticFilePath returns the path to the critic comment file for a given source file
// For example: "main.go" -> ".main.go.critic.md"
func (s *Storage) GetCriticFilePath(filePath string) string {
	dir := filepath.Dir(filePath)
	base := filepath.Base(filePath)
	criticFile := "." + base + ".critic.md"

	// If filePath is absolute, return absolute path
	if filepath.IsAbs(filePath) {
		return filepath.Join(dir, criticFile)
	}

	// Otherwise return relative to repo root
	return filepath.Join(s.repoRoot, dir, criticFile)
}

// LoadComments loads all comments for a given file
func (s *Storage) LoadComments(filePath string) (map[int]*Comment, error) {
	criticPath := s.GetCriticFilePath(filePath)
	comments := make(map[int]*Comment)

	// Check if file exists
	if _, err := os.Stat(criticPath); os.IsNotExist(err) {
		// No comments file yet, return empty map
		return comments, nil
	}

	file, err := os.Open(criticPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open critic file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	var currentComment *Comment
	var commentLines []string

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Check for CRITIC start marker
		if strings.HasPrefix(line, criticStartPrefix) {
			// Parse: "--- CRITIC 5 of lines" -> line number is 5
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				num, err := strconv.Atoi(parts[2])
				if err == nil {
					currentComment = &Comment{
						LineNumber: num,
						Lines:      []string{},
					}
					commentLines = []string{}
				}
			}
			continue
		}

		// Check for CRITIC end marker
		if strings.HasPrefix(line, criticEnd) {
			if currentComment != nil {
				currentComment.Lines = commentLines
				currentComment.Content = strings.Join(commentLines, "\n")
				comments[currentComment.LineNumber] = currentComment
				currentComment = nil
				commentLines = nil
			}
			continue
		}

		// Accumulate comment content
		if currentComment != nil {
			commentLines = append(commentLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading critic file: %w", err)
	}

	logger.Info("LoadComments: Loaded %d comments from %s", len(comments), criticPath)
	return comments, nil
}

// SaveComment saves a single comment for a file at a specific line
func (s *Storage) SaveComment(filePath string, lineNumber int, content string) error {
	// Load existing comments
	comments, err := s.LoadComments(filePath)
	if err != nil {
		return err
	}

	// Add or update the comment
	lines := strings.Split(strings.TrimSpace(content), "\n")
	comments[lineNumber] = &Comment{
		LineNumber: lineNumber,
		Content:    content,
		Lines:      lines,
	}

	// Write all comments back
	return s.writeComments(filePath, comments)
}

// DeleteComment deletes a comment at a specific line
func (s *Storage) DeleteComment(filePath string, lineNumber int) error {
	// Load existing comments
	comments, err := s.LoadComments(filePath)
	if err != nil {
		return err
	}

	// Delete the comment
	delete(comments, lineNumber)

	// If no comments left, delete the file
	if len(comments) == 0 {
		criticPath := s.GetCriticFilePath(filePath)
		if err := os.Remove(criticPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove empty critic file: %w", err)
		}
		return nil
	}

	// Write remaining comments back
	return s.writeComments(filePath, comments)
}

// writeComments writes all comments to the critic file
func (s *Storage) writeComments(filePath string, comments map[int]*Comment) error {
	criticPath := s.GetCriticFilePath(filePath)

	// Create directory if it doesn't exist
	dir := filepath.Dir(criticPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create or truncate the file
	file, err := os.Create(criticPath)
	if err != nil {
		return fmt.Errorf("failed to create critic file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	// Sort line numbers for consistent output
	lineNumbers := make([]int, 0, len(comments))
	for lineNum := range comments {
		lineNumbers = append(lineNumbers, lineNum)
	}
	// Sort in ascending order
	for i := 0; i < len(lineNumbers); i++ {
		for j := i + 1; j < len(lineNumbers); j++ {
			if lineNumbers[i] > lineNumbers[j] {
				lineNumbers[i], lineNumbers[j] = lineNumbers[j], lineNumbers[i]
			}
		}
	}

	// Write each comment
	for _, lineNum := range lineNumbers {
		comment := comments[lineNum]
		numLines := len(comment.Lines)

		// Write CRITIC start marker
		fmt.Fprintf(writer, "%s%d of lines\n", criticStartPrefix, numLines)

		// Write comment content
		for _, line := range comment.Lines {
			fmt.Fprintf(writer, "%s\n", line)
		}

		// Write CRITIC end marker
		fmt.Fprintf(writer, "%s\n\n", criticEnd)
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to write critic file: %w", err)
	}

	logger.Info("SaveComment: Wrote %d comments to %s", len(comments), criticPath)
	return nil
}

// GetComment retrieves a single comment at a specific line
func (s *Storage) GetComment(filePath string, lineNumber int) (*Comment, error) {
	comments, err := s.LoadComments(filePath)
	if err != nil {
		return nil, err
	}

	comment, exists := comments[lineNumber]
	if !exists {
		return nil, nil
	}

	return comment, nil
}

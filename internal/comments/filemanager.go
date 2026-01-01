package comments

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"git.15b.it/eno/critic/pkg/types"
)

// FileManager handles saving and loading critic comment files
type FileManager struct {
	// Git repository root directory
	gitRoot string
	// Target name (e.g. "current", "main", "HEAD")
	target string
}

// NewFileManager creates a new file manager
func NewFileManager(gitRoot string, target string) *FileManager {
	return &FileManager{
		gitRoot: gitRoot,
		target:  target,
	}
}

// GetCriticDir returns the base .critic directory path
func (fm *FileManager) GetCriticDir() string {
	return filepath.Join(fm.gitRoot, ".critic", fm.target)
}

// GetCriticCommentsPath returns the path to the .comments file for a given original file
// e.g., .critic/current/src/main.go.comments
func (fm *FileManager) GetCriticCommentsPath(originalPath string) string {
	return filepath.Join(fm.GetCriticDir(), originalPath+".comments")
}

// GetCriticOriginalPath returns the path to the original file copy
// e.g., .critic/current/src/main.go
func (fm *FileManager) GetCriticOriginalPath(originalPath string) string {
	return filepath.Join(fm.GetCriticDir(), originalPath)
}

// LoadComments loads comments from a .comments file
func (fm *FileManager) LoadComments(originalPath string) (*types.CriticFile, error) {
	commentsPath := fm.GetCriticCommentsPath(originalPath)

	// Check if the comments file exists
	if _, err := os.Stat(commentsPath); os.IsNotExist(err) {
		// No comments file exists, return empty comment structure
		// Read from the working directory original file
		originalLines, err := fm.readFileLines(originalPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read original file: %w", err)
		}

		return &types.CriticFile{
			FilePath:      originalPath,
			OriginalLines: originalLines,
			Comments:      make(map[int]*types.CriticBlock),
		}, nil
	}

	// Parse the existing comments file
	criticFile, err := ParseCriticFile(commentsPath)
	if err != nil {
		return nil, err
	}

	// Set the correct file path (parser sets it to the .comments file path)
	criticFile.FilePath = originalPath
	return criticFile, nil
}

// SaveComments saves comments to .comments and original copy files
func (fm *FileManager) SaveComments(criticFile *types.CriticFile) error {
	originalCopyPath := fm.GetCriticOriginalPath(criticFile.FilePath)
	commentsPath := fm.GetCriticCommentsPath(criticFile.FilePath)

	// Ensure the directory exists
	dir := filepath.Dir(commentsPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Save the original copy (just the original content without comments)
	if err := fm.writeFileLines(originalCopyPath, criticFile.OriginalLines); err != nil {
		return fmt.Errorf("failed to write original copy: %w", err)
	}

	// Save the .comments file (original content with embedded comments)
	formattedContent := FormatCriticFile(criticFile)
	if err := os.WriteFile(commentsPath, []byte(formattedContent), 0644); err != nil {
		return fmt.Errorf("failed to write comments file: %w", err)
	}

	return nil
}

// HasComments checks if a file has any critic comments
func (fm *FileManager) HasComments(originalPath string) bool {
	commentsPath := fm.GetCriticCommentsPath(originalPath)

	// Check if comments file exists
	if _, err := os.Stat(commentsPath); os.IsNotExist(err) {
		return false
	}

	// Check if it actually has comments (not just an empty file)
	criticFile, err := fm.LoadComments(originalPath)
	if err != nil {
		return false
	}

	return len(criticFile.Comments) > 0
}

// DeleteComments removes the critic comment files for a given original file
func (fm *FileManager) DeleteComments(originalPath string) error {
	commentsPath := fm.GetCriticCommentsPath(originalPath)
	originalCopyPath := fm.GetCriticOriginalPath(originalPath)

	// Remove both files, ignoring "not found" errors
	if err := os.Remove(commentsPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove comments file: %w", err)
	}

	if err := os.Remove(originalCopyPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove original copy: %w", err)
	}

	return nil
}

// readFileLines reads all lines from a file
func (fm *FileManager) readFileLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

// writeFileLines writes lines to a file
func (fm *FileManager) writeFileLines(path string, lines []string) error {
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

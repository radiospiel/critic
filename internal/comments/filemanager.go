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
	// Base directory for resolving relative paths
	baseDir string
}

// NewFileManager creates a new file manager
func NewFileManager(baseDir string) *FileManager {
	return &FileManager{
		baseDir: baseDir,
	}
}

// GetCriticFilePath returns the path to the .critic.md file for a given original file
func (fm *FileManager) GetCriticFilePath(originalPath string) string {
	return originalPath + ".critic.md"
}

// GetCriticOriginalPath returns the path to the .critic.original file for a given original file
func (fm *FileManager) GetCriticOriginalPath(originalPath string) string {
	return originalPath + ".critic.original"
}

// LoadComments loads comments from a .critic.md file
func (fm *FileManager) LoadComments(originalPath string) (*types.CriticFile, error) {
	criticPath := fm.GetCriticFilePath(originalPath)

	// Check if the critic file exists
	if _, err := os.Stat(criticPath); os.IsNotExist(err) {
		// No comments file exists, return empty comment structure
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

	// Parse the existing critic file
	return ParseCriticFile(criticPath)
}

// SaveComments saves comments to .critic.md and .critic.original files
func (fm *FileManager) SaveComments(criticFile *types.CriticFile) error {
	// Ensure the directory exists
	dir := filepath.Dir(criticFile.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Save the .critic.original file (just the original content without comments)
	originalPath := fm.GetCriticOriginalPath(criticFile.FilePath)
	if err := fm.writeFileLines(originalPath, criticFile.OriginalLines); err != nil {
		return fmt.Errorf("failed to write critic.original file: %w", err)
	}

	// Save the .critic.md file (original content with embedded comments)
	criticPath := fm.GetCriticFilePath(criticFile.FilePath)
	formattedContent := FormatCriticFile(criticFile)
	if err := os.WriteFile(criticPath, []byte(formattedContent), 0644); err != nil {
		return fmt.Errorf("failed to write critic.md file: %w", err)
	}

	return nil
}

// HasComments checks if a file has any critic comments
func (fm *FileManager) HasComments(originalPath string) bool {
	criticPath := fm.GetCriticFilePath(originalPath)
	_, err := os.Stat(criticPath)
	return err == nil
}

// DeleteComments removes the critic comment files for a given original file
func (fm *FileManager) DeleteComments(originalPath string) error {
	criticPath := fm.GetCriticFilePath(originalPath)
	originalBackupPath := fm.GetCriticOriginalPath(originalPath)

	// Remove both files, ignoring "not found" errors
	if err := os.Remove(criticPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove critic.md file: %w", err)
	}

	if err := os.Remove(originalBackupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove critic.original file: %w", err)
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

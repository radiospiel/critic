package git

import "os"

// GetFileContent retrieves file content from either the working directory or a git revision.
// If revision is empty, reads from the working directory.
// Otherwise, reads from git at the specified revision (e.g., "HEAD").
func GetFileContent(path string, revision string) (string, error) {
	return GetFileContentWithExecutor(path, revision, defaultExecutor)
}

// GetFileContentWithExecutor retrieves file content using the provided executor.
// This is the testable version that accepts a custom executor.
func GetFileContentWithExecutor(path string, revision string, executor CommandExecutor) (string, error) {
	if revision == "" {
		// Read from working directory
		content, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(content), nil
	}

	// Read from git at specific revision
	output, err := executor.Run("git", "show", revision+":"+path)
	if err != nil {
		return "", err
	}

	return string(output), nil
}

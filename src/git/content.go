package git

import (
	"os"

	"github.com/radiospiel/critic/simple-go/must"
)

// GetFileContent retrieves file content from either the working directory or a git revision.
// If revision is empty, reads from the working directory.
// Otherwise, reads from git at the specified revision (e.g., "HEAD").
func GetFileContent(path string, revision string) (string, error) {
	if revision == "" {
		// Read from working directory
		content, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(content), nil
	}

	// Read from git at specific revision
	output := must.Exec("git", "show", revision+":"+path)
	return string(output), nil
}

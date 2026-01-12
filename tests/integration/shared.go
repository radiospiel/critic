package critic_integration

import (
	"fmt"
	"os"
	"testing"

	"git.15b.it/eno/critic/simple-go/must"
)

func TestMain(m *testing.M) {
	fmt.Println("⚠️  WARNING: Integration tests must run with -p 1 flag")
	fmt.Println("   Run: go test -p 1 -v")
	fmt.Println()
	os.Exit(m.Run())
}

// SetupGitRepo creates a temporary git repository for testing and changes into it.
// IMPORTANT: Tests using this function cannot run in parallel due to os.Chdir().
// Use -p 1 flag when running tests: go test -p 1 -v
func SetupGitRepo(t *testing.T) {
	t.Helper()

	// Save and restore original directory
	originalDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(originalDir) })

	tmpDir := t.TempDir()
	os.Chdir(tmpDir)

	// Initialize git repo
	must.Exec("git", "init")

	// Configure git
	must.Exec("git", "config", "user.name", "Test User")
	must.Exec("git", "config", "user.email", "test@example.com")
}

// CommitFile commits an existing file in the current directory
func CommitFile(t *testing.T, filename string) {
	t.Helper()
	must.Exec("git", "add", filename)
	must.Exec("git", "commit", "-m", "commit "+filename)
}

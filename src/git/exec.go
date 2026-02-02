package git

import (
	"github.com/radiospiel/critic/simple-go/must"
)

// git executes a git command and returns the output, panicking on error
func git(args ...string) []byte {
	return must.Exec("git", args...)
}

// tryGit executes a git command and returns the output and error
func tryGit(args ...string) ([]byte, error) {
	return must.TryExec("git", args...)
}

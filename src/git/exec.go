package git

import (
	"runtime"

	"github.com/radiospiel/critic/simple-go/logger"
	"github.com/radiospiel/critic/simple-go/must"
)

// git executes a git command and returns the output, panicking on error.
func git(args ...string) []byte {
	_, file, line, _ := runtime.Caller(1)
	output, err := must.TryExecWithCaller(logger.WithCaller(file, line), "git", args...)
	if err != nil {
		panic(err)
	}
	return output
}

// tryGit executes a git command and returns the output and error.
func tryGit(args ...string) ([]byte, error) {
	_, file, line, _ := runtime.Caller(1)
	return must.TryExecWithCaller(logger.WithCaller(file, line), "git", args...)
}

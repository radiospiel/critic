package must

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"

	sio "git.15b.it/eno/critic/simple-go/io"
	"git.15b.it/eno/critic/simple-go/logger"
	"github.com/samber/lo"
)

// MaxPipeBufferSize is the maximum number of bytes that PipeInto will
// buffer before truncating. Default is 10 MB.
const MaxPipeBufferSize = 10 * 1024 * 1024

// PipeInto executes a command and pipes its stdout through the given
// SectionPipe, returning the filtered output. It runs command execution
// and output reading in parallel to avoid blocking on pipe buffer limits.
//
// The command's stdout is connected to the pipe's writer. The filtered
// output is read from the pipe's reader into a memory buffer (up to
// MaxPipeBufferSize bytes).
//
// Panics if the command fails.
func PipeInto(pipe *sio.SectionPipe, name string, args ...string) []byte {
	// Log the command
	stringified := lo.Map(args, escapeIfNecessary)
	logger.Info("%s %s | <pipe>", name, strings.Join(stringified, " "))

	// Create the pipe endpoints
	pr, pw := pipe.Pipe()

	// Set up the command
	cmd := exec.Command(name, args...)
	cmd.Stdout = pw

	// Buffer for collecting filtered output
	var output bytes.Buffer
	var readErr error

	// Read from pipe in a goroutine (parallel with command execution)
	done := make(chan struct{})
	go func() {
		defer close(done)
		// Use LimitedReader to enforce max buffer size
		limited := &io.LimitedReader{R: pr, N: MaxPipeBufferSize}
		_, readErr = io.Copy(&output, limited)
		// Drain any remaining data to prevent writer blocking
		io.Copy(io.Discard, pr)
		pr.Close()
	}()

	// Run the command
	err := cmd.Run()

	// Close the writer to signal EOF to the filter goroutine
	pw.Close()

	// Wait for reading to complete
	<-done

	// Handle errors
	if err != nil {
		msg := fmt.Sprintf("in %s: PipeInto(%s %v) failed: %v", Getwd(), name, args, err)
		if exitErr, ok := err.(*exec.ExitError); ok {
			msg = msg + "\nerrout: " + string(exitErr.Stderr)
		}
		panic(msg)
	}

	if readErr != nil && readErr != io.EOF {
		panic(fmt.Sprintf("PipeInto: read error: %v", readErr))
	}

	return output.Bytes()
}

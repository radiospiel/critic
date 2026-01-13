package io

import (
	"bufio"
	"io"
	"os"
	"strings"
)

// SectionPipe is a filtering pipe that skips a number of lines and then
// forwards a limited number of lines. It's useful for extracting a specific
// range of lines from a stream (similar to `sed -n 'start,endp'`).
type SectionPipe struct {
	skip int
	take int
}

// NewSectionPipe creates a SectionPipe that skips `skip` lines
// and then forwards the next `take` lines.
//
// Example: NewSectionPipe(4, 6) skips 4 lines and takes 6 lines,
// equivalent to `sed -n '5,10p'`.
func NewSectionPipe(skip, take int) *SectionPipe {
	return &SectionPipe{
		skip: skip,
		take: take,
	}
}

// NewSectionPipeLines creates a SectionPipe that extracts lines from
// startLine to endLine (1-indexed, inclusive). This is equivalent to
// `sed -n 'startLine,endLinep'`.
//
// Example: NewSectionPipeLines(5, 10) extracts lines 5 through 10.
func NewSectionPipeLines(startLine, endLine int) *SectionPipe {
	if startLine < 1 {
		startLine = 1
	}
	skip := startLine - 1
	take := endLine - startLine + 1
	if take < 0 {
		take = 0
	}
	return &SectionPipe{
		skip: skip,
		take: take,
	}
}

// Pipe creates and returns a connected PipeReader and PipeWriter.
// Data written to the PipeWriter is filtered (lines are skipped/taken
// according to the SectionPipe configuration) and the filtered output
// can be read from the PipeReader.
//
// A goroutine is started internally to perform the filtering. The
// PipeWriter should be closed when done writing to signal EOF.
func (sp *SectionPipe) Pipe() (*io.PipeReader, *io.PipeWriter) {
	// Create the input pipe (where data is written)
	inputR, inputW := io.Pipe()
	// Create the output pipe (where filtered data is read)
	outputR, outputW := io.Pipe()

	// Start the filtering goroutine
	go sp.filter(inputR, outputW)

	return outputR, inputW
}

// filter reads from the input pipe, applies line filtering, and writes
// to the output pipe. It runs as a goroutine.
func (sp *SectionPipe) filter(r *io.PipeReader, w *io.PipeWriter) {
	scanner := bufio.NewScanner(r)
	lineNo := 0
	written := 0
	outputClosed := false

	for scanner.Scan() {
		lineNo++

		// Skip lines until we've skipped enough
		if lineNo <= sp.skip {
			continue
		}

		// Stop forwarding if we've taken enough lines, but keep
		// draining the input to prevent SIGPIPE on the writer
		if written >= sp.take {
			if !outputClosed {
				w.Close()
				outputClosed = true
			}
			// Continue draining, but don't write
			continue
		}

		// Write the line to output
		w.Write(scanner.Bytes())
		w.Write([]byte{'\n'})
		written++
	}

	// Close the output if not already closed
	if !outputClosed {
		w.Close()
	}

	// Close the input after fully draining
	r.Close()
}

// ReadFileLines reads a range of lines from a file.
// It reads from startLine to endLine (1-indexed, inclusive).
// This is a pure Go implementation that doesn't spawn any external process.
func ReadFileLines(path string, startLine, endLine int) (string, error) {
	if startLine < 1 {
		startLine = 1
	}

	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNo := 0
	var result strings.Builder

	for scanner.Scan() {
		lineNo++

		if lineNo < startLine {
			continue
		}
		if lineNo > endLine {
			break
		}

		result.Write(scanner.Bytes())
		result.WriteByte('\n')
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return result.String(), nil
}

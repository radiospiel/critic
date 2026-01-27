package git

import (
	"bufio"
	"os"
	"strings"
)

// readFileLines reads a range of lines from a file.
// It reads from startLine to endLine (1-indexed, inclusive).
// This is a pure Go implementation that doesn't spawn any external process.
func readFileLines(path string, startLine, endLine int) (string, error) {
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

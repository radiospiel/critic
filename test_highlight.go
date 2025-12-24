package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"git.15b.it/eno/critic/internal/highlight"
	"github.com/charmbracelet/lipgloss"
)

func main() {
	// Open main.go
	file, err := os.Open("cmd/critic/main.go")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// Create highlighter
	h := highlight.NewHighlighter()

	// Background color code (dark greenish)
	bgColor := "\x1b[48;2;26;58;26m"

	// Read first 10 lines
	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() && lineNum < 10 {
		lineNum++
		line := scanner.Text()

		// Highlight the line
		highlighted := h.HighlightLine(line, "main.go")

		// Simulate applyLineBackground logic
		// 1. Strip background codes (none in our case)
		cleaned := highlighted

		// 2. Replace full resets with foreground-only resets
		processed := strings.Replace(cleaned, "\x1b[0m", "\x1b[39m", -1)
		processed = strings.Replace(processed, "\x1b[m", "\x1b[39m", -1)

		// 3. Calculate padding (or truncate) - use lipgloss.Width like real code
		width := 80
		visibleWidth := lipgloss.Width(cleaned) // Proper width accounting for tabs/ANSI

		fmt.Printf("  Raw len=%d, Visible width=%d\n", len(line), visibleWidth)

		var final string
		if visibleWidth > width {
			// Truncate long lines - simulate what the app does
			fmt.Printf("  (Line too long, would truncate to %d)\n", width)
			final = processed[:width] // Simple truncation for test
		} else {
			// Add padding
			paddingWidth := width - visibleWidth
			fmt.Printf("  Padding width=%d\n", paddingWidth)
			final = processed + strings.Repeat(" ", paddingWidth)
		}

		// 4. Build final with background
		withBg := bgColor + final + "\x1b[0m"

		// Print results
		fmt.Printf("Line %d (original): %q\n", lineNum, line)
		fmt.Printf("Line %d (with background codes): %q\n", lineNum, withBg)
		fmt.Printf("Line %d (rendered with bg): %s\n", lineNum, withBg)
		fmt.Printf("\n")
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading: %v\n", err)
		os.Exit(1)
	}
}

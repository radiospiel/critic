package ui

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var bgCodeRegex = regexp.MustCompile(`\x1b\[4[0-9](;[0-9]+)*m`)

// stripBackgroundCodes removes background color ANSI codes from a string
func stripBackgroundCodes(s string) string {
	// Remove all background color codes (40-49 are background colors)
	return bgCodeRegex.ReplaceAllString(s, "")
}

// expandTabsInANSI expands tabs to spaces while preserving ANSI codes
func expandTabsInANSI(s string) string {
	const tabWidth = 4
	var result strings.Builder
	col := 0 // Current column position (visible characters)
	inANSI := false

	for i := 0; i < len(s); i++ {
		ch := s[i]

		// Track ANSI escape sequences (don't count toward column position)
		if ch == '\x1b' {
			inANSI = true
			result.WriteByte(ch)
			continue
		}

		if inANSI {
			result.WriteByte(ch)
			if ch == 'm' {
				inANSI = false
			}
			continue
		}

		// Expand tabs
		if ch == '\t' {
			// Calculate spaces needed to reach next tab stop
			spacesToAdd := tabWidth - (col % tabWidth)
			result.WriteString(strings.Repeat(" ", spacesToAdd))
			col += spacesToAdd
		} else {
			result.WriteByte(ch)
			col++
		}
	}

	return result.String()
}

// truncateANSI truncates a string to maxWidth while preserving ANSI codes
func truncateANSI(s string, maxWidth int) string {
	// Use lipgloss to truncate while preserving ANSI codes
	return lipgloss.NewStyle().MaxWidth(maxWidth).Render(s)
}

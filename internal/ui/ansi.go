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

// truncateANSI truncates a string to maxWidth while preserving ANSI codes
func truncateANSI(s string, maxWidth int) string {
	// Use lipgloss to truncate while preserving ANSI codes
	return lipgloss.NewStyle().MaxWidth(maxWidth).Render(s)
}

package tui

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// match all background color codes:
// - \x1b[4[0-9]m - basic 16-color backgrounds (40-49)
// - \x1b[48;5;NNNm - 256-color backgrounds
// - \x1b[48;2;R;G;Bm - true color backgrounds
var bgCodeRegex = regexp.MustCompile(`\x1b\[(?:4[0-9]|48;[25];[0-9;]+)m`)
var resetCodeRegex = regexp.MustCompile(`\x1b\[0?m`)

// stripBackgroundCodes removes background color ANSI codes from a string
func stripBackgroundCodes(s string) string {
	// Remove all background color codes (40-49 are background colors)
	return bgCodeRegex.ReplaceAllString(s, "")
}

// stripResetCodes removes all ANSI reset codes from a string
func stripResetCodes(s string) string {
	// Remove \x1b[0m and \x1b[m (reset codes)
	return resetCodeRegex.ReplaceAllString(s, "")
}

// stripAllStyleCodes removes all ANSI style codes except foreground colors
func stripAllStyleCodes(s string) string {
	// First strip backgrounds
	s = stripBackgroundCodes(s)
	// Then strip resets
	s = stripResetCodes(s)

	// Also strip any compound SGR sequences that might have backgrounds mixed in
	// match sequences like \x1b[1;32;42m and remove background parts
	compoundRegex := regexp.MustCompile(`\x1b\[([0-9;]+)m`)
	s = compoundRegex.ReplaceAllStringFunc(s, func(match string) string {
		// Extract the parameters
		params := match[2 : len(match)-1] // Remove \x1b[ and m
		parts := regexp.MustCompile(`;`).Split(params, -1)

		// Keep only foreground color codes (30-37, 38, 90-97)
		var fgParts []string
		i := 0
		for i < len(parts) {
			code := parts[i]
			// Foreground colors: 30-37 (basic), 90-97 (bright), 38 (extended)
			if (code >= "30" && code <= "37") || (code >= "90" && code <= "97") {
				fgParts = append(fgParts, code)
				i++
			} else if code == "38" && i+2 < len(parts) {
				// 38;5;N or 38;2;R;G;B
				if parts[i+1] == "5" && i+2 < len(parts) {
					fgParts = append(fgParts, code, parts[i+1], parts[i+2])
					i += 3
				} else if parts[i+1] == "2" && i+4 < len(parts) {
					fgParts = append(fgParts, code, parts[i+1], parts[i+2], parts[i+3], parts[i+4])
					i += 5
				} else {
					i++
				}
			} else {
				// Skip non-foreground codes
				i++
			}
		}

		if len(fgParts) > 0 {
			return "\x1b[" + regexp.MustCompile(`;`).ReplaceAllString(strings.Join(fgParts, ";"), ";") + "m"
		}
		return ""
	})

	return s
}

// truncateANSI truncates a string to maxWidth while preserving ANSI codes
func truncateANSI(s string, maxWidth int) string {
	// Use lipgloss to truncate while preserving ANSI codes
	// Inline(true) prevents word wrapping
	return lipgloss.NewStyle().MaxWidth(maxWidth).Inline(true).Render(s)
}

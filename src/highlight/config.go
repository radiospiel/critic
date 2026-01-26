package highlight

import "strings"

// TabWidth returns the tab width for a given language
func TabWidth(language string) int {
	// Language-specific tab widths
	switch strings.ToLower(language) {
	case "ruby", "rb":
		return 2
	case "go", "golang":
		return 4
	default:
		return 4
	}
}

// expandTabs expands tab characters to spaces based on tab width
func expandTabs(code string, tabWidth int) string {
	var result strings.Builder
	col := 0 // Current column position

	for i := 0; i < len(code); i++ {
		ch := code[i]

		if ch == '\t' {
			// Calculate spaces needed to reach next tab stop
			spacesToAdd := tabWidth - (col % tabWidth)
			result.WriteString(strings.Repeat(" ", spacesToAdd))
			col += spacesToAdd
		} else if ch == '\n' {
			result.WriteByte(ch)
			col = 0 // Reset column on newline
		} else {
			result.WriteByte(ch)
			col++
		}
	}

	return result.String()
}

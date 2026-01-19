package teapot

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ansiRegex matches ANSI escape sequences
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// StripANSI removes ANSI escape sequences from a string.
func StripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// PrintableWidth returns the visible width of a string, ignoring ANSI sequences.
func PrintableWidth(s string) int {
	return len([]rune(StripANSI(s)))
}

// ParseANSILine parses an ANSI-encoded line and returns cells with styles.
// This properly handles escape sequences and extracts visible characters with their styles.
func ParseANSILine(line string) []Cell {
	var cells []Cell
	var currentStyle lipgloss.Style

	i := 0
	runes := []rune(line)
	for i < len(runes) {
		r := runes[i]

		// Check for ANSI escape sequence
		if r == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			// Find the end of the escape sequence
			j := i + 2
			for j < len(runes) && !isANSITerminator(runes[j]) {
				j++
			}
			if j < len(runes) {
				// Parse the escape sequence and update style
				seq := string(runes[i : j+1])
				currentStyle = applyANSISequence(currentStyle, seq)
				i = j + 1
				continue
			}
		}

		// Regular visible character
		cells = append(cells, Cell{Rune: r, Style: currentStyle})
		i++
	}

	return cells
}

// isANSITerminator returns true if r is an ANSI sequence terminator
func isANSITerminator(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}

// applyANSISequence updates a style based on an ANSI escape sequence
func applyANSISequence(style lipgloss.Style, seq string) lipgloss.Style {
	// Extract parameters from sequence like "\x1b[38;5;220m"
	if len(seq) < 3 {
		return style
	}

	// Get the part between '[' and the terminator
	inner := seq[2 : len(seq)-1]
	terminator := seq[len(seq)-1]

	// Only handle SGR (Select Graphic Rendition) sequences ending in 'm'
	if terminator != 'm' {
		return style
	}

	// Reset
	if inner == "" || inner == "0" {
		return lipgloss.NewStyle()
	}

	// Parse semicolon-separated parameters
	params := strings.Split(inner, ";")
	i := 0
	for i < len(params) {
		p := params[i]
		switch p {
		case "0":
			style = lipgloss.NewStyle()
		case "1":
			style = style.Bold(true)
		case "2":
			style = style.Faint(true)
		case "3":
			style = style.Italic(true)
		case "4":
			style = style.Underline(true)
		case "5":
			style = style.Blink(true)
		case "7":
			style = style.Reverse(true)
		case "22":
			style = style.Bold(false).Faint(false)
		case "23":
			style = style.Italic(false)
		case "24":
			style = style.Underline(false)
		case "27":
			style = style.Reverse(false)
		case "38": // Foreground color
			if i+1 < len(params) {
				if params[i+1] == "5" && i+2 < len(params) {
					// 256 color: \x1b[38;5;COLORm
					style = style.Foreground(lipgloss.Color(params[i+2]))
					i += 2
				} else if params[i+1] == "2" && i+4 < len(params) {
					// RGB: \x1b[38;2;R;G;Bm
					style = style.Foreground(lipgloss.Color("#" + rgbToHex(params[i+2], params[i+3], params[i+4])))
					i += 4
				}
			}
		case "48": // Background color
			if i+1 < len(params) {
				if params[i+1] == "5" && i+2 < len(params) {
					style = style.Background(lipgloss.Color(params[i+2]))
					i += 2
				} else if params[i+1] == "2" && i+4 < len(params) {
					style = style.Background(lipgloss.Color("#" + rgbToHex(params[i+2], params[i+3], params[i+4])))
					i += 4
				}
			}
		case "39":
			// Default foreground - clear foreground
			style = style.UnsetForeground()
		case "49":
			// Default background - clear background
			style = style.UnsetBackground()
		default:
			// Basic colors 30-37 (foreground) and 40-47 (background)
			if len(p) > 0 {
				code := 0
				for _, c := range p {
					if c >= '0' && c <= '9' {
						code = code*10 + int(c-'0')
					}
				}
				if code >= 30 && code <= 37 {
					style = style.Foreground(lipgloss.Color(p))
				} else if code >= 40 && code <= 47 {
					style = style.Background(lipgloss.Color(p))
				} else if code >= 90 && code <= 97 {
					// Bright foreground
					style = style.Foreground(lipgloss.Color(p))
				} else if code >= 100 && code <= 107 {
					// Bright background
					style = style.Background(lipgloss.Color(p))
				}
			}
		}
		i++
	}

	return style
}

// rgbToHex converts r,g,b strings to a hex color string
func rgbToHex(r, g, b string) string {
	ri, _ := strconv.Atoi(r)
	gi, _ := strconv.Atoi(g)
	bi, _ := strconv.Atoi(b)
	return fmt.Sprintf("%02x%02x%02x", ri, gi, bi)
}

package teapot

import (
	"testing"

	"github.com/radiospiel/critic/simple-go/assert"
	"github.com/charmbracelet/lipgloss"
)

func TestParseANSILine_BasicColors(t *testing.T) {
	// Test basic foreground colors (30-37 should map to palette 0-7)
	tests := []struct {
		name     string
		input    string
		expected lipgloss.Color
		isFg     bool // true for foreground, false for background
	}{
		// Basic foreground colors (30-37)
		{"fg black", "\x1b[30mX", lipgloss.Color("0"), true},
		{"fg red", "\x1b[31mX", lipgloss.Color("1"), true},
		{"fg green", "\x1b[32mX", lipgloss.Color("2"), true},
		{"fg yellow", "\x1b[33mX", lipgloss.Color("3"), true},
		{"fg blue", "\x1b[34mX", lipgloss.Color("4"), true},
		{"fg magenta", "\x1b[35mX", lipgloss.Color("5"), true},
		{"fg cyan", "\x1b[36mX", lipgloss.Color("6"), true},
		{"fg white", "\x1b[37mX", lipgloss.Color("7"), true},

		// Basic background colors (40-47)
		{"bg black", "\x1b[40mX", lipgloss.Color("0"), false},
		{"bg red", "\x1b[41mX", lipgloss.Color("1"), false},
		{"bg green", "\x1b[42mX", lipgloss.Color("2"), false},
		{"bg yellow", "\x1b[43mX", lipgloss.Color("3"), false},
		{"bg blue", "\x1b[44mX", lipgloss.Color("4"), false},
		{"bg magenta", "\x1b[45mX", lipgloss.Color("5"), false},
		{"bg cyan", "\x1b[46mX", lipgloss.Color("6"), false},
		{"bg white", "\x1b[47mX", lipgloss.Color("7"), false},

		// Bright foreground colors (90-97)
		{"bright fg black", "\x1b[90mX", lipgloss.Color("8"), true},
		{"bright fg red", "\x1b[91mX", lipgloss.Color("9"), true},
		{"bright fg green", "\x1b[92mX", lipgloss.Color("10"), true},
		{"bright fg white", "\x1b[97mX", lipgloss.Color("15"), true},

		// Bright background colors (100-107)
		{"bright bg black", "\x1b[100mX", lipgloss.Color("8"), false},
		{"bright bg red", "\x1b[101mX", lipgloss.Color("9"), false},
		{"bright bg green", "\x1b[102mX", lipgloss.Color("10"), false},
		{"bright bg white", "\x1b[107mX", lipgloss.Color("15"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cells := ParseANSILine(tt.input)
			assert.Equals(t, 1, len(cells), "expected 1 cell")

			// Get the style's rendered output to check the color
			style := cells[0].Style
			rendered := style.Render("")

			// Create a reference style with the expected color
			var refStyle lipgloss.Style
			if tt.isFg {
				refStyle = lipgloss.NewStyle().Foreground(tt.expected)
			} else {
				refStyle = lipgloss.NewStyle().Background(tt.expected)
			}
			refRendered := refStyle.Render("")

			assert.Equals(t, refRendered, rendered, "color mismatch for %s", tt.name)
		})
	}
}

func TestParseANSILine_256Colors(t *testing.T) {
	// Test 256-color mode (38;5;N and 48;5;N)
	tests := []struct {
		name     string
		input    string
		expected lipgloss.Color
		isFg     bool
	}{
		{"256 fg black", "\x1b[38;5;0mX", lipgloss.Color("0"), true},
		{"256 fg red", "\x1b[38;5;1mX", lipgloss.Color("1"), true},
		{"256 fg color 42", "\x1b[38;5;42mX", lipgloss.Color("42"), true},
		{"256 fg color 255", "\x1b[38;5;255mX", lipgloss.Color("255"), true},
		{"256 bg black", "\x1b[48;5;0mX", lipgloss.Color("0"), false},
		{"256 bg red", "\x1b[48;5;1mX", lipgloss.Color("1"), false},
		{"256 bg color 42", "\x1b[48;5;42mX", lipgloss.Color("42"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cells := ParseANSILine(tt.input)
			assert.Equals(t, 1, len(cells), "expected 1 cell")

			style := cells[0].Style
			rendered := style.Render("")

			var refStyle lipgloss.Style
			if tt.isFg {
				refStyle = lipgloss.NewStyle().Foreground(tt.expected)
			} else {
				refStyle = lipgloss.NewStyle().Background(tt.expected)
			}
			refRendered := refStyle.Render("")

			assert.Equals(t, refRendered, rendered, "color mismatch for %s", tt.name)
		})
	}
}

func TestParseANSILine_BackgroundBlackNotGreen(t *testing.T) {
	// This is a regression test for the bug where ANSI code 40 (black background)
	// was incorrectly interpreted as palette color 40 (green) instead of palette color 0 (black)
	input := "\x1b[40mtext"
	cells := ParseANSILine(input)

	assert.Equals(t, 4, len(cells), "expected 4 cells for 'text'")

	// All cells should have black background (palette 0), not green (palette 40)
	refStyle := lipgloss.NewStyle().Background(lipgloss.Color("0"))
	refRendered := refStyle.Render("")

	for i, cell := range cells {
		rendered := cell.Style.Render("")
		assert.Equals(t, refRendered, rendered, "cell %d should have black background, not green", i)
	}
}

func TestParseANSILine_Reset(t *testing.T) {
	// Test that reset (\x1b[0m) clears styles
	input := "\x1b[31mR\x1b[0mN"
	cells := ParseANSILine(input)

	assert.Equals(t, 2, len(cells), "expected 2 cells")

	// First cell should have red foreground
	redStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	assert.Equals(t, redStyle.Render(""), cells[0].Style.Render(""), "first cell should be red")

	// Second cell should have no style (reset)
	noStyle := lipgloss.NewStyle()
	assert.Equals(t, noStyle.Render(""), cells[1].Style.Render(""), "second cell should have no style")
}

func TestParseANSILine_RoundTrip(t *testing.T) {
	// Test that colors survive the round-trip: style → render → parse → re-render
	// This verifies that regardless of terminal color settings, the parsed style
	// produces the same output as the original style.
	tests := []struct {
		name  string
		style lipgloss.Style
	}{
		{"black bg", lipgloss.NewStyle().Background(lipgloss.Color("0"))},
		{"red bg", lipgloss.NewStyle().Background(lipgloss.Color("1"))},
		{"green bg", lipgloss.NewStyle().Background(lipgloss.Color("2"))},
		{"white fg", lipgloss.NewStyle().Foreground(lipgloss.Color("7"))},
		{"bright red fg", lipgloss.NewStyle().Foreground(lipgloss.Color("9"))},
		{"256 color 42 bg", lipgloss.NewStyle().Background(lipgloss.Color("42"))},
		{"combined fg+bg", lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Background(lipgloss.Color("0"))},
		{"bold + color", lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("1"))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Render with original style
			original := tt.style.Render("X")

			// Parse the rendered output
			cells := ParseANSILine(original)
			if len(cells) == 0 {
				t.Fatal("expected at least 1 cell")
			}

			// Re-render using the parsed style
			reRendered := cells[0].Style.Render("X")

			// The round-trip should produce identical output
			assert.Equals(t, original, reRendered, "round-trip should preserve style")
		})
	}
}

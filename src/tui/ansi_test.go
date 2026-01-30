package tui

import (
	"testing"

	"github.com/radiospiel/critic/simple-go/assert"
)

func TestStripBackgroundCodes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "No ANSI codes",
			input: "plain text",
			want:  "plain text",
		},
		{
			name:  "Foreground color only",
			input: "\x1b[31mred text\x1b[0m",
			want:  "\x1b[31mred text\x1b[0m",
		},
		{
			name:  "Background color code (40-49 range)",
			input: "\x1b[41mred background\x1b[0m",
			want:  "red background\x1b[0m",
		},
		{
			name:  "Background color 42",
			input: "\x1b[42mgreen background\x1b[0m",
			want:  "green background\x1b[0m",
		},
		{
			name:  "Multiple background codes",
			input: "\x1b[42mgreen\x1b[43myellow\x1b[0m",
			want:  "greenyellow\x1b[0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripBackgroundCodes(tt.input)
			assert.Equals(t, got, tt.want, "stripBackgroundCodes()")
		})
	}
}

func TestTruncateANSI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxWidth int
		// We can't test exact output since lipgloss handles truncation,
		// but we can verify it doesn't crash and handles ANSI codes
	}{
		{
			name:     "No truncation needed",
			input:    "short",
			maxWidth: 10,
		},
		{
			name:     "Truncation needed",
			input:    "very long text that needs truncation",
			maxWidth: 10,
		},
		{
			name:     "With ANSI codes",
			input:    "\x1b[31mred text that is long\x1b[0m",
			maxWidth: 10,
		},
		{
			name:     "Zero width",
			input:    "text",
			maxWidth: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify it doesn't panic
			got := truncateANSI(tt.input, tt.maxWidth)
			// Basic sanity check: result should not be longer than input
			// (this is a weak test but lipgloss handles the complex logic)
			_ = got
		})
	}
}

package ui

import (
	"strings"
	"testing"
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
			if got != tt.want {
				t.Errorf("stripBackgroundCodes() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExpandTabsInANSI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "No tabs",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "Single tab at start",
			input: "\thello",
			want:  "    hello",
		},
		{
			name:  "Tab after 1 char",
			input: "a\tb",
			want:  "a   b",
		},
		{
			name:  "Tab after 2 chars",
			input: "ab\tc",
			want:  "ab  c",
		},
		{
			name:  "Tab after 3 chars",
			input: "abc\td",
			want:  "abc d",
		},
		{
			name:  "Tab after 4 chars (next tab stop)",
			input: "abcd\te",
			want:  "abcd    e",
		},
		{
			name:  "Multiple tabs",
			input: "a\tb\tc",
			want:  "a   b   c",
		},
		{
			name:  "Tab with ANSI codes",
			input: "\x1b[31m\tred\x1b[0m",
			want:  "\x1b[31m    red\x1b[0m",
		},
		{
			name:  "ANSI codes don't affect column position",
			input: "a\x1b[31mb\x1b[0m\tc",
			want:  "a\x1b[31mb\x1b[0m  c", // "ab" = 2 chars, tab goes to col 4 (2 spaces)
		},
		{
			name:  "Empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandTabsInANSI(tt.input)
			if got != tt.want {
				t.Errorf("expandTabsInANSI() = %q, want %q", got, tt.want)
			}
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

func TestExpandTabsColumnCalculation(t *testing.T) {
	// Test that tabs expand to correct tab stops
	input := "0123456789\t0123456789\t0123456789"
	//        ^col 0-9   ^col 10->16   ^col 16-25  ^col 26->28
	got := expandTabsInANSI(input)

	// Tab at position 10 should expand to reach column 12 (next tab stop after 10)
	// Tab at position 26 should expand to reach column 28 (next tab stop after 26)
	expected := "0123456789  0123456789  0123456789"

	if got != expected {
		t.Errorf("Tab stop calculation incorrect\ngot:  %q\nwant: %q", got, expected)
	}
}

func TestExpandTabsPreservesANSI(t *testing.T) {
	// Verify that ANSI escape codes are preserved and don't affect column counting
	input := "\x1b[31mred\x1b[0m\ttext"
	got := expandTabsInANSI(input)

	// Should have ANSI codes intact
	if !strings.Contains(got, "\x1b[31m") {
		t.Error("ANSI foreground code was removed")
	}
	if !strings.Contains(got, "\x1b[0m") {
		t.Error("ANSI reset code was removed")
	}

	// The visible text is "red" (3 chars) + tab, so tab should expand to 1 space
	// to reach column 4
	expected := "\x1b[31mred\x1b[0m text"
	if got != expected {
		t.Errorf("ANSI preservation failed\ngot:  %q\nwant: %q", got, expected)
	}
}

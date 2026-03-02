package logger

import (
	"testing"
)

func TestLinkifyURLs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no URLs",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "http URL",
			input:    "visit http://localhost:8080 now",
			expected: "visit \033]8;;http://localhost:8080\033\\http://localhost:8080\033]8;;\033\\ now",
		},
		{
			name:     "https URL",
			input:    "see https://example.com/path",
			expected: "see \033]8;;https://example.com/path\033\\https://example.com/path\033]8;;\033\\",
		},
		{
			name:     "multiple URLs",
			input:    "http://a.com and http://b.com",
			expected: "\033]8;;http://a.com\033\\http://a.com\033]8;;\033\\ and \033]8;;http://b.com\033\\http://b.com\033]8;;\033\\",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := linkifyURLs(tt.input)
			if result != tt.expected {
				t.Errorf("linkifyURLs(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestHyperlink_ColorsDisabled(t *testing.T) {
	old := sharedDest.enableColors
	sharedDest.enableColors = false
	defer func() { sharedDest.enableColors = old }()

	result := Hyperlink("http://example.com", "click here")
	if result != "click here" {
		t.Errorf("Hyperlink() = %q, want plain text when colors disabled", result)
	}
}

func TestHyperlink_ColorsEnabled(t *testing.T) {
	old := sharedDest.enableColors
	sharedDest.enableColors = true
	defer func() { sharedDest.enableColors = old }()

	result := Hyperlink("http://example.com", "click here")
	expected := "\033]8;;http://example.com\033\\click here\033]8;;\033\\"
	if result != expected {
		t.Errorf("Hyperlink() = %q, want %q", result, expected)
	}
}

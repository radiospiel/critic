package highlight

import (
	"strings"
	"testing"
)

func TestNewHighlighter(t *testing.T) {
	h := NewHighlighter()
	if h == nil {
		t.Fatal("NewHighlighter() returned nil")
	}

	if h.formatter == nil {
		t.Error("Highlighter formatter is nil")
	}

	if h.style == nil {
		t.Error("Highlighter style is nil")
	}
}

func TestHighlight_Go(t *testing.T) {
	h := NewHighlighter()
	code := `package main

import "fmt"

func main() {
	fmt.Println("hello")
}`

	result, err := h.Highlight(code, "test.go")
	if err != nil {
		t.Fatalf("Highlight() error = %v", err)
	}

	// Result should contain ANSI codes (indicated by ESC character)
	if !strings.Contains(result, "\x1b[") {
		t.Error("Highlight() should add ANSI color codes")
	}

	// Result should still contain the original text
	if !strings.Contains(result, "package") || !strings.Contains(result, "main") {
		t.Error("Highlight() should preserve original text")
	}
}

func TestHighlight_JavaScript(t *testing.T) {
	h := NewHighlighter()
	code := `function hello() {
	console.log("world");
}`

	result, err := h.Highlight(code, "test.js")
	if err != nil {
		t.Fatalf("Highlight() error = %v", err)
	}

	if !strings.Contains(result, "\x1b[") {
		t.Error("Highlight() should add ANSI color codes for JavaScript")
	}

	if !strings.Contains(result, "function") || !strings.Contains(result, "console") {
		t.Error("Highlight() should preserve JavaScript code")
	}
}

func TestHighlight_Python(t *testing.T) {
	h := NewHighlighter()
	code := `def hello():
    print("world")`

	result, err := h.Highlight(code, "test.py")
	if err != nil {
		t.Fatalf("Highlight() error = %v", err)
	}

	if !strings.Contains(result, "\x1b[") {
		t.Error("Highlight() should add ANSI color codes for Python")
	}

	if !strings.Contains(result, "def") || !strings.Contains(result, "print") {
		t.Error("Highlight() should preserve Python code")
	}
}

func TestHighlight_UnknownExtension(t *testing.T) {
	h := NewHighlighter()
	code := "some plain text without syntax"

	result, err := h.Highlight(code, "test.unknown")
	if err != nil {
		t.Fatalf("Highlight() error = %v", err)
	}

	// Should return code as-is or with minimal highlighting
	if !strings.Contains(result, "plain text") {
		t.Error("Highlight() should preserve text for unknown extensions")
	}
}

func TestHighlight_EmptyCode(t *testing.T) {
	h := NewHighlighter()

	result, err := h.Highlight("", "test.go")
	if err != nil {
		t.Fatalf("Highlight() error = %v", err)
	}

	// Empty code should return empty or minimal result
	if len(result) > 10 {
		t.Errorf("Highlight() for empty code returned %d chars, expected minimal output", len(result))
	}
}

func TestHighlightLine(t *testing.T) {
	h := NewHighlighter()
	line := `import "fmt"`

	result := h.HighlightLine(line, "test.go")

	// Should have ANSI codes
	if !strings.Contains(result, "\x1b[") {
		t.Error("HighlightLine() should add ANSI color codes")
	}

	// Should not have trailing newline
	if strings.HasSuffix(result, "\n") {
		t.Error("HighlightLine() should not have trailing newline")
	}

	// Should preserve content
	if !strings.Contains(result, "import") {
		t.Error("HighlightLine() should preserve content")
	}
}

func TestHighlightLines_Multiple(t *testing.T) {
	h := NewHighlighter()
	lines := []string{
		"package main",
		"",
		"import \"fmt\"",
		"",
		"func main() {",
		"}",
	}

	result := h.HighlightLines(lines, "test.go")

	// Should return same number of lines
	if len(result) != len(lines) {
		t.Fatalf("HighlightLines() returned %d lines, want %d", len(result), len(lines))
	}

	// Verify we got results (some might be empty lines which is OK)
	nonEmptyCount := 0
	for _, line := range result {
		if line != "" {
			nonEmptyCount++
		}
	}
	if nonEmptyCount == 0 {
		t.Error("HighlightLines() should return some non-empty results")
	}
}

func TestHighlightLines_Empty(t *testing.T) {
	h := NewHighlighter()
	lines := []string{}

	result := h.HighlightLines(lines, "test.go")

	if len(result) != 0 {
		t.Errorf("HighlightLines() for empty input returned %d lines, want 0", len(result))
	}
}

func TestHighlightLines_SingleLine(t *testing.T) {
	h := NewHighlighter()
	lines := []string{"package main"}

	result := h.HighlightLines(lines, "test.go")

	if len(result) != 1 {
		t.Fatalf("HighlightLines() returned %d lines, want 1", len(result))
	}

	if !strings.Contains(result[0], "package") {
		t.Error("HighlightLines() should preserve content")
	}
}

func TestGetLexer(t *testing.T) {
	h := NewHighlighter()

	tests := []struct {
		filename string
		wantNil  bool
	}{
		{"test.go", false},
		{"test.js", false},
		{"test.py", false},
		{"test.rb", false},
		{"test.java", false},
		{"test.c", false},
		{"test.cpp", false},
		{"test.rs", false},
		{"test.ts", false},
		{"test.sh", false},
		{"Makefile", false},
		{"test.unknown", false}, // Should fallback to plaintext
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			lexer := h.getLexer(tt.filename)
			if tt.wantNil && lexer != nil {
				t.Errorf("getLexer(%q) = %v, want nil", tt.filename, lexer)
			}
			if !tt.wantNil && lexer == nil {
				t.Errorf("getLexer(%q) = nil, want non-nil", tt.filename)
			}
		})
	}
}

func TestGetLanguage(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"test.go", "go"},
		{"test.js", "javascript"},
		{"test.py", "python"},
		{"test.rb", "ruby"},
		{"test.java", "java"},
		{"test.unknown", "text"},
		{"no-extension", "text"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := GetLanguage(tt.filename)
			// Language names may vary slightly, so just check it's not empty for known types
			if tt.want != "text" && got == "" {
				t.Errorf("GetLanguage(%q) = %q, want non-empty", tt.filename, got)
			}
			if tt.want == "text" && got != "text" {
				t.Errorf("GetLanguage(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}

func TestHighlight_PreservesStructure(t *testing.T) {
	h := NewHighlighter()
	code := `line1
line2
line3`

	result, err := h.Highlight(code, "test.txt")
	if err != nil {
		t.Fatalf("Highlight() error = %v", err)
	}

	// Should preserve all three lines
	lines := strings.Split(strings.TrimSpace(result), "\n")
	if len(lines) != 3 {
		t.Errorf("Highlight() returned %d lines, want 3", len(lines))
	}
}

func TestHighlightLines_Performance(t *testing.T) {
	// Test that batch highlighting works for many lines
	h := NewHighlighter()

	lines := make([]string, 1000)
	for i := range lines {
		lines[i] = `fmt.Println("test")`
	}

	result := h.HighlightLines(lines, "test.go")

	if len(result) != len(lines) {
		t.Errorf("HighlightLines() for %d lines returned %d lines", len(lines), len(result))
	}
}

func TestHighlightLines_ErrorRecovery(t *testing.T) {
	// Test that HighlightLines returns originals on error
	// This is hard to trigger since Highlight is quite robust,
	// but we test the line count validation
	h := NewHighlighter()

	lines := []string{"line1", "line2", "line3"}
	result := h.HighlightLines(lines, "test.go")

	// Should always return same number of lines
	if len(result) != len(lines) {
		t.Errorf("HighlightLines() should return same line count on any condition")
	}
}

func TestCustomStyle(t *testing.T) {
	// Verify that custom style is registered and used
	if customStyle == nil {
		t.Fatal("customStyle should be initialized")
	}

	h := NewHighlighter()
	if h.style != customStyle {
		t.Error("Highlighter should use customStyle")
	}
}

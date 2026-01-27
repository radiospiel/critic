package highlight

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.org/radiospiel/critic/simple-go/assert"
)

func TestNewHighlighter(t *testing.T) {
	h := NewHighlighter()
	assert.NotNil(t, h, "NewHighlighter() returned nil")
	assert.NotNil(t, h.formatter, "Highlighter formatter is nil")
}

func TestHighlight_Go(t *testing.T) {
	h := NewHighlighter()
	code := `package main

import "fmt"

func main() {
	fmt.Println("hello")
}`

	result, err := h.Highlight(code, "test.go")
	assert.NoError(t, err, "Highlight()")

	// Result should contain ANSI codes (indicated by ESC character)
	assert.True(t, strings.Contains(result, "\x1b["), "Highlight() should add ANSI color codes")

	// Result should still contain the original text
	assert.True(t, strings.Contains(result, "package") && strings.Contains(result, "main"), "Highlight() should preserve original text")
}

func TestHighlight_JavaScript(t *testing.T) {
	h := NewHighlighter()
	code := `function hello() {
	console.log("world");
}`

	result, err := h.Highlight(code, "test.js")
	assert.NoError(t, err, "Highlight()")

	assert.True(t, strings.Contains(result, "\x1b["), "Highlight() should add ANSI color codes for JavaScript")
	assert.True(t, strings.Contains(result, "function") && strings.Contains(result, "console"), "Highlight() should preserve JavaScript code")
}

func TestHighlight_Python(t *testing.T) {
	h := NewHighlighter()
	code := `def hello():
    print("world")`

	result, err := h.Highlight(code, "test.py")
	assert.NoError(t, err, "Highlight()")

	assert.True(t, strings.Contains(result, "\x1b["), "Highlight() should add ANSI color codes for Python")
	assert.True(t, strings.Contains(result, "def") && strings.Contains(result, "print"), "Highlight() should preserve Python code")
}

func TestHighlight_UnknownExtension(t *testing.T) {
	h := NewHighlighter()
	code := "some plain text without syntax"

	result, err := h.Highlight(code, "test.unknown")
	assert.NoError(t, err, "Highlight()")

	// Should return code as-is or with minimal highlighting
	assert.True(t, strings.Contains(result, "plain text"), "Highlight() should preserve text for unknown extensions")
}

func TestHighlight_EmptyCode(t *testing.T) {
	h := NewHighlighter()

	result, err := h.Highlight("", "test.go")
	assert.NoError(t, err, "Highlight()")

	// Empty code should return empty or minimal result
	assert.True(t, len(result) <= 10, "Highlight() for empty code returned %d chars, expected minimal output", len(result))
}

func TestHighlightLine(t *testing.T) {
	h := NewHighlighter()
	line := `import "fmt"`

	result := h.HighlightLine(line, "test.go")

	// Should have ANSI codes
	assert.True(t, strings.Contains(result, "\x1b["), "HighlightLine() should add ANSI color codes")

	// Should not have trailing newline
	assert.False(t, strings.HasSuffix(result, "\n"), "HighlightLine() should not have trailing newline")

	// Should preserve content
	assert.True(t, strings.Contains(result, "import"), "HighlightLine() should preserve content")
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
	assert.Equals(t, len(result), len(lines), "HighlightLines() returned %d lines, want %d", len(result), len(lines))

	// Verify we got results (some might be empty lines which is OK)
	nonEmptyCount := 0
	for _, line := range result {
		if line != "" {
			nonEmptyCount++
		}
	}
	assert.True(t, nonEmptyCount > 0, "HighlightLines() should return some non-empty results")
}

func TestHighlightLines_Empty(t *testing.T) {
	h := NewHighlighter()
	lines := []string{}

	result := h.HighlightLines(lines, "test.go")

	assert.Equals(t, len(result), 0, "HighlightLines() for empty input")
}

func TestHighlightLines_SingleLine(t *testing.T) {
	h := NewHighlighter()
	lines := []string{"package main"}

	result := h.HighlightLines(lines, "test.go")

	assert.Equals(t, len(result), 1, "HighlightLines() returned %d lines, want 1", len(result))
	assert.True(t, strings.Contains(result[0], "package"), "HighlightLines() should preserve content")
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
			if tt.wantNil {
				assert.Nil(t, lexer, "getLexer(%q) should be nil", tt.filename)
			} else {
				assert.NotNil(t, lexer, "getLexer(%q) should not be nil", tt.filename)
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
			if tt.want != "text" {
				assert.NotEquals(t, got, "", "GetLanguage(%q) should be non-empty", tt.filename)
			} else {
				assert.Equals(t, got, "text", "GetLanguage(%q)", tt.filename)
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
	assert.NoError(t, err, "Highlight()")

	// Should preserve all three lines
	lines := strings.Split(strings.TrimSpace(result), "\n")
	assert.Equals(t, len(lines), 3, "Highlight() should return 3 lines")
}

func TestHighlightLines_Performance(t *testing.T) {
	// Test that batch highlighting works for many lines
	h := NewHighlighter()

	lines := make([]string, 1000)
	for i := range lines {
		lines[i] = `fmt.Println("test")`
	}

	result := h.HighlightLines(lines, "test.go")

	assert.Equals(t, len(result), len(lines), "HighlightLines() for %d lines", len(lines))
}

func TestHighlightLines_ErrorRecovery(t *testing.T) {
	// Test that HighlightLines returns originals on error
	// This is hard to trigger since Highlight is quite robust,
	// but we test the line count validation
	h := NewHighlighter()

	lines := []string{"line1", "line2", "line3"}
	result := h.HighlightLines(lines, "test.go")

	// Should always return same number of lines
	assert.Equals(t, len(result), len(lines), "HighlightLines() should return same line count on any condition")
}

func TestCustomStyle(t *testing.T) {
	// Verify that the highlighter is initialized with a formatter
	h := NewHighlighter()
	assert.NotNil(t, h.formatter, "Highlighter should have a formatter initialized")
}

func TestTabWidth(t *testing.T) {
	tests := []struct {
		language string
		want     int
	}{
		{"go", 4},
		{"Go", 4},
		{"golang", 4},
		{"ruby", 2},
		{"Ruby", 2},
		{"rb", 2},
		{"python", 4},
		{"javascript", 4},
		{"unknown", 4},
		{"", 4},
	}

	for _, tt := range tests {
		t.Run(tt.language, func(t *testing.T) {
			got := TabWidth(tt.language)
			assert.Equals(t, got, tt.want, "TabWidth(%q)", tt.language)
		})
	}
}

func TestExpandTabs(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		tabWidth int
		want     string
	}{
		{
			name:     "No tabs",
			code:     "hello world",
			tabWidth: 4,
			want:     "hello world",
		},
		{
			name:     "Single tab at start (4 spaces)",
			code:     "\thello",
			tabWidth: 4,
			want:     "    hello",
		},
		{
			name:     "Single tab at start (2 spaces)",
			code:     "\thello",
			tabWidth: 2,
			want:     "  hello",
		},
		{
			name:     "Tab after 1 char (4-space tabs)",
			code:     "a\tb",
			tabWidth: 4,
			want:     "a   b",
		},
		{
			name:     "Tab after 1 char (2-space tabs)",
			code:     "a\tb",
			tabWidth: 2,
			want:     "a b",
		},
		{
			name:     "Multiple tabs (4-space)",
			code:     "\t\thello",
			tabWidth: 4,
			want:     "        hello",
		},
		{
			name:     "Multiple tabs (2-space)",
			code:     "\t\thello",
			tabWidth: 2,
			want:     "    hello",
		},
		{
			name:     "Tab with newline reset",
			code:     "\thello\n\tworld",
			tabWidth: 4,
			want:     "    hello\n    world",
		},
		{
			name:     "Mixed tabs and spaces",
			code:     " \thello",
			tabWidth: 4,
			want:     "    hello",
		},
		{
			name:     "Empty string",
			code:     "",
			tabWidth: 4,
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandTabs(tt.code, tt.tabWidth)
			assert.Equals(t, got, tt.want, "expandTabs()")
		})
	}
}

func TestHighlight_ExpandsTabsBeforeHighlighting(t *testing.T) {
	h := NewHighlighter()

	// Go code with tabs (should expand to 4 spaces)
	goCode := "package main\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}"
	result, err := h.Highlight(goCode, "test.go")
	assert.NoError(t, err, "Highlight()")

	// Result should not contain tab characters
	assert.False(t, strings.Contains(result, "\t"), "Highlight() result should not contain tab characters")

	// Verify content is present (check separately due to ANSI codes)
	assert.True(t, strings.Contains(result, "fmt"), "Highlight() should preserve 'fmt'")
	assert.True(t, strings.Contains(result, "Println"), "Highlight() should preserve 'Println'")
}

func TestHighlight_WithEmoji(t *testing.T) {
	h := NewHighlighter()
	code := "// Comment with emoji: 🎨 🚀 ✨\npackage main"

	result, err := h.Highlight(code, "test.go")
	assert.NoError(t, err, "Highlight()")

	// Emoji should be preserved
	assert.True(t, strings.Contains(result, "🎨") && strings.Contains(result, "🚀") && strings.Contains(result, "✨"), "Highlight() should preserve emoji characters")
}

func TestHighlight_WithUmlautsAndSpecialChars(t *testing.T) {
	h := NewHighlighter()
	code := "# Kommentar mit Umlauten: äöü ÄÖÜ ß\nclass Grüße"

	result, err := h.Highlight(code, "test.rb")
	assert.NoError(t, err, "Highlight()")

	// German umlauts and special characters should be preserved
	specialChars := []string{"ä", "ö", "ü", "Ä", "Ö", "Ü", "ß", "ü"}
	for _, char := range specialChars {
		assert.True(t, strings.Contains(result, char), "Highlight() should preserve character %q", char)
	}
}

func TestHighlight_WithNonASCII(t *testing.T) {
	h := NewHighlighter()
	tests := []struct {
		name     string
		code     string
		filename string
		chars    []string
	}{
		{
			name:     "Japanese characters",
			code:     "# こんにちは世界\nclass Hello",
			filename: "test.rb",
			chars:    []string{"こ", "ん", "に", "ち", "は", "世", "界"},
		},
		{
			name:     "Chinese characters",
			code:     "# 你好世界\nclass Hello",
			filename: "test.rb",
			chars:    []string{"你", "好", "世", "界"},
		},
		{
			name:     "Cyrillic characters",
			code:     "# Привет мир\nclass Hello",
			filename: "test.rb",
			chars:    []string{"П", "р", "и", "в", "е", "т"},
		},
		{
			name:     "Mixed emoji and text",
			code:     "// TODO: Add feature 📝\n// BUG: Fix issue 🐛\npackage main",
			filename: "test.go",
			chars:    []string{"📝", "🐛"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := h.Highlight(tt.code, tt.filename)
			assert.NoError(t, err, "Highlight()")

			for _, char := range tt.chars {
				assert.True(t, strings.Contains(result, char), "Highlight() should preserve character %q", char)
			}
		})
	}
}

func TestHighlight_RealWorldGoFile(t *testing.T) {
	h := NewHighlighter()

	// Read the real Go test file with tabs
	content, err := os.ReadFile(filepath.Join("testdata", "sample.go"))
	assert.NoError(t, err, "Failed to read testdata/sample.go")

	code := string(content)

	// Verify the file actually contains tabs
	if !strings.Contains(code, "\t") {
		t.Skip("Test file does not contain tabs")
	}

	result, err := h.Highlight(code, "sample.go")
	assert.NoError(t, err, "Highlight()")

	// Result should not contain tabs
	assert.False(t, strings.Contains(result, "\t"), "Highlight() result should not contain tab characters")

	// Should preserve special characters from the file
	assert.True(t, strings.Contains(result, "äöü"), "Highlight() should preserve German umlauts from test file")
	assert.True(t, strings.Contains(result, "🎨"), "Highlight() should preserve emoji from test file")

	// Verify essential content is present (check separately due to ANSI codes)
	assert.True(t, strings.Contains(result, "func"), "Highlight() should preserve 'func' keyword")
	assert.True(t, strings.Contains(result, "main"), "Highlight() should preserve 'main' function name")
	assert.True(t, strings.Contains(result, "fmt"), "Highlight() should preserve 'fmt' package")
	assert.True(t, strings.Contains(result, "Println"), "Highlight() should preserve 'Println' method")
}

func TestHighlight_RealWorldRubyFile(t *testing.T) {
	h := NewHighlighter()

	// Read the real Ruby test file with tabs
	content, err := os.ReadFile(filepath.Join("testdata", "sample.rb"))
	assert.NoError(t, err, "Failed to read testdata/sample.rb")

	code := string(content)

	// Verify the file actually contains tabs
	if !strings.Contains(code, "\t") {
		t.Skip("Test file does not contain tabs")
	}

	result, err := h.Highlight(code, "sample.rb")
	assert.NoError(t, err, "Highlight()")

	// Result should not contain tabs
	assert.False(t, strings.Contains(result, "\t"), "Highlight() result should not contain tab characters")

	// Should preserve special characters from the file
	specialChars := []string{"äöü", "ß", "🚀"}
	for _, char := range specialChars {
		assert.True(t, strings.Contains(result, char), "Highlight() should preserve %q from test file", char)
	}

	// Verify essential content is present (check separately due to ANSI codes)
	assert.True(t, strings.Contains(result, "class"), "Highlight() should preserve 'class' keyword")
	assert.True(t, strings.Contains(result, "Greeter"), "Highlight() should preserve 'Greeter' class name")
	assert.True(t, strings.Contains(result, "def"), "Highlight() should preserve 'def' keyword")
	assert.True(t, strings.Contains(result, "initialize"), "Highlight() should preserve 'initialize' method")
}

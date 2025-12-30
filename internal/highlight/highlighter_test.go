package highlight

import (
	"os"
	"path/filepath"
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
	// Verify that the highlighter is initialized with a formatter
	h := NewHighlighter()
	if h.formatter == nil {
		t.Error("Highlighter should have a formatter initialized")
	}
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
			if got != tt.want {
				t.Errorf("TabWidth(%q) = %d, want %d", tt.language, got, tt.want)
			}
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
			if got != tt.want {
				t.Errorf("expandTabs() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHighlight_ExpandsTabsBeforeHighlighting(t *testing.T) {
	h := NewHighlighter()

	// Go code with tabs (should expand to 4 spaces)
	goCode := "package main\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}"
	result, err := h.Highlight(goCode, "test.go")
	if err != nil {
		t.Fatalf("Highlight() error = %v", err)
	}

	// Result should not contain tab characters
	if strings.Contains(result, "\t") {
		t.Error("Highlight() result contains tab characters, should be expanded to spaces")
	}

	// Verify content is present (check separately due to ANSI codes)
	if !strings.Contains(result, "fmt") {
		t.Error("Highlight() should preserve 'fmt'")
	}
	if !strings.Contains(result, "Println") {
		t.Error("Highlight() should preserve 'Println'")
	}
}

func TestHighlight_WithEmoji(t *testing.T) {
	h := NewHighlighter()
	code := "// Comment with emoji: 🎨 🚀 ✨\npackage main"

	result, err := h.Highlight(code, "test.go")
	if err != nil {
		t.Fatalf("Highlight() error = %v", err)
	}

	// Emoji should be preserved
	if !strings.Contains(result, "🎨") || !strings.Contains(result, "🚀") || !strings.Contains(result, "✨") {
		t.Error("Highlight() should preserve emoji characters")
	}
}

func TestHighlight_WithUmlautsAndSpecialChars(t *testing.T) {
	h := NewHighlighter()
	code := "# Kommentar mit Umlauten: äöü ÄÖÜ ß\nclass Grüße"

	result, err := h.Highlight(code, "test.rb")
	if err != nil {
		t.Fatalf("Highlight() error = %v", err)
	}

	// German umlauts and special characters should be preserved
	specialChars := []string{"ä", "ö", "ü", "Ä", "Ö", "Ü", "ß", "ü"}
	for _, char := range specialChars {
		if !strings.Contains(result, char) {
			t.Errorf("Highlight() should preserve character %q", char)
		}
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
			if err != nil {
				t.Fatalf("Highlight() error = %v", err)
			}

			for _, char := range tt.chars {
				if !strings.Contains(result, char) {
					t.Errorf("Highlight() should preserve character %q", char)
				}
			}
		})
	}
}

func TestHighlight_RealWorldGoFile(t *testing.T) {
	h := NewHighlighter()

	// Read the real Go test file with tabs
	content, err := os.ReadFile(filepath.Join("testdata", "sample.go"))
	if err != nil {
		t.Fatalf("Failed to read testdata/sample.go: %v", err)
	}

	code := string(content)

	// Verify the file actually contains tabs
	if !strings.Contains(code, "\t") {
		t.Skip("Test file does not contain tabs")
	}

	result, err := h.Highlight(code, "sample.go")
	if err != nil {
		t.Fatalf("Highlight() error = %v", err)
	}

	// Result should not contain tabs
	if strings.Contains(result, "\t") {
		t.Error("Highlight() result contains tab characters, should be expanded")
	}

	// Should preserve special characters from the file
	if !strings.Contains(result, "äöü") {
		t.Error("Highlight() should preserve German umlauts from test file")
	}
	if !strings.Contains(result, "🎨") {
		t.Error("Highlight() should preserve emoji from test file")
	}

	// Verify essential content is present (check separately due to ANSI codes)
	if !strings.Contains(result, "func") {
		t.Error("Highlight() should preserve 'func' keyword")
	}
	if !strings.Contains(result, "main") {
		t.Error("Highlight() should preserve 'main' function name")
	}
	if !strings.Contains(result, "fmt") {
		t.Error("Highlight() should preserve 'fmt' package")
	}
	if !strings.Contains(result, "Println") {
		t.Error("Highlight() should preserve 'Println' method")
	}
}

func TestHighlight_RealWorldRubyFile(t *testing.T) {
	h := NewHighlighter()

	// Read the real Ruby test file with tabs
	content, err := os.ReadFile(filepath.Join("testdata", "sample.rb"))
	if err != nil {
		t.Fatalf("Failed to read testdata/sample.rb: %v", err)
	}

	code := string(content)

	// Verify the file actually contains tabs
	if !strings.Contains(code, "\t") {
		t.Skip("Test file does not contain tabs")
	}

	result, err := h.Highlight(code, "sample.rb")
	if err != nil {
		t.Fatalf("Highlight() error = %v", err)
	}

	// Result should not contain tabs
	if strings.Contains(result, "\t") {
		t.Error("Highlight() result contains tab characters, should be expanded")
	}

	// Should preserve special characters from the file
	specialChars := []string{"äöü", "ß", "🚀"}
	for _, char := range specialChars {
		if !strings.Contains(result, char) {
			t.Errorf("Highlight() should preserve %q from test file", char)
		}
	}

	// Verify essential content is present (check separately due to ANSI codes)
	if !strings.Contains(result, "class") {
		t.Error("Highlight() should preserve 'class' keyword")
	}
	if !strings.Contains(result, "Greeter") {
		t.Error("Highlight() should preserve 'Greeter' class name")
	}
	if !strings.Contains(result, "def") {
		t.Error("Highlight() should preserve 'def' keyword")
	}
	if !strings.Contains(result, "initialize") {
		t.Error("Highlight() should preserve 'initialize' method")
	}
}

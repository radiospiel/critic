package highlight

import (
	"bytes"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// Base foreground colors
var baseFg = chroma.StyleEntries{
	chroma.Text:                "#ffffff",
	chroma.Keyword:             "#66d9ef",
	chroma.KeywordNamespace:    "#f92672",
	chroma.KeywordType:         "#66d9ef",
	chroma.Name:                "#ffffff",
	chroma.NameClass:           "#a6e22e",
	chroma.NameFunction:        "#a6e22e",
	chroma.NameBuiltin:         "#66d9ef",
	chroma.NameVariable:        "#ffffff",
	chroma.LiteralString:       "#e6db74",
	chroma.LiteralNumber:       "#ae81ff",
	chroma.Comment:             "#99aa88",
	chroma.CommentPreproc:      "#99aa88",
	chroma.Operator:            "#f92672",
	chroma.Punctuation:         "#ffffff",
	chroma.Generic:             "#ffffff",
	chroma.GenericHeading:      "#75715e",
	chroma.GenericSubheading:   "#75715e",
	chroma.GenericDeleted:      "#ffffff",
	chroma.GenericInserted:     "#ffffff",
	chroma.GenericEmph:         "italic",
	chroma.GenericStrong:       "bold",
}

// Helper to add background to all style entries
func withBg(base chroma.StyleEntries, bg string) chroma.StyleEntries {
	result := make(chroma.StyleEntries)
	result[chroma.Background] = "bg:" + bg
	for token, fg := range base {
		result[token] = fg + " bg:" + bg
	}
	return result
}

// Styles with different backgrounds
var styleAdded = styles.Register(chroma.MustNewStyle("critic-added", withBg(baseFg, "#1a3a1a")))
var styleDeleted = styles.Register(chroma.MustNewStyle("critic-deleted", withBg(baseFg, "#3a1a1a")))
var styleContext = styles.Register(chroma.MustNewStyle("critic-context", withBg(baseFg, "#000000")))

// Highlighter provides syntax highlighting for code
type Highlighter struct {
	formatter chroma.Formatter
}

// NewHighlighter creates a new syntax highlighter
func NewHighlighter() *Highlighter {
	// Use terminal256 formatter for better Terminal.app compatibility
	// terminal16m (true color) can have issues with Terminal.app
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Get("terminal16m")
	}

	return &Highlighter{
		formatter: formatter,
	}
}

// HighlightWithStyle highlights code with a specific chroma style
func (h *Highlighter) HighlightWithStyle(code, filename string, style *chroma.Style) (string, error) {
	lexer := h.getLexer(filename)
	if lexer == nil {
		return code, nil
	}

	language := GetLanguage(filename)
	tabWidth := TabWidth(language)
	code = expandTabs(code, tabWidth)

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code, err
	}

	var buf bytes.Buffer
	err = h.formatter.Format(&buf, style, iterator)
	if err != nil {
		return code, err
	}

	return buf.String(), nil
}

// Highlight applies syntax highlighting with context style
func (h *Highlighter) Highlight(code, filename string) (string, error) {
	return h.HighlightWithStyle(code, filename, styleContext)
}

// GetAddedStyle returns the style for added lines
func GetAddedStyle() *chroma.Style {
	return styleAdded
}

// GetDeletedStyle returns the style for deleted lines
func GetDeletedStyle() *chroma.Style {
	return styleDeleted
}

// GetContextStyle returns the style for context lines
func GetContextStyle() *chroma.Style {
	return styleContext
}

// HighlightLine highlights a single line of code
func (h *Highlighter) HighlightLine(line, filename string) string {
	highlighted, err := h.Highlight(line, filename)
	if err != nil {
		return line
	}

	// Remove trailing newline if added by formatter
	return strings.TrimSuffix(highlighted, "\n")
}

// HighlightLines highlights multiple lines at once (more efficient than line-by-line)
// Returns a slice with the same number of elements as input
func (h *Highlighter) HighlightLines(lines []string, filename string) []string {
	if len(lines) == 0 {
		return lines
	}

	// Join all lines into one block
	combined := strings.Join(lines, "\n")

	// Highlight the entire block at once
	highlighted, err := h.Highlight(combined, filename)
	if err != nil {
		// Return originals on error
		return lines
	}

	// Split back into lines
	result := strings.Split(highlighted, "\n")

	// Handle edge case: if formatter adds extra newline at end
	if len(result) > len(lines) && result[len(result)-1] == "" {
		result = result[:len(lines)]
	}

	// If line count mismatch (shouldn't happen), return originals
	if len(result) != len(lines) {
		return lines
	}

	return result
}

// getLexer returns the appropriate lexer for the given filename
func (h *Highlighter) getLexer(filename string) chroma.Lexer {
	// Try to get lexer by filename
	lexer := lexers.Match(filename)
	if lexer != nil {
		return chroma.Coalesce(lexer)
	}

	// Try by extension
	ext := filepath.Ext(filename)
	if ext != "" {
		lexer = lexers.Get(strings.TrimPrefix(ext, "."))
		if lexer != nil {
			return chroma.Coalesce(lexer)
		}
	}

	// Fallback to plaintext
	return lexers.Fallback
}

// GetLanguage returns the detected language name for a filename
func GetLanguage(filename string) string {
	lexer := lexers.Match(filename)
	if lexer == nil {
		ext := filepath.Ext(filename)
		if ext != "" {
			lexer = lexers.Get(strings.TrimPrefix(ext, "."))
		}
	}

	if lexer != nil {
		config := lexer.Config()
		if config != nil && len(config.Aliases) > 0 {
			return config.Aliases[0]
		}
		if config != nil {
			return config.Name
		}
	}

	return "text"
}

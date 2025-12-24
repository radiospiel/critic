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

// Highlighter provides syntax highlighting for code
type Highlighter struct {
	formatter chroma.Formatter
	style     *chroma.Style
}

// NewHighlighter creates a new syntax highlighter
func NewHighlighter() *Highlighter {
	return &Highlighter{
		formatter: formatters.Get("terminal256"),
		style:     styles.Get("monokai"),
	}
}

// Highlight applies syntax highlighting to the given code
func (h *Highlighter) Highlight(code, filename string) (string, error) {
	// Get lexer based on filename
	lexer := h.getLexer(filename)
	if lexer == nil {
		// If no lexer found, return code as-is
		return code, nil
	}

	// Tokenize the code
	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code, err
	}

	// Format with ANSI colors
	var buf bytes.Buffer
	err = h.formatter.Format(&buf, h.style, iterator)
	if err != nil {
		return code, err
	}

	return buf.String(), nil
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

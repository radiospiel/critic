// Package fnmatch provides shell-style pattern matching (fnmatch) functionality.
// Patterns are converted to regular expressions.
package fnmatch

import (
	"errors"
	"regexp"
	"strings"
)

// Options configures fnmatch behavior.
type Options struct {
	// Separator is a character that * and ? will not match.
	// When empty, * matches any character. When set (e.g., "."),
	// * only matches characters within a single segment.
	Separator string
}

// Fnmatcher represents a compiled fnmatch pattern.
type Fnmatcher struct {
	pattern string
	re      *regexp.Regexp
}

// MustCompile compiles an fnmatch pattern and returns a Fnmatcher.
// Options can be provided to customize matching behavior.
// It panics if the pattern is invalid.
func MustCompile(pattern string, opts ...Options) *Fnmatcher {
	var opt Options
	if len(opts) > 0 {
		opt = opts[0]
	}

	regexStr, err := fnmatchToRegex(pattern, opt)
	if err != nil {
		panic(err)
	}
	re := regexp.MustCompile(regexStr)
	return &Fnmatcher{
		pattern: pattern,
		re:      re,
	}
}

// Match checks if the path matches the fnmatch pattern.
func (f *Fnmatcher) Match(path string) bool {
	return f.re.MatchString(path)
}

// Fnmatch checks if the path matches the fnmatch pattern.
// Options can be provided to customize matching behavior.
// When Separator is set, * only matches characters except the separator.
func Fnmatch(pattern, path string, opts ...Options) bool {
	return MustCompile(pattern, opts...).Match(path)
}

// fnmatchToRegex converts an fnmatch pattern to a regular expression string.
func fnmatchToRegex(pattern string, opt Options) (string, error) {
	var buf strings.Builder
	buf.WriteString("^")

	// Build the regex patterns for * and ? based on separator
	var starPattern, questionPattern string
	if opt.Separator != "" {
		// * and ? don't match the separator
		escapedSep := regexp.QuoteMeta(opt.Separator)
		starPattern = "[^" + escapedSep + "]*"
		questionPattern = "[^" + escapedSep + "]"
	} else {
		// * and ? match any character
		starPattern = ".*"
		questionPattern = "."
	}

	i := 0
	for i < len(pattern) {
		c := pattern[i]
		switch c {
		case '*':
			buf.WriteString(starPattern)
		case '?':
			buf.WriteString(questionPattern)
		case '[':
			// Find closing bracket
			j := i + 1
			// Handle [!...] and []...] edge cases
			if j < len(pattern) && (pattern[j] == '!' || pattern[j] == '^') {
				j++
			}
			if j < len(pattern) && pattern[j] == ']' {
				j++
			}
			for j < len(pattern) && pattern[j] != ']' {
				j++
			}
			if j >= len(pattern) {
				return "", errors.New("unclosed bracket")
			}
			// Copy bracket expression, converting ! to ^
			buf.WriteByte('[')
			if i+1 < len(pattern) && pattern[i+1] == '!' {
				buf.WriteByte('^')
				buf.WriteString(pattern[i+2 : j])
			} else {
				buf.WriteString(pattern[i+1 : j])
			}
			buf.WriteByte(']')
			i = j
		case '\\':
			// Escape next character
			if i+1 < len(pattern) {
				i++
				buf.WriteString(regexp.QuoteMeta(string(pattern[i])))
			}
		default:
			// Escape regex metacharacters
			buf.WriteString(regexp.QuoteMeta(string(c)))
		}
		i++
	}

	buf.WriteByte('$')
	return buf.String(), nil
}

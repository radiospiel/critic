// Package fnmatch provides shell-style pattern matching (fnmatch) functionality.
// Patterns are converted to regular expressions.
package fnmatch

import (
	"errors"
	"regexp"
	"strings"
)

// Fnmatcher represents a compiled fnmatch pattern.
type Fnmatcher struct {
	pattern string
	re      *regexp.Regexp
}

// MustCompile compiles an fnmatch pattern and returns a Fnmatcher.
// It panics if the pattern is invalid.
func MustCompile(pattern string) *Fnmatcher {
	regexStr, err := fnmatchToRegex(pattern)
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
// The wildcard * matches any characters including dots.
func Fnmatch(pattern, path string) bool {
	return MustCompile(pattern).Match(path)
}

// MustCompilePath compiles an fnmatch pattern for path matching and returns a Fnmatcher.
// Unlike MustCompile, the wildcard * only matches characters within a single segment
// (does not match the "." separator).
// It panics if the pattern is invalid.
func MustCompilePath(pattern string) *Fnmatcher {
	regexStr, err := fnmatchPathToRegex(pattern)
	if err != nil {
		panic(err)
	}
	re := regexp.MustCompile(regexStr)
	return &Fnmatcher{
		pattern: pattern,
		re:      re,
	}
}

// FnmatchPath checks if the key matches the fnmatch pattern using path semantics.
// The wildcard * matches any characters except "." (single segment only).
// This is suitable for matching dot-separated paths like "foo.bar.baz".
func FnmatchPath(pattern, key string) bool {
	return MustCompilePath(pattern).Match(key)
}

// fnmatchPathToRegex converts an fnmatch pattern to a regex where * doesn't match dots.
func fnmatchPathToRegex(pattern string) (string, error) {
	var buf strings.Builder
	buf.WriteString("^")

	i := 0
	for i < len(pattern) {
		c := pattern[i]
		switch c {
		case '*':
			buf.WriteString("[^.]*") // Match any char except dot
		case '?':
			buf.WriteString("[^.]") // Match single char except dot
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

// fnmatchToRegex converts an fnmatch pattern to a regular expression string.
func fnmatchToRegex(pattern string) (string, error) {
	var buf strings.Builder
	buf.WriteString("^")

	i := 0
	for i < len(pattern) {
		c := pattern[i]
		switch c {
		case '*':
			buf.WriteString(".*")
		case '?':
			buf.WriteByte('.')
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

// Package fnmatch provides shell-style pattern matching (fnmatch) functionality.
// Patterns are converted to regular expressions.
package fnmatch

import (
	"regexp"
	"strings"

	"git.15b.it/eno/critic/simple-go/preconditions"
)

type Matcher interface {
	// MatchString reports whether the string s
	// contains any match of the regular expression re.
	MatchString(s string) bool
}

// DefaultSeparators is used when no Options are provided.
// By default, * and ? do not match "/" or "\" (i.e. Unix and Windows
// path separators.
const DefaultSeparators = `/\`

// Options configures fnmatch behavior.
type Options struct {
	// Separators is a string where each character acts as a separator.
	// When empty, * matches any character. When set (e.g., "/."),
	// * only matches characters that are not separators.
	// For example, with Separators: "/.", "*" matches neither "/" nor ".".
	// Default (when no Options provided): "/\" - see DefaultSeparators.
	// To match any character, explicitly pass Options{Separators: ""}.
	Separators string
}

// MustCompile compiles an fnmatch pattern and returns a Fnmatcher.
// Options can be provided to customize matching behavior.
// It panics if the pattern is invalid.
func MustCompile(pattern string, opts ...Options) Matcher {
	re, err := Compile(pattern, opts...)
	preconditions.Check(err == nil, "failed to compile regexp: %v", err)
	return re
}

// Compile compiles an fnmatch pattern and returns a Matcher.
// Options can be provided to customize matching behavior.
// When no Options are provided, DefaultSeparators ("/\") is used.
// To match any character with *, pass Options{Separators: ""}.
func Compile(pattern string, opts ...Options) (Matcher, error) {
	preconditions.Check(len(opts) <= 1, "Only zero or one Options are allowed")

	// Use default separators when no options provided
	separators := DefaultSeparators
	if len(opts) > 0 {
		separators = opts[0].Separators
	}

	return regexp.Compile(fnmatchToRegex(pattern, separators))
}

// Fnmatch checks if the path matches the fnmatch pattern.
// Options can be provided to customize matching behavior.
// When Separator is set, * only matches characters except the separator.
func Fnmatch(pattern, path string, opts ...Options) bool {
	return MustCompile(pattern, opts...).MatchString(path)
}

// fnmatchToRegex converts an fnmatch pattern to a regular expression string.
func fnmatchToRegex(pattern string, separators string) string {
	var buf strings.Builder
	buf.WriteString("^")

	// Build the regex patterns for * and ? based on separators
	var starPattern, questionPattern string
	if separators != "" {
		// * and ? don't match any separator character
		// Build character class excluding all separator chars
		var escapedSeps strings.Builder
		for _, r := range separators {
			escapedSeps.WriteString(regexp.QuoteMeta(string(r)))
		}
		starPattern = "[^" + escapedSeps.String() + "]*"
		questionPattern = "[^" + escapedSeps.String() + "]"
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
	return buf.String()
}

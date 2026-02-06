package config

import (
	"regexp"
	"strings"

	"github.com/radiospiel/critic/simple-go/utils"
)

// PathspecMatch checks if a file path matches a gitignore-style pathspec glob pattern.
//
// Pattern rules follow git's pathspec glob matching as documented at:
// https://git-scm.com/docs/gitignore#_pattern_format
//
// Key matching rules:
//   - "*" matches anything except "/"
//   - "**" matches anything including "/"
//   - "?" matches a single character except "/"
//   - A leading "/" anchors the match to the repository root
//   - A pattern WITHOUT a "/" (other than leading) matches against the basename only
//   - A pattern WITH a "/" (other than leading) matches against the full path from root
//   - A leading "**/" matches in all directories
//   - A trailing "/**" matches everything inside a directory
//   - "/**/" in the middle matches zero or more directories
func PathspecMatch(pattern string, path string) bool {
	return CompilePathspec(pattern).MatchString(path)
}

// PathspecMatchAny returns true if the path matches any of the given patterns.
func PathspecMatchAny(patterns []string, path string) bool {
	for _, pattern := range patterns {
		if PathspecMatch(pattern, path) {
			return true
		}
	}
	return false
}

// Matcher is an interface for compiled pattern matchers.
type Matcher interface {
	MatchString(s string) bool
}

// pathspecCache caches compiled pathspec patterns to avoid repeated regex compilation.
var pathspecCache = utils.NewLRUCache(256, func(pattern string) (Matcher, error) {
	return pathspecToRegexp(pattern), nil
})

// CompilePathspec compiles a pathspec pattern and returns a Matcher.
// Results are cached.
func CompilePathspec(pattern string) Matcher {
	m, _ := pathspecCache.Get(pattern)
	return m
}

// pathspecToRegexp converts a gitignore-style pathspec pattern to a compiled regex.
//
// See https://git-scm.com/docs/gitignore#_pattern_format for the full specification.
func pathspecToRegexp(pattern string) *regexp.Regexp {
	// Handle leading /
	isRooted := false
	if len(pattern) > 0 && pattern[0] == '/' {
		isRooted = true
		pattern = pattern[1:]
	}

	// Check if pattern has a slash (which means it matches full path, not just basename).
	// A trailing slash doesn't count for this purpose.
	hasSlash := strings.Contains(strings.TrimRight(pattern, "/"), "/")

	var buf strings.Builder

	if !isRooted && !hasSlash {
		// No slash in pattern: match against the basename at any depth.
		// (?:.*/)? matches an optional directory prefix ending with /
		buf.WriteString("^(?:.*/)?")
	} else {
		// Has slash or is rooted: match from the root
		buf.WriteString("^")
	}

	i := 0
	for i < len(pattern) {
		c := pattern[i]
		switch c {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				// ** pattern
				if i+2 < len(pattern) && pattern[i+2] == '/' {
					// **/ - match zero or more directories
					buf.WriteString("(?:.*/)?")
					i += 3
					continue
				}
				// ** at end or without trailing / - match everything
				buf.WriteString(".*")
				i += 2
				continue
			}
			// Single * - match anything except /
			buf.WriteString("[^/]*")
		case '?':
			// ? - match single character except /
			buf.WriteString("[^/]")
		case '\\':
			// Escape next character
			if i+1 < len(pattern) {
				i++
				buf.WriteString(regexp.QuoteMeta(string(pattern[i])))
			}
		default:
			buf.WriteString(regexp.QuoteMeta(string(c)))
		}
		i++
	}

	buf.WriteString("$")
	return regexp.MustCompile(buf.String())
}

package config

import (
	"strings"

	"github.com/radiospiel/critic/simple-go/fnmatch"
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
//
// This function delegates to fnmatch for the actual matching, transforming the
// pattern to handle basename-only matching by prepending "**/" when needed.
func PathspecMatch(pattern string, path string) bool {
	return fnmatch.Fnmatch(pathspecToFnmatch(pattern), path)
}

// PathspecMatchAny returns true if the path matches any of the given patterns.
// Supports negation patterns: patterns starting with "!" exclude matching files.
// A file matches only if it matches at least one positive pattern and none of
// the negative patterns.
func PathspecMatchAny(patterns []string, path string) bool {
	var positive, negative []string
	for _, p := range patterns {
		if p == "" {
			continue
		}
		if p[0] == '!' {
			negative = append(negative, p[1:])
		} else {
			positive = append(positive, p)
		}
	}

	matched := false
	for _, p := range positive {
		if PathspecMatch(p, path) {
			matched = true
			break
		}
	}
	if !matched {
		return false
	}

	for _, p := range negative {
		if PathspecMatch(p, path) {
			return false
		}
	}
	return true
}

// pathspecToFnmatch transforms a gitignore-style pathspec pattern into an
// fnmatch pattern. The key transformation is for patterns without a slash:
// they match the basename at any depth, achieved by prepending "**/".
func pathspecToFnmatch(pattern string) string {
	// Handle leading /
	if len(pattern) > 0 && pattern[0] == '/' {
		// Rooted: strip leading / and match from root
		return pattern[1:]
	}

	// Check if pattern has a slash (which means it matches full path, not just basename).
	// A trailing slash doesn't count for this purpose.
	if strings.Contains(strings.TrimRight(pattern, "/"), "/") {
		// Has slash: match from the root
		return pattern
	}

	// No slash in pattern: match against the basename at any depth.
	// Prepend **/ so fnmatch matches at any directory level.
	return "**/" + pattern
}

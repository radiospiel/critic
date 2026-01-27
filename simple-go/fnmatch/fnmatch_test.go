package fnmatch

import (
	"testing"

	"github.org/radiospiel/critic/simple-go/assert"
)

func TestFnmatchToRegexp(t *testing.T) {
	tests := []struct {
		pattern  string
		expected string
	}{
		// Basic wildcards
		{"*.go", `^.*\.go$`},
		{"foo?.txt", `^foo.\.txt$`},
		{"*", `^.*$`},
		{"?", `^.$`},

		// Escaped regex metacharacters
		{"data.*.json", `^data\..*\.json$`},
		{"file(1).txt", `^file\(1\)\.txt$`},
		{"test+file.go", `^test\+file\.go$`},
		{"a$b", `^a\$b$`},
		{"a^b", `^a\^b$`},

		// Backslash escapes
		{`foo\*bar`, `^foo\*bar$`},
		{`foo\?bar`, `^foo\?bar$`},

		// Combined patterns
		{"src/**/*.go", `^src/.*.*/.*\.go$`},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			result := fnmatchToRegexp(tt.pattern, "") // no separators
			assert.Equals(t, result.String(), tt.expected, "pattern: %s", tt.pattern)
		})
	}
}

func TestFnmatchToRegexpWithSeparator(t *testing.T) {
	tests := []struct {
		pattern  string
		expected string
	}{
		// Basic wildcards - * matches anything except dots
		{"foo.*", `^foo\.[^\.]*$`},
		{"foo.?", `^foo\.[^\.]$`},
		{"*", `^[^\.]*$`},
		{"?", `^[^\.]$`},

		// Multi-segment patterns
		{"foo.*.bar", `^foo\.[^\.]*\.bar$`},
		{"*.*.baz", `^[^\.]*\.[^\.]*\.baz$`},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			result := fnmatchToRegexp(tt.pattern, ".") // dot separator
			assert.Equals(t, result.String(), tt.expected, "pattern: %s", tt.pattern)
		})
	}
}

func TestFnmatcherMatch(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		matches bool
	}{
		// Basic wildcards
		{"*.go", "main.go", true},
		{"*.go", "main.txt", false},
		{"*.go", "test.go", true},
		{"foo?.txt", "foo1.txt", true},
		{"foo?.txt", "foo12.txt", false},
		{"foo?.txt", "bar1.txt", false},

		// Dot files
		{"*.go", ".go", true},
		{"?*.go", "a.go", true},

		// Complex patterns
		{"data.*.json", "data.test.json", true},
		{"data.*.json", "data.json", false},
		{"src/*.go", "src/main.go", true},      // * matches "main" (. is not a separator by default)
		{"src/*.go", "src/sub/main.go", false}, // * does NOT match "/" with default separators

		// Exact matches
		{"exact.txt", "exact.txt", true},
		{"exact.txt", "other.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.path, func(t *testing.T) {
			matcher := MustCompile(tt.pattern)
			result := matcher.MatchString(tt.path)
			assert.Equals(t, result, tt.matches, "pattern: %s, path: %s", tt.pattern, tt.path)
		})
	}
}

func TestFnmatchFunction(t *testing.T) {
	// Test the convenience function
	assert.True(t, Fnmatch("*.go", "main.go"))
	assert.False(t, Fnmatch("*.go", "main.txt"))
}

func TestFnmatchWithSeparator(t *testing.T) {
	dotOpts := Options{Separators: "."}

	tests := []struct {
		pattern string
		key     string
		matches bool
	}{
		// Single segment wildcard
		{"foo.*", "foo.bar", true},
		{"foo.*", "foo.baz", true},
		{"foo.*", "foo.bar.baz", false}, // * does NOT match dots
		{"foo.*", "bar.baz", false},

		// Multi-segment patterns
		{"foo.*.bar", "foo.x.bar", true},
		{"foo.*.bar", "foo.y.bar", true},
		{"foo.*.bar", "foo.x.y.bar", false}, // * doesn't cross segments
		{"foo.*.bar", "foo.bar", false},     // * must match something

		// Nested wildcards
		{"*.*.baz", "a.b.baz", true},
		{"*.*.baz", "x.y.baz", true},
		{"*.*.baz", "a.baz", false},
		{"*.*.baz", "a.b.c.baz", false},

		// Question mark (single char, not dot)
		{"foo.?", "foo.a", true},
		{"foo.?", "foo.ab", false},
		{"foo.?", "foo.", false},

		// Exact matches
		{"foo.bar.baz", "foo.bar.baz", true},
		{"foo.bar.baz", "foo.bar.qux", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.key, func(t *testing.T) {
			result := Fnmatch(tt.pattern, tt.key, dotOpts)
			assert.Equals(t, result, tt.matches, "pattern: %s, key: %s", tt.pattern, tt.key)
		})
	}
}

func TestFnmatchWithMultipleSeparators(t *testing.T) {
	// With separators "/.", * matches neither "/" nor "."
	opts := Options{Separators: "/."}

	tests := []struct {
		pattern string
		path    string
		matches bool
	}{
		// * doesn't match dots
		{"foo.*", "foo.bar", true},
		{"foo.*", "foo.bar.baz", false},

		// * doesn't match slashes
		{"src/*", "src/main", true},
		{"src/*", "src/sub/main", false},

		// * doesn't match either
		{"*", "foo", true},
		{"*", "foo.bar", false},
		{"*", "foo/bar", false},

		// Multi-segment with mixed separators
		{"src/*.go", "src/main.go", true},  // * matches "main" (no / or . in it)
		{"src/*.*", "src/main.go", true},   // each * matches a single segment
		{"src/*", "src/main.go", false},    // * can't match the dot in "main.go"

		// Duplicate separators in options should work the same
		{"foo.*", "foo.bar", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.path, func(t *testing.T) {
			result := Fnmatch(tt.pattern, tt.path, opts)
			assert.Equals(t, result, tt.matches, "pattern: %s, path: %s", tt.pattern, tt.path)
		})
	}

	// Test with duplicate separators (should behave same as unique)
	dupeOpts := Options{Separators: "/../.."}
	assert.True(t, Fnmatch("*", "foo", dupeOpts))
	assert.False(t, Fnmatch("*", "foo.bar", dupeOpts))
	assert.False(t, Fnmatch("*", "foo/bar", dupeOpts))
}

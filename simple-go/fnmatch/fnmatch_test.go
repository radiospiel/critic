package fnmatch

import (
	"testing"

	"git.15b.it/eno/critic/simple-go/assert"
)

func TestFnmatchToRegex(t *testing.T) {
	tests := []struct {
		pattern  string
		expected string
	}{
		// Basic wildcards
		{"*.go", `^.*\.go$`},
		{"foo?.txt", `^foo.\.txt$`},
		{"*", `^.*$`},
		{"?", `^.$`},

		// Character classes
		{"[abc]*.log", `^[abc].*\.log$`},
		{"[!0-9]*", `^[^0-9].*$`},
		{"[a-z]", `^[a-z]$`},

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
		{"test_[0-9].txt", `^test_[0-9]\.txt$`},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			result, err := fnmatchToRegex(tt.pattern)
			assert.NoError(t, err)
			assert.Equals(t, result, tt.expected, "pattern: %s", tt.pattern)
		})
	}
}

func TestFnmatchToRegexError(t *testing.T) {
	// Unclosed bracket
	_, err := fnmatchToRegex("[abc")
	assert.Error(t, err, "unclosed bracket")
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

		// Character classes
		{"[abc]*.log", "a.log", true},
		{"[abc]*.log", "abc.log", true},
		{"[abc]*.log", "d.log", false},
		{"[!0-9]*", "abc", true},
		{"[!0-9]*", "1abc", false},

		// Dot files
		{"*.go", ".go", true},
		{"?*.go", "a.go", true},

		// Complex patterns
		{"data.*.json", "data.test.json", true},
		{"data.*.json", "data.json", false},
		{"src/*.go", "src/main.go", true},
		{"src/*.go", "src/sub/main.go", true}, // * matches any char including /

		// Exact matches
		{"exact.txt", "exact.txt", true},
		{"exact.txt", "other.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.path, func(t *testing.T) {
			matcher := MustCompile(tt.pattern)
			result := matcher.Match(tt.path)
			assert.Equals(t, result, tt.matches, "pattern: %s, path: %s", tt.pattern, tt.path)
		})
	}
}

func TestMustCompilePanic(t *testing.T) {
	defer func() {
		r := recover()
		assert.NotNil(t, r, "expected panic for invalid pattern")
	}()
	MustCompile("[abc")
}

func TestFnmatchFunction(t *testing.T) {
	// Test the convenience function
	assert.True(t, Fnmatch("*.go", "main.go"))
	assert.False(t, Fnmatch("*.go", "main.txt"))
	assert.True(t, Fnmatch("[a-z]*.txt", "hello.txt"))
	assert.False(t, Fnmatch("[a-z]*.txt", "123.txt"))
}

func TestFnmatchCaching(t *testing.T) {
	// Call multiple times with the same pattern to exercise the cache
	for i := 0; i < 10; i++ {
		assert.True(t, Fnmatch("*.go", "test.go"))
	}

	// Use different patterns to populate cache
	patterns := []string{"*.go", "*.txt", "*.md", "*.json", "*.yaml"}
	for _, p := range patterns {
		Fnmatch(p, "test"+p[1:])
	}
}

func TestFnmatchPathToRegex(t *testing.T) {
	tests := []struct {
		pattern  string
		expected string
	}{
		// Basic wildcards - * matches anything except dots
		{"foo.*", `^foo\.[^.]*$`},
		{"foo.?", `^foo\.[^.]$`},
		{"*", `^[^.]*$`},
		{"?", `^[^.]$`},

		// Multi-segment patterns
		{"foo.*.bar", `^foo\.[^.]*\.bar$`},
		{"*.*.baz", `^[^.]*\.[^.]*\.baz$`},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			result, err := fnmatchPathToRegex(tt.pattern)
			assert.NoError(t, err)
			assert.Equals(t, result, tt.expected, "pattern: %s", tt.pattern)
		})
	}
}

func TestFnmatchPath(t *testing.T) {
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

		// Character classes
		{"foo.[abc]", "foo.a", true},
		{"foo.[abc]", "foo.d", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.key, func(t *testing.T) {
			result := FnmatchPath(tt.pattern, tt.key)
			assert.Equals(t, result, tt.matches, "pattern: %s, key: %s", tt.pattern, tt.key)
		})
	}
}

func TestMustCompilePathPanic(t *testing.T) {
	defer func() {
		r := recover()
		assert.NotNil(t, r, "expected panic for invalid pattern")
	}()
	MustCompilePath("[abc")
}

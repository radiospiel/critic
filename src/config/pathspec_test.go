package config

import (
	"testing"

	"github.com/radiospiel/critic/simple-go/assert"
)

// Tests for git pathspec/gitignore-style glob matching.
// Pattern rules follow https://git-scm.com/docs/gitignore#_pattern_format

func TestPathspecMatch_BasicWildcard(t *testing.T) {
	// * matches anything except /
	tests := []struct {
		pattern string
		path    string
		matches bool
	}{
		{"*.go", "main.go", true},
		{"*.go", "src/main.go", true},       // no slash in pattern => matches basename
		{"*.go", "src/pkg/main.go", true},    // matches at any depth
		{"*.go", "main.txt", false},
		{"*.go", "main.go.bak", false},
		{"*_test.go", "main_test.go", true},
		{"*_test.go", "src/main_test.go", true},
		{"*_test.go", "src/pkg/main_test.go", true},
		{"*_test.go", "main.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.path, func(t *testing.T) {
			result := PathspecMatch(tt.pattern, tt.path)
			assert.Equals(t, result, tt.matches, "PathspecMatch(%q, %q)", tt.pattern, tt.path)
		})
	}
}

func TestPathspecMatch_QuestionMark(t *testing.T) {
	// ? matches exactly one character except /
	tests := []struct {
		pattern string
		path    string
		matches bool
	}{
		{"file?.go", "file1.go", true},
		{"file?.go", "fileA.go", true},
		{"file?.go", "file12.go", false},    // ? matches exactly one
		{"file?.go", "file.go", false},      // ? must match something
		{"file?.go", "src/file1.go", true},  // basename matching
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.path, func(t *testing.T) {
			result := PathspecMatch(tt.pattern, tt.path)
			assert.Equals(t, result, tt.matches, "PathspecMatch(%q, %q)", tt.pattern, tt.path)
		})
	}
}

func TestPathspecMatch_DoubleStarPrefix(t *testing.T) {
	// Leading ** followed by / matches in all directories
	tests := []struct {
		pattern string
		path    string
		matches bool
	}{
		{"**/test.go", "test.go", true},
		{"**/test.go", "src/test.go", true},
		{"**/test.go", "src/pkg/test.go", true},
		{"**/test.go", "test.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.path, func(t *testing.T) {
			result := PathspecMatch(tt.pattern, tt.path)
			assert.Equals(t, result, tt.matches, "PathspecMatch(%q, %q)", tt.pattern, tt.path)
		})
	}
}

func TestPathspecMatch_DoubleStarSuffix(t *testing.T) {
	// Trailing /** matches everything inside
	tests := []struct {
		pattern string
		path    string
		matches bool
	}{
		{"test/**", "test/a.go", true},
		{"test/**", "test/sub/a.go", true},
		{"test/**", "test/sub/deep/a.go", true},
		{"test/**", "other/a.go", false},
		{"test/**", "test", false}, // directory itself doesn't match trailing /**
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.path, func(t *testing.T) {
			result := PathspecMatch(tt.pattern, tt.path)
			assert.Equals(t, result, tt.matches, "PathspecMatch(%q, %q)", tt.pattern, tt.path)
		})
	}
}

func TestPathspecMatch_DoubleStarMiddle(t *testing.T) {
	// /**/ in the middle matches zero or more directories
	tests := []struct {
		pattern string
		path    string
		matches bool
	}{
		{"src/**/test.go", "src/test.go", true},         // zero directories
		{"src/**/test.go", "src/pkg/test.go", true},     // one directory
		{"src/**/test.go", "src/a/b/test.go", true},     // multiple directories
		{"src/**/test.go", "other/test.go", false},
		{"src/**/*.go", "src/main.go", true},
		{"src/**/*.go", "src/pkg/main.go", true},
		{"src/**/*.go", "src/a/b/main.go", true},
		{"src/**/*.go", "src/main.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.path, func(t *testing.T) {
			result := PathspecMatch(tt.pattern, tt.path)
			assert.Equals(t, result, tt.matches, "PathspecMatch(%q, %q)", tt.pattern, tt.path)
		})
	}
}

func TestPathspecMatch_LeadingSlash(t *testing.T) {
	// Leading / anchors the pattern to the root
	tests := []struct {
		pattern string
		path    string
		matches bool
	}{
		{"/test/*", "test/a.go", true},
		{"/test/*", "test/b.txt", true},
		{"/test/*", "src/test/a.go", false},  // anchored to root
		{"/test/*", "test/sub/a.go", false},  // * doesn't match /
		{"/src/*.go", "src/main.go", true},
		{"/src/*.go", "lib/src/main.go", false}, // anchored
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.path, func(t *testing.T) {
			result := PathspecMatch(tt.pattern, tt.path)
			assert.Equals(t, result, tt.matches, "PathspecMatch(%q, %q)", tt.pattern, tt.path)
		})
	}
}

func TestPathspecMatch_SlashInPattern(t *testing.T) {
	// A pattern containing a slash (other than leading) matches against the full path
	tests := []struct {
		pattern string
		path    string
		matches bool
	}{
		{"src/*.go", "src/main.go", true},
		{"src/*.go", "lib/src/main.go", false},    // slash in pattern => match from root
		{"test/fixtures/*", "test/fixtures/a.go", true},
		{"test/fixtures/*", "test/fixtures/sub/a.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.path, func(t *testing.T) {
			result := PathspecMatch(tt.pattern, tt.path)
			assert.Equals(t, result, tt.matches, "PathspecMatch(%q, %q)", tt.pattern, tt.path)
		})
	}
}

func TestPathspecMatch_DotPattern(t *testing.T) {
	// .* pattern matches hidden files (dot-prefixed)
	tests := []struct {
		pattern string
		path    string
		matches bool
	}{
		{".*", ".gitignore", true},
		{".*", ".env", true},
		{".*", ".hidden", true},
		{".*", "visible.txt", false},
		{".*", "src/.env", true},          // basename matching
		{".*", "src/pkg/.hidden", true},   // basename matching at depth
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.path, func(t *testing.T) {
			result := PathspecMatch(tt.pattern, tt.path)
			assert.Equals(t, result, tt.matches, "PathspecMatch(%q, %q)", tt.pattern, tt.path)
		})
	}
}

func TestPathspecMatch_NoSlashBasenameOnly(t *testing.T) {
	// A pattern without a slash matches the basename only
	tests := []struct {
		pattern string
		path    string
		matches bool
	}{
		{"Makefile", "Makefile", true},
		{"Makefile", "src/Makefile", true},
		{"Makefile", "src/pkg/Makefile", true},
		{"README.md", "README.md", true},
		{"README.md", "docs/README.md", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.path, func(t *testing.T) {
			result := PathspecMatch(tt.pattern, tt.path)
			assert.Equals(t, result, tt.matches, "PathspecMatch(%q, %q)", tt.pattern, tt.path)
		})
	}
}

func TestPathspecMatchAny(t *testing.T) {
	tests := []struct {
		patterns []string
		path     string
		matches  bool
	}{
		{[]string{"*.go", "*.ts"}, "main.go", true},
		{[]string{"*.go", "*.ts"}, "app.ts", true},
		{[]string{"*.go", "*.ts"}, "readme.md", false},
		{[]string{}, "anything.go", false},
		{nil, "anything.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := PathspecMatchAny(tt.patterns, tt.path)
			assert.Equals(t, result, tt.matches, "PathspecMatchAny(%v, %q)", tt.patterns, tt.path)
		})
	}
}

func TestPathspecMatchAny_Negation(t *testing.T) {
	// Negation patterns: a file must match at least one positive pattern
	// and none of the negative patterns.
	tests := []struct {
		patterns []string
		path     string
		matches  bool
	}{
		// Basic negation
		{[]string{"*.go", "!*_test.go"}, "main.go", true},
		{[]string{"*.go", "!*_test.go"}, "main_test.go", false},
		// Negation with doublestar
		{[]string{"src/**/*.go", "!*_test.go"}, "src/config/project.go", true},
		{[]string{"src/**/*.go", "!*_test.go"}, "src/config/project_test.go", false},
		// Only negative patterns — no positive match
		{[]string{"!*_test.go"}, "main.go", false},
		// Empty patterns
		{[]string{}, "main.go", false},
		{nil, "main.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := PathspecMatchAny(tt.patterns, tt.path)
			assert.Equals(t, result, tt.matches, "PathspecMatchAny(%v, %q)", tt.patterns, tt.path)
		})
	}
}

package config

import (
	"testing"

	"github.org/radiospiel/critic/simple-go/assert"
)

func TestHasExtension(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		extensions []string
		want       bool
	}{
		{
			name:       "Empty extensions list allows all",
			path:       "file.xyz",
			extensions: nil,
			want:       true,
		},
		{
			name:       "Matching extension",
			path:       "main.go",
			extensions: []string{"go", "rs"},
			want:       true,
		},
		{
			name:       "Non-matching extension",
			path:       "main.go",
			extensions: []string{"rs", "c"},
			want:       false,
		},
		{
			name:       "Path with directory",
			path:       "src/main.go",
			extensions: []string{"go"},
			want:       true,
		},
		{
			name:       "No extension",
			path:       "Makefile",
			extensions: []string{"go"},
			want:       false,
		},
		{
			name:       "Dot in directory name",
			path:       "src.old/main.go",
			extensions: []string{"go"},
			want:       true,
		},
		{
			name:       "Hidden file with extension",
			path:       ".gitignore",
			extensions: []string{"gitignore"},
			want:       true,
		},
		{
			name:       "Multiple dots",
			path:       "file.test.go",
			extensions: []string{"go"},
			want:       true,
		},
		{
			name:       "Case sensitive match",
			path:       "README.MD",
			extensions: []string{"md"},
			want:       false,
		},
		{
			name:       "Case sensitive match uppercase",
			path:       "README.MD",
			extensions: []string{"MD"},
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasExtension(tt.path, tt.extensions)
			assert.Equals(t, got, tt.want, "HasExtension(%q, %v)", tt.path, tt.extensions)
		})
	}
}

func TestDefaultFileExtensions(t *testing.T) {
	// Verify that DefaultFileExtensions is not empty
	assert.True(t, len(DefaultFileExtensions) > 0, "DefaultFileExtensions should not be empty")

	// Verify some common extensions are present
	requiredExtensions := []string{"go", "rs", "py", "js", "md"}
	for _, ext := range requiredExtensions {
		assert.Contains(t, DefaultFileExtensions, ext, "DefaultFileExtensions missing required extension: %s", ext)
	}
}

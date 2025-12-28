package config

import "testing"

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
			if got != tt.want {
				t.Errorf("HasExtension(%q, %v) = %v, want %v",
					tt.path, tt.extensions, got, tt.want)
			}
		})
	}
}

func TestDefaultFileExtensions(t *testing.T) {
	// Verify that DefaultFileExtensions is not empty
	if len(DefaultFileExtensions) == 0 {
		t.Error("DefaultFileExtensions should not be empty")
	}

	// Verify some common extensions are present
	requiredExtensions := []string{"go", "rs", "py", "js", "md"}
	for _, ext := range requiredExtensions {
		found := false
		for _, def := range DefaultFileExtensions {
			if def == ext {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("DefaultFileExtensions missing required extension: %s", ext)
		}
	}
}

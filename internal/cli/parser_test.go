package cli

import (
	"testing"

	"git.15b.it/eno/critic/internal/app"
	"git.15b.it/eno/critic/simple-go/assert"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expected      *app.Args
		expectedError string
	}{
		{
			name: "Empty args uses defaults",
			args: []string{},
			expected: &app.Args{
				Bases:      nil, // No bases specified - app layer will add defaults
				Paths:      []string{"."},
				Extensions: []string{}, // No extensions specified - app layer will add defaults
			},
		},
		{
			name: "Custom extensions",
			args: []string{"--extensions=go,rs"},
			expected: &app.Args{
				Bases:      nil, // No bases specified - app layer will add defaults
				Paths:      []string{"."},
				Extensions: []string{"go", "rs"},
			},
		},
		{
			name: "Single base",
			args: []string{"main"},
			expected: &app.Args{
				Bases:      []string{"main"},
				Paths:      []string{"."},
				Extensions: []string{},
			},
		},
		{
			name: "Multiple bases",
			args: []string{"merge-base,origin/main,HEAD"},
			expected: &app.Args{
				Bases:      []string{"merge-base", "origin/main", "HEAD"},
				Paths:      []string{"."},
				Extensions: []string{},
			},
		},
		{
			name: "With explicit paths",
			args: []string{"main", "--", "src", "tests"},
			expected: &app.Args{
				Bases:      []string{"main"},
				Paths:      []string{"src", "tests"},
				Extensions: []string{},
			},
		},
		{
			name: "Extensions and paths",
			args: []string{"--extensions=go,rs", "main", "--", "src"},
			expected: &app.Args{
				Bases:      []string{"main"},
				Paths:      []string{"src"},
				Extensions: []string{"go", "rs"},
			},
		},
		{
			name:          "Unknown flag",
			args:          []string{"--unknown"},
			expectedError: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := ParseArgsForTesting(tt.args)
			if tt.expectedError != "" {
				assert.Error(t, err, tt.expectedError)
				return
			}
			assert.NoError(t, err)

			assert.Equals(t, actual.Extensions, tt.expected.Extensions)
			assert.Equals(t, actual.Bases, tt.expected.Bases)
			assert.Equals(t, actual.Paths, tt.expected.Paths)
		})
	}
}

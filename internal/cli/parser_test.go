package cli

import (
	"testing"

	"git.15b.it/eno/critic/internal/app"
	"git.15b.it/eno/critic/internal/assert"
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
				Current:    "current",
				Paths:      []string{"."},
				Extensions: []string{}, // No extensions specified - app layer will add defaults
			},
		},
		{
			name: "Custom extensions",
			args: []string{"--extensions=go,rs"},
			expected: &app.Args{
				Bases:      nil, // No bases specified - app layer will add defaults
				Current:    "current",
				Paths:      []string{"."},
				Extensions: []string{"go", "rs"},
			},
		},
		{
			name: "Single base to current",
			args: []string{"main..current"},
			expected: &app.Args{
				Bases:      []string{"main"},
				Current:    "current",
				Paths:      []string{"."},
				Extensions: []string{},
			},
		},
		{
			name: "Multiple bases to current",
			args: []string{"merge-base,origin/main,HEAD..current"},
			expected: &app.Args{
				Bases:      []string{"merge-base", "origin/main", "HEAD"},
				Current:    "current",
				Paths:      []string{"."},
				Extensions: []string{},
			},
		},
		{
			name: "Bases to specific ref",
			args: []string{"main,develop..v1.0.0"},
			expected: &app.Args{
				Bases:      []string{"main", "develop"},
				Current:    "v1.0.0",
				Paths:      []string{"."},
				Extensions: []string{},
			},
		},
		{
			name: "With explicit paths",
			args: []string{"main..current", "--", "src", "tests"},
			expected: &app.Args{
				Bases:      []string{"main"},
				Current:    "current",
				Paths:      []string{"src", "tests"},
				Extensions: []string{},
			},
		},
		{
			name: "Extensions and paths",
			args: []string{"--extensions=go,rs", "main..current", "--", "src"},
			expected: &app.Args{
				Bases:      []string{"main"},
				Current:    "current",
				Paths:      []string{"src"},
				Extensions: []string{"go", "rs"},
			},
		},
		{
			name: "Just bases (no current specified)",
			args: []string{"main,develop"},
			expected: &app.Args{
				Bases:      []string{"main", "develop"},
				Current:    "current", // Default
				Paths:      []string{"."},
				Extensions: []string{},
			},
		},
		{
			name:          "Unknown flag",
			args:          []string{"--unknown"},
			expectedError: "unknown",
		},
		{
			name:          "Too many .. separators",
			args:          []string{"a..b..c"},
			expectedError: "too many '..' separators",
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
			assert.Equals(t, actual.Current, tt.expected.Current)
			assert.Equals(t, actual.Paths, tt.expected.Paths)
		})
	}
}

func TestParseBasesCurrent(t *testing.T) {
	tests := []struct {
		name            string
		arg             string
		expectedBases   []string
		expectedCurrent string
		expectedError   string
	}{
		{
			name:            "Single base",
			arg:             "main..current",
			expectedBases:   []string{"main"},
			expectedCurrent: "current",
		},
		{
			name:            "Multiple bases",
			arg:             "a,b,c..current",
			expectedBases:   []string{"a", "b", "c"},
			expectedCurrent: "current",
		},
		{
			name:            "No current (just bases)",
			arg:             "main,develop",
			expectedBases:   []string{"main", "develop"},
			expectedCurrent: "current", // Default
		},
		{
			name:            "Empty bases",
			arg:             "..current",
			expectedBases:   nil,
			expectedCurrent: "current",
		},
		{
			name:          "Too many separators",
			arg:           "a..b..c",
			expectedError: "too many '..' separators",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := &app.Args{
				Current: "current", // Default
			}
			err := parseBasesCurrent(tt.arg, actual)
			if tt.expectedError != "" {
				assert.Error(t, err, tt.expectedError)
				return
			}
			assert.NoError(t, err)

			assert.Equals(t, actual.Bases, tt.expectedBases)
			assert.Equals(t, actual.Current, tt.expectedCurrent)
		})
	}
}

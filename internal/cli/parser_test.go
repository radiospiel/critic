package cli

import (
	"reflect"
	"testing"

	"git.15b.it/eno/critic/internal/app"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    *app.Args
		wantErr bool
	}{
		{
			name: "Empty args uses defaults",
			args: []string{},
			want: &app.Args{
				Bases:      nil, // No bases specified - app layer will add defaults
				Current:    "current",
				Paths:      []string{"."},
				Extensions: nil, // Will check separately
			},
		},
		{
			name: "Custom extensions",
			args: []string{"--extensions=go,rs"},
			want: &app.Args{
				Bases:      nil, // No bases specified - app layer will add defaults
				Current:    "current",
				Paths:      []string{"."},
				Extensions: []string{"go", "rs"},
			},
		},
		{
			name: "Single base to current",
			args: []string{"main..current"},
			want: &app.Args{
				Bases:      []string{"main"},
				Current:    "current",
				Paths:      []string{"."},
				Extensions: nil,
			},
		},
		{
			name: "Multiple bases to current",
			args: []string{"merge-base,origin/main,HEAD..current"},
			want: &app.Args{
				Bases:      []string{"merge-base", "origin/main", "HEAD"},
				Current:    "current",
				Paths:      []string{"."},
				Extensions: nil,
			},
		},
		{
			name: "Bases to specific ref",
			args: []string{"main,develop..v1.0.0"},
			want: &app.Args{
				Bases:      []string{"main", "develop"},
				Current:    "v1.0.0",
				Paths:      []string{"."},
				Extensions: nil,
			},
		},
		{
			name: "With explicit paths",
			args: []string{"main..current", "--", "src", "tests"},
			want: &app.Args{
				Bases:      []string{"main"},
				Current:    "current",
				Paths:      []string{"src", "tests"},
				Extensions: nil,
			},
		},
		{
			name: "Extensions and paths",
			args: []string{"--extensions=go,rs", "main..current", "--", "src"},
			want: &app.Args{
				Bases:      []string{"main"},
				Current:    "current",
				Paths:      []string{"src"},
				Extensions: []string{"go", "rs"},
			},
		},
		{
			name: "Just bases (no current specified)",
			args: []string{"main,develop"},
			want: &app.Args{
				Bases:      []string{"main", "develop"},
				Current:    "current", // Default
				Paths:      []string{"."},
				Extensions: nil,
			},
		},
		{
			name:    "Unknown flag",
			args:    []string{"--unknown"},
			wantErr: true,
		},
		{
			name:    "Too many .. separators",
			args:    []string{"a..b..c"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseArgsForTesting(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Skip extensions check if expected is nil (will use defaults)
			if tt.want.Extensions != nil {
				if !reflect.DeepEqual(got.Extensions, tt.want.Extensions) {
					t.Errorf("Parse() Extensions = %v, want %v", got.Extensions, tt.want.Extensions)
				}
			} else {
				// Just check that extensions were set
				if len(got.Extensions) == 0 {
					t.Error("Parse() Extensions should not be empty when using defaults")
				}
			}

			// Check bases
			if !reflect.DeepEqual(got.Bases, tt.want.Bases) {
				t.Errorf("Parse() Bases = %v, want %v", got.Bases, tt.want.Bases)
			}

			if got.Current != tt.want.Current {
				t.Errorf("Parse() Current = %v, want %v", got.Current, tt.want.Current)
			}
			if !reflect.DeepEqual(got.Paths, tt.want.Paths) {
				t.Errorf("Parse() Paths = %v, want %v", got.Paths, tt.want.Paths)
			}
		})
	}
}

func TestParseBasesCurrent(t *testing.T) {
	tests := []struct {
		name      string
		arg       string
		wantBases []string
		wantCurrent string
		wantErr   bool
	}{
		{
			name:      "Single base",
			arg:       "main..current",
			wantBases: []string{"main"},
			wantCurrent: "current",
		},
		{
			name:      "Multiple bases",
			arg:       "a,b,c..current",
			wantBases: []string{"a", "b", "c"},
			wantCurrent: "current",
		},
		{
			name:      "No current (just bases)",
			arg:       "main,develop",
			wantBases: []string{"main", "develop"},
			wantCurrent: "current", // Default
		},
		{
			name:      "Empty bases",
			arg:       "..current",
			wantBases: nil,
			wantCurrent: "current",
		},
		{
			name:    "Too many separators",
			arg:     "a..b..c",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &app.Args{
				Current: "current", // Default
			}
			err := parseBasesCurrent(tt.arg, result)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseBasesCurrent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if !reflect.DeepEqual(result.Bases, tt.wantBases) {
				t.Errorf("parseBasesCurrent() Bases = %v, want %v", result.Bases, tt.wantBases)
			}
			if result.Current != tt.wantCurrent {
				t.Errorf("parseBasesCurrent() Current = %v, want %v", result.Current, tt.wantCurrent)
			}
		})
	}
}

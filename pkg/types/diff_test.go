package types

import "testing"

func TestLineType_String(t *testing.T) {
	tests := []struct {
		name     string
		lineType LineType
		want     string
	}{
		{
			name:     "Added line",
			lineType: LineAdded,
			want:     "+",
		},
		{
			name:     "Deleted line",
			lineType: LineDeleted,
			want:     "-",
		},
		{
			name:     "Context line",
			lineType: LineContext,
			want:     " ",
		},
		{
			name:     "Unknown line type",
			lineType: LineType(99),
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.lineType.String()
			if got != tt.want {
				t.Errorf("LineType.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStructCreation(t *testing.T) {
	// Test Diff creation
	diff := &Diff{
		Files: []*FileDiff{},
	}
	if diff == nil {
		t.Error("Failed to create Diff")
	}
	if diff.Files == nil {
		t.Error("Diff.Files should not be nil")
	}

	// Test FileDiff creation
	fileDiff := &FileDiff{
		OldPath:   "old.go",
		NewPath:   "new.go",
		IsRenamed: true,
		Hunks:     []*Hunk{},
	}
	if fileDiff.OldPath != "old.go" {
		t.Errorf("FileDiff.OldPath = %q, want %q", fileDiff.OldPath, "old.go")
	}
	if !fileDiff.IsRenamed {
		t.Error("FileDiff.IsRenamed should be true")
	}

	// Test Hunk creation
	hunk := &Hunk{
		OldStart: 10,
		OldLines: 5,
		NewStart: 15,
		NewLines: 6,
		Header:   "func main() {",
		Lines:    []*Line{},
	}
	if hunk.OldStart != 10 {
		t.Errorf("Hunk.OldStart = %d, want %d", hunk.OldStart, 10)
	}

	// Test Line creation
	line := &Line{
		Type:    LineAdded,
		Content: "fmt.Println(\"hello\")",
		OldNum:  0,
		NewNum:  16,
	}
	if line.Type != LineAdded {
		t.Errorf("Line.Type = %v, want %v", line.Type, LineAdded)
	}
	if line.OldNum != 0 {
		t.Errorf("Line.OldNum = %d, want %d for added line", line.OldNum, 0)
	}
}

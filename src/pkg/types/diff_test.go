package types

import (
	"testing"

	"github.com/radiospiel/critic/simple-go/assert"
)

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
			assert.Equals(t, tt.lineType.String(), tt.want)
		})
	}
}

func TestStructCreation(t *testing.T) {
	// Test FileDiff slice creation
	files := []*FileDiff{}
	assert.NotNil(t, files, "FileDiff slice should not be nil")

	// Test FileDiff creation
	fileDiff := &FileDiff{
		OldPath:   "old.go",
		NewPath:   "new.go",
		IsRenamed: true,
		Hunks:     []*Hunk{},
	}
	assert.Equals(t, fileDiff.OldPath, "old.go")
	assert.True(t, fileDiff.IsRenamed, "FileDiff.IsRenamed should be true")

	// Test Hunk creation
	hunk := &Hunk{
		OldStart: 10,
		OldLines: 5,
		NewStart: 15,
		NewLines: 6,
		Header:   "func main() {",
		Lines:    []*Line{},
	}
	assert.Equals(t, hunk.OldStart, 10)

	// Test Line creation
	line := &Line{
		Type:    LineAdded,
		Content: "fmt.Println(\"hello\")",
		OldNum:  0,
		NewNum:  16,
	}
	assert.Equals(t, line.Type, LineAdded)
	assert.Equals(t, line.OldNum, 0, "OldNum should be 0 for added line")
}

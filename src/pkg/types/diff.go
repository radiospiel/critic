package types

import (
	"encoding/json"

	"github.com/radiospiel/critic/simple-go/utils"
)

// FileDiff represents changes to a single file
type FileDiff struct {
	OldPath     string  `json:"old_path"`
	NewPath     string  `json:"new_path"`
	OldMode     string  `json:"old_mode,omitempty"`
	NewMode     string  `json:"new_mode,omitempty"`
	IsNew       bool    `json:"is_new,omitempty"`
	IsDeleted   bool    `json:"is_deleted,omitempty"`
	IsRenamed   bool    `json:"is_renamed,omitempty"`
	IsBinary    bool    `json:"is_binary,omitempty"`
	IsUntracked bool    `json:"is_untracked,omitempty"`
	Hunks       []*Hunk `json:"hunks"`
}

func (d FileDiff) GetPath() string {
	return utils.IfElse(d.IsDeleted, d.OldPath, d.NewPath)
}

// HunkStats holds line statistics for a hunk
type HunkStats struct {
	Added   int `json:"added"`
	Deleted int `json:"deleted"`
}

// Hunk represents a chunk of changes within a file
type Hunk struct {
	OldStart int       `json:"old_start"`
	OldLines int       `json:"old_lines"`
	NewStart int       `json:"new_start"`
	NewLines int       `json:"new_lines"`
	Header   string    `json:"header,omitempty"` // The @@ ... @@ header line
	Lines    []*Line   `json:"lines"`
	Stats    HunkStats `json:"stats"`
}

// Line represents a single line in a diff hunk
type Line struct {
	Type    LineType `json:"type"`
	Content string   `json:"content"`
	OldNum  int      `json:"old_num"` // Line number in old file (0 if added line)
	NewNum  int      `json:"new_num"` // Line number in new file (0 if deleted line)
}

// LineType represents the type of diff line
type LineType int

const (
	LineContext LineType = iota // Context line (no change)
	LineAdded                   // Added line (+)
	LineDeleted                 // Deleted line (-)
)

var lineTypeNames = map[LineType]string{
	LineContext: "LineContext",
	LineAdded:   "LineAdded",
	LineDeleted: "LineDeleted",
}

// String returns a string representation of the line type
func (lt LineType) String() string {
	switch lt {
	case LineAdded:
		return "+"
	case LineDeleted:
		return "-"
	case LineContext:
		return " "
	default:
		return ""
	}
}

// MarshalJSON implements json.Marshaler for LineType
func (lt LineType) MarshalJSON() ([]byte, error) {
	if name, ok := lineTypeNames[lt]; ok {
		return json.Marshal(name)
	}
	return json.Marshal("LineUnknown")
}

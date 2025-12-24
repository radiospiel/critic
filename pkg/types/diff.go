package types

// Diff represents a git diff with multiple file changes
type Diff struct {
	Files []*FileDiff
}

// FileDiff represents changes to a single file
type FileDiff struct {
	OldPath   string
	NewPath   string
	OldMode   string
	NewMode   string
	IsNew     bool
	IsDeleted bool
	IsRenamed bool
	IsBinary  bool
	Hunks     []*Hunk
}

// Hunk represents a chunk of changes within a file
type Hunk struct {
	OldStart int
	OldLines int
	NewStart int
	NewLines int
	Header   string // The @@ ... @@ header line
	Lines    []*Line
}

// Line represents a single line in a diff hunk
type Line struct {
	Type    LineType
	Content string
	OldNum  int // Line number in old file (0 if added line)
	NewNum  int // Line number in new file (0 if deleted line)
}

// LineType represents the type of diff line
type LineType int

const (
	LineContext LineType = iota // Context line (no change)
	LineAdded                   // Added line (+)
	LineDeleted                 // Deleted line (-)
)

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

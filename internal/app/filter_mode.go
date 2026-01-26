package app

// FilterMode represents the current file/hunk filter mode
type FilterMode int

const (
	// FilterModeNone shows all files and hunks (default)
	FilterModeNone FilterMode = iota
	// FilterModeWithComments shows only files with comments, and only hunks with comments
	FilterModeWithComments
	// FilterModeWithUnresolved shows only files with unresolved comments, and only hunks with unresolved comments
	FilterModeWithUnresolved
)

// String returns a display name for the filter mode
func (m FilterMode) String() string {
	switch m {
	case FilterModeWithComments:
		return "With Comments"
	case FilterModeWithUnresolved:
		return "Unresolved Only"
	default:
		return "All"
	}
}

package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Color palette
	colorSubtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	colorHighlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	colorGreen     = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	colorRed       = lipgloss.AdaptiveColor{Light: "#FF0000", Dark: "#FF6B6B"}
	colorYellow    = lipgloss.AdaptiveColor{Light: "#FFA500", Dark: "#FFD700"}

	// Base style
	baseStyle = lipgloss.NewStyle().
		Padding(0, 1)

	// File listView styles - active selection (full reverse)
	selectedFileActiveStyle = lipgloss.NewStyle().
		Reverse(true).
		Bold(true)

	// File listView styles - inactive selection (muted reverse)
	selectedFileInactiveStyle = lipgloss.NewStyle().
		Reverse(true).
		Faint(true)

	normalFileStyle = lipgloss.NewStyle()

	// Diff line styles - use adaptive colors for better terminal compatibility
	addedLineStyle = lipgloss.NewStyle().
		Background(lipgloss.AdaptiveColor{Light: "#d4f4dd", Dark: "#1a3a1a"}) // Greenish background

	deletedLineStyle = lipgloss.NewStyle().
		Background(lipgloss.AdaptiveColor{Light: "#ffdce0", Dark: "#3a1a1a"}) // Reddish background

	contextLineStyle = lipgloss.NewStyle()

	// Hunk header style
	hunkHeaderStyle = lipgloss.NewStyle().
		Foreground(colorHighlight).
		Bold(true)

	// Pane border styles
	activeBorderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorHighlight)

	inactiveBorderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorSubtle)

	// Title styles
	titleStyle = lipgloss.NewStyle().
		Foreground(colorHighlight).
		Bold(true).
		Padding(0, 1)

	// Help text style
	helpStyle = lipgloss.NewStyle().
		Foreground(colorSubtle).
		Padding(1, 0)

	// Status bar style
	statusStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("#6B95D8")). // Darker blue background
		Foreground(lipgloss.Color("#000000")). // Black text for contrast
		Padding(0, 1)
)

// GetStatusStyle returns the status bar style
func GetStatusStyle() lipgloss.Style {
	return statusStyle
}

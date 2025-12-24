package ui

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

	// File list styles
	selectedFileStyle = lipgloss.NewStyle().
				Reverse(true).
				Bold(true)

	normalFileStyle = lipgloss.NewStyle()

	// Diff line styles
	addedLineStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1a3a1a")) // Dark greenish background

	deletedLineStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#3a1a1a")) // Dark reddish background

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
			Foreground(colorSubtle).
			Padding(0, 1)
)

// ApplyDiffLineStyle applies the appropriate style based on line type
func ApplyDiffLineStyle(lineType rune, content string) string {
	switch lineType {
	case '+':
		return addedLineStyle.Render(content)
	case '-':
		return deletedLineStyle.Render(content)
	default:
		return contextLineStyle.Render(content)
	}
}

// RenderTitle renders a pane title
func RenderTitle(title string, active bool) string {
	style := titleStyle
	if !active {
		style = style.Foreground(colorSubtle)
	}
	return style.Render(title)
}

// RenderBorder renders content with a border
func RenderBorder(content string, active bool, title string) string {
	style := inactiveBorderStyle
	if active {
		style = activeBorderStyle
	}

	if title != "" {
		style = style.BorderTop(true).BorderLeft(true).BorderRight(true).BorderBottom(true)
		return style.Render(content)
	}

	return style.Render(content)
}

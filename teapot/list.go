package teapot

import (
	"strings"

	"git.15b.it/eno/critic/simple-go/logger"
	"git.15b.it/eno/critic/simple-go/utils"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ListItem represents a single item in a List.
type ListItem interface {
	// FilterValue returns the string used for filtering/searching.
	FilterValue() string
}

// ItemRenderer is a function that renders a list item to a buffer line.
// It receives the item, whether it's selected, whether the list is focused,
// and the available width. It should render to the buffer at y=0.
type ItemRenderer[T ListItem] func(buf *SubBuffer, item T, selected bool, focused bool, width int)

// List is a generic scrollable list with selection.
// It can be used for file lists, branch selectors, commit selectors, etc.
type List[T ListItem] struct {
	BaseView
	items        []T
	selected     int // Index of selected item
	scrollOffset int // First visible item index
	renderer     ItemRenderer[T]

	// Styles
	selectedStyle          lipgloss.Style
	selectedUnfocusedStyle lipgloss.Style
	normalStyle            lipgloss.Style

	// Callbacks
	onChange func(items []T) // Called when items change
}

// NewSelectableList creates a new selectable list with the given renderer.
func NewSelectableList[T ListItem](renderer ItemRenderer[T]) *List[T] {
	list := &List[T]{
		BaseView: NewBaseView(),
		renderer: renderer,
		selectedStyle: lipgloss.NewStyle().
			Bold(true).
			Reverse(true),
		selectedUnfocusedStyle: lipgloss.NewStyle().
			Faint(true).
			Reverse(true),
		normalStyle: lipgloss.NewStyle(),
	}
	return list
}

// SetItems sets the list items.
func (l *List[T]) SetItems(items []T) {
	l.items = items
	// Adjust selection if needed
	if l.selected >= len(items) {
		l.selected = max(0, len(items)-1)
	}
	l.ensureVisible()
	l.Repaint() // Mark as dirty for compositor re-render
	if l.onChange != nil {
		l.onChange(items)
	}
}

// Items returns all items.
func (l *List[T]) Items() []T {
	return l.items
}

// Selected returns the currently selected item.
func (l *List[T]) Selected() (T, bool) {
	var zero T
	if l.selected < 0 || l.selected >= len(l.items) {
		return zero, false
	}
	return l.items[l.selected], true
}

// SelectedIndex returns the index of the selected item.
func (l *List[T]) SelectedIndex() int {
	return l.selected
}

// SetSelectedIndex sets the selected index.
func (l *List[T]) SetSelectedIndex(index int) {
	l.selected = utils.Clamp(index, 0, len(l.items)-1)
	l.ensureVisible()
	l.Repaint() // Mark as dirty for compositor re-render
}

// SetStyles sets the selection styles.
func (l *List[T]) SetStyles(selected, selectedUnfocused, normal lipgloss.Style) {
	l.selectedStyle = selected
	l.selectedUnfocusedStyle = selectedUnfocused
	l.normalStyle = normal
}

// OnChange sets a callback for when items change.
func (l *List[T]) OnChange(fn func(items []T)) {
	l.onChange = fn
}

// visibleCount returns the number of visible items.
func (l *List[T]) visibleCount() int {
	return l.bounds.Height
}

// ensureVisible ensures the selected item is visible.
func (l *List[T]) ensureVisible() {
	visible := l.visibleCount()
	if visible <= 0 {
		return
	}

	// Scroll up if selection is above viewport
	if l.selected < l.scrollOffset {
		l.scrollOffset = l.selected
	}

	// Scroll down if selection is below viewport
	if l.selected >= l.scrollOffset+visible {
		l.scrollOffset = l.selected - visible + 1
	}
}

// Render renders the list to the buffer.
func (l *List[T]) Render(buf *SubBuffer) {
	visible := l.visibleCount()
	if visible <= 0 || len(l.items) == 0 {
		return
	}

	for i := 0; i < visible; i++ {
		itemIdx := l.scrollOffset + i
		if itemIdx >= len(l.items) {
			break
		}

		item := l.items[itemIdx]
		isSelected := itemIdx == l.selected

		// Create a sub-buffer for this line
		lineBuf := NewSubBuffer(buf.parent, Rect{
			Position{
				X: buf.offset.X,
				Y: buf.offset.Y + i,
			},
			Size{
				Width:  buf.Width(),
				Height: 1,
			},
		})

		// Use renderer if provided
		if l.renderer != nil {
			l.renderer(lineBuf, item, isSelected, l.focused, buf.Width())
		} else {
			// Default rendering
			style := l.normalStyle
			if isSelected {
				if l.focused {
					style = l.selectedStyle
				} else {
					style = l.selectedUnfocusedStyle
				}
			}

			// Render filter value as fallback
			text := item.FilterValue()
			lineBuf.SetStringTruncated(0, 0, text, buf.Width(), style)

			// Fill remaining width with style background
			remaining := buf.Width() - len(text)
			if remaining > 0 {
				lineBuf.SetString(len(text), 0, strings.Repeat(" ", remaining), style)
			}
		}
	}
}

// HandleKey handles keyboard input.
func (l *List[T]) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	logger.Info("*** List.HandleKey")

	switch msg.String() {
	case "up", "k":
		l.moveUp()
		return true, nil
	case "down", "j":
		l.moveDown()
		return true, nil
	case "home", "g":
		if len(l.items) > 0 {
			l.SetSelectedIndex(0)
		}
		return true, nil
	case "end", "G":
		if len(l.items) > 0 {
			l.SetSelectedIndex(len(l.items) - 1)
		}
		return true, nil
	case "pgup":
		l.pageUp()
		return true, nil
	case "pgdown":
		l.pageDown()
		return true, nil
	case "enter":
		return true, nil
	}
	return false, nil
}

func (l *List[T]) moveUp() {
	if l.selected > 0 {
		l.selected--
		l.ensureVisible()
		l.Repaint()
	}
}

func (l *List[T]) moveDown() {
	if l.selected < len(l.items)-1 {
		l.selected++
		l.ensureVisible()
		l.Repaint()
	}
}

func (l *List[T]) pageUp() {
	visible := l.visibleCount()
	newSelected := max(0, l.selected-visible)
	l.SetSelectedIndex(newSelected)
}

func (l *List[T]) pageDown() {
	visible := l.visibleCount()
	newSelected := min(len(l.items)-1, l.selected+visible)
	l.SetSelectedIndex(newSelected)
}

package teapot

import (
	"strings"

	"git.15b.it/eno/critic/simple-go/utils"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ListItem represents a single item in a SelectableList.
type ListItem interface {
	// FilterValue returns the string used for filtering/searching.
	FilterValue() string
}

// ItemRenderer is a function that renders a list item to a buffer line.
// It receives the item, whether it's selected, whether the list is focused,
// and the available width. It should render to the buffer at y=0.
type ItemRenderer[T ListItem] func(buf *SubBuffer, item T, selected bool, focused bool, width int)

// SelectableList is a generic scrollable list with selection.
// It can be used for file lists, branch selectors, commit selectors, etc.
type SelectableList[T ListItem] struct {
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
	onSelect  func(item T)    // Called when selection changes
	onConfirm func(item T)    // Called when Enter is pressed
	onChange  func(items []T) // Called when items change
}

// NewSelectableList creates a new selectable list with the given renderer.
func NewSelectableList[T ListItem](renderer ItemRenderer[T]) *SelectableList[T] {
	list := &SelectableList[T]{
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
func (l *SelectableList[T]) SetItems(items []T) {
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
func (l *SelectableList[T]) Items() []T {
	return l.items
}

// Selected returns the currently selected item.
func (l *SelectableList[T]) Selected() (T, bool) {
	var zero T
	if l.selected < 0 || l.selected >= len(l.items) {
		return zero, false
	}
	return l.items[l.selected], true
}

// SelectedIndex returns the index of the selected item.
func (l *SelectableList[T]) SelectedIndex() int {
	return l.selected
}

// SetSelectedIndex sets the selected index.
func (l *SelectableList[T]) SetSelectedIndex(index int) {
	if index < 0 {
		index = 0
	}
	if index >= len(l.items) {
		index = len(l.items) - 1
	}
	l.selected = index
	l.ensureVisible()
	l.Repaint() // Mark as dirty for compositor re-render
	if l.onSelect != nil && len(l.items) > 0 {
		l.onSelect(l.items[l.selected])
	}
}

// SelectByPredicate selects the first item matching the predicate.
func (l *SelectableList[T]) SelectByPredicate(pred func(T) bool) bool {
	for i, item := range l.items {
		if pred(item) {
			l.SetSelectedIndex(i)
			return true
		}
	}
	return false
}

// SetStyles sets the selection styles.
func (l *SelectableList[T]) SetStyles(selected, selectedUnfocused, normal lipgloss.Style) {
	l.selectedStyle = selected
	l.selectedUnfocusedStyle = selectedUnfocused
	l.normalStyle = normal
}

// OnSelect sets a callback for when selection changes.
func (l *SelectableList[T]) OnSelect(fn func(item T)) {
	l.onSelect = fn
}

// OnConfirm sets a callback for when Enter is pressed.
func (l *SelectableList[T]) OnConfirm(fn func(item T)) {
	l.onConfirm = fn
}

// OnChange sets a callback for when items change.
func (l *SelectableList[T]) OnChange(fn func(items []T)) {
	l.onChange = fn
}

// visibleCount returns the number of visible items.
func (l *SelectableList[T]) visibleCount() int {
	return l.bounds.Height
}

// ensureVisible ensures the selected item is visible.
func (l *SelectableList[T]) ensureVisible() {
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
func (l *SelectableList[T]) Render(buf *SubBuffer) {
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
func (l *SelectableList[T]) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
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
		if l.onConfirm != nil {
			if item, ok := l.Selected(); ok {
				l.onConfirm(item)
			}
		}
		return true, nil
	}
	return false, nil
}

func (l *SelectableList[T]) moveUp() {
	if l.selected > 0 {
		l.selected--
		l.ensureVisible()
		l.Repaint()
		if l.onSelect != nil {
			l.onSelect(l.items[l.selected])
		}
	}
}

func (l *SelectableList[T]) moveDown() {
	if l.selected < len(l.items)-1 {
		l.selected++
		l.ensureVisible()
		l.Repaint()
		if l.onSelect != nil {
			l.onSelect(l.items[l.selected])
		}
	}
}

func (l *SelectableList[T]) pageUp() {
	visible := l.visibleCount()
	newSelected := max(0, l.selected-visible)
	l.SetSelectedIndex(newSelected)
}

func (l *SelectableList[T]) pageDown() {
	visible := l.visibleCount()
	newSelected := min(len(l.items)-1, l.selected+visible)
	l.SetSelectedIndex(newSelected)
}

// ScrollView is a widget that provides scrolling for content larger than its bounds.
type ScrollView struct {
	BaseView
	content     View
	scrollX     int
	scrollY     int
	contentSize Size
}

// NewScrollView creates a new scroll view with the given content.
func NewScrollView(content View) *ScrollView {
	sv := &ScrollView{
		BaseView: NewBaseView(),
		content:  content,
	}
	if content != nil {
		content.SetParent(sv)
	}
	return sv
}

// SetContent sets the scroll view's content.
func (s *ScrollView) SetContent(w View) {
	if s.content != nil {
		s.content.SetParent(nil)
	}
	s.content = w
	if w != nil {
		w.SetParent(s)
	}
}

// SetContentSize sets the virtual content size.
func (s *ScrollView) SetContentSize(size Size) {
	s.contentSize = size
}

// SetScroll sets the scroll position.
func (s *ScrollView) SetScroll(x, y int) {
	s.scrollX = utils.Clamp(x, 0, s.contentSize.Width-s.bounds.Width)
	s.scrollY = utils.Clamp(y, 0, s.contentSize.Height-s.bounds.Height)
}

// ScrollTo ensures the given position is visible.
func (s *ScrollView) ScrollTo(x, y int) {
	// Scroll horizontally if needed
	if x < s.scrollX {
		s.scrollX = x
	} else if x >= s.scrollX+s.bounds.Width {
		s.scrollX = x - s.bounds.Width + 1
	}

	// Scroll vertically if needed
	if y < s.scrollY {
		s.scrollY = y
	} else if y >= s.scrollY+s.bounds.Height {
		s.scrollY = y - s.bounds.Height + 1
	}
}

// ScrollPosition returns the current scroll position.
func (s *ScrollView) ScrollPosition() (x, y int) {
	return s.scrollX, s.scrollY
}

// Children returns the content widget.
func (s *ScrollView) Children() []View {
	if s.content != nil {
		return []View{s.content}
	}
	return nil
}

// SetBounds sets the scroll view's bounds.
func (s *ScrollView) SetBounds(bounds Rect) {
	s.BaseView.SetBounds(bounds)
	if s.content != nil {
		// Content gets virtual bounds at the scroll offset
		s.content.SetBounds(Rect{
			Position{
				X: -s.scrollX,
				Y: -s.scrollY,
			},
			Size{
				Width:  s.contentSize.Width,
				Height: s.contentSize.Height,
			},
		})
	}
}

// Render renders the visible portion of the content.
func (s *ScrollView) Render(buf *SubBuffer) {
	if s.content == nil {
		return
	}

	// Render content with scroll offset applied
	// The content renders to its own buffer, then we blit the visible portion
	contentBuf := NewBuffer(s.contentSize.Width, s.contentSize.Height)
	contentSub := NewSubBuffer(contentBuf, contentBuf.Bounds())
	RenderWidget(s.content, contentSub)

	// Copy visible portion
	for y := 0; y < buf.Height(); y++ {
		srcY := s.scrollY + y
		if srcY < 0 || srcY >= s.contentSize.Height {
			continue
		}

		// Calculate the range of cells to copy for this row
		srcStartX := s.scrollX
		if srcStartX < 0 {
			srcStartX = 0
		}
		srcEndX := s.scrollX + buf.Width()
		if srcEndX > s.contentSize.Width {
			srcEndX = s.contentSize.Width
		}
		if srcStartX >= srcEndX {
			continue
		}

		// Build a row of cells to copy
		cells := make([]Cell, srcEndX-srcStartX)
		for i := 0; i < len(cells); i++ {
			cells[i] = contentBuf.GetCell(srcStartX+i, srcY)
		}

		destX := srcStartX - s.scrollX
		buf.SetCells(destX, y, cells)
	}
}

// HandleKey handles keyboard input for scrolling.
func (s *ScrollView) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "up":
		s.SetScroll(s.scrollX, s.scrollY-1)
		return true, nil
	case "down":
		s.SetScroll(s.scrollX, s.scrollY+1)
		return true, nil
	case "left":
		s.SetScroll(s.scrollX-1, s.scrollY)
		return true, nil
	case "right":
		s.SetScroll(s.scrollX+1, s.scrollY)
		return true, nil
	case "pgup":
		s.SetScroll(s.scrollX, s.scrollY-s.bounds.Height)
		return true, nil
	case "pgdown":
		s.SetScroll(s.scrollX, s.scrollY+s.bounds.Height)
		return true, nil
	case "home":
		s.SetScroll(s.scrollX, 0)
		return true, nil
	case "end":
		s.SetScroll(s.scrollX, s.contentSize.Height-s.bounds.Height)
		return true, nil
	}

	// Pass to content if not handled
	if s.content != nil {
		return s.content.HandleKey(msg)
	}
	return false, nil
}

package teapot

import (
	"testing"

	"github.com/radiospiel/critic/simple-go/assert"
	tea "github.com/charmbracelet/bubbletea"
)

// testItem implements ListItem for testing
type testItem struct {
	name string
	id   int
}

func (t testItem) FilterValue() string {
	return t.name
}

func TestSelectableListBasics(t *testing.T) {
	list := NewSelectableList[testItem](nil)

	items := []testItem{
		{name: "Apple", id: 1},
		{name: "Banana", id: 2},
		{name: "Cherry", id: 3},
	}
	list.SetItems(items)

	assert.Equals(t, len(list.Items()), 3)
	assert.Equals(t, list.SelectedIndex(), 0)

	item, ok := list.Selected()
	assert.True(t, ok, "should have a selected item")
	assert.Equals(t, item.name, "Apple")
}

func TestSelectableListNavigation(t *testing.T) {
	list := NewSelectableList[testItem](nil)
	list.SetBounds(NewRect(0, 0, 20, 10))

	items := []testItem{
		{name: "Apple", id: 1},
		{name: "Banana", id: 2},
		{name: "Cherry", id: 3},
	}
	list.SetItems(items)

	// Move down
	list.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equals(t, list.SelectedIndex(), 1)

	list.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equals(t, list.SelectedIndex(), 2)

	// Move up
	list.HandleKey(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equals(t, list.SelectedIndex(), 1)

	list.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.Equals(t, list.SelectedIndex(), 0)

	// Can't go above 0
	list.HandleKey(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equals(t, list.SelectedIndex(), 0)
}

func TestSelectableListHomeEnd(t *testing.T) {
	list := NewSelectableList[testItem](nil)
	list.SetBounds(NewRect(0, 0, 20, 10))

	items := []testItem{
		{name: "Apple", id: 1},
		{name: "Banana", id: 2},
		{name: "Cherry", id: 3},
		{name: "Date", id: 4},
		{name: "Elderberry", id: 5},
	}
	list.SetItems(items)

	// Go to end (G)
	list.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	assert.Equals(t, list.SelectedIndex(), 4)

	// Go to home (g)
	list.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	assert.Equals(t, list.SelectedIndex(), 0)
}

func TestSelectableListSetSelectedIndex(t *testing.T) {
	list := NewSelectableList[testItem](nil)

	items := []testItem{
		{name: "Apple", id: 1},
		{name: "Banana", id: 2},
		{name: "Cherry", id: 3},
	}
	list.SetItems(items)

	list.SetSelectedIndex(1)
	assert.Equals(t, list.SelectedIndex(), 1)

	// Out of bounds should clamp
	list.SetSelectedIndex(100)
	assert.Equals(t, list.SelectedIndex(), 2)

	list.SetSelectedIndex(-5)
	assert.Equals(t, list.SelectedIndex(), 0)
}

func TestSelectableListRender(t *testing.T) {
	// Custom renderer
	renderer := func(buf *SubBuffer, item testItem, selected bool, focused bool, width int) {
		prefix := "  "
		if selected {
			prefix = "> "
		}
		buf.SetString(0, 0, prefix+item.name, EmptyCell.Style)
	}

	list := NewSelectableList[testItem](renderer)
	list.SetBounds(NewRect(0, 0, 20, 5))
	list.SetFocused(true)

	items := []testItem{
		{name: "Apple", id: 1},
		{name: "Banana", id: 2},
		{name: "Cherry", id: 3},
	}
	list.SetItems(items)

	buf := NewBuffer(20, 5)
	sub := NewSubBuffer(buf, buf.Bounds())
	list.Render(sub)

	// First item should have selection prefix
	assert.Equals(t, buf.GetCell(0, 0).Rune, '>')
	assert.Equals(t, buf.GetCell(2, 0).Rune, 'A')

	// Second item should have regular prefix
	assert.Equals(t, buf.GetCell(0, 1).Rune, ' ')
	assert.Equals(t, buf.GetCell(2, 1).Rune, 'B')
}

func TestSelectableListScrolling(t *testing.T) {
	list := NewSelectableList[testItem](nil)
	list.SetBounds(NewRect(0, 0, 20, 3)) // Only 3 visible items

	items := []testItem{
		{name: "Apple", id: 1},
		{name: "Banana", id: 2},
		{name: "Cherry", id: 3},
		{name: "Date", id: 4},
		{name: "Elderberry", id: 5},
	}
	list.SetItems(items)

	// Initially at top
	assert.Equals(t, list.SelectedIndex(), 0)

	// Move down past visible area
	list.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	list.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	list.HandleKey(tea.KeyMsg{Type: tea.KeyDown})

	// Should now be at index 3 (Date)
	assert.Equals(t, list.SelectedIndex(), 3)
}

func TestSelectableListEmptyList(t *testing.T) {
	list := NewSelectableList[testItem](nil)
	list.SetBounds(NewRect(0, 0, 20, 5))

	// Empty list operations should not panic
	list.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	list.HandleKey(tea.KeyMsg{Type: tea.KeyUp})

	item, ok := list.Selected()
	assert.False(t, ok, "empty list should have no selection")
	assert.Equals(t, item.name, "")
}

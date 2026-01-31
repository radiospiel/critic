package tui

import pot "github.com/radiospiel/critic/teapot"

// Pane represents which pane is currently focused
type Pane int

const (
	FileListPane Pane = iota
	DiffViewPane
)

// FocusManager manages focus traversal among a set of focusable views.
type FocusManager struct {
	children     []pot.View // Child views in focus order
	focusedIndex int        // Index of currently focused child
	focusedPane  Pane       // Legacy pane tracking for compatibility
}

// NewFocusManager creates a new focus manager with the given children.
// The first focusable child will be focused initially.
func NewFocusManager(children ...pot.View) *FocusManager {
	fm := &FocusManager{
		children:     children,
		focusedIndex: -1,
		focusedPane:  FileListPane,
	}

	// Find first focusable child
	for i, child := range children {
		if child.AcceptsFocus() {
			fm.focusedIndex = i
			child.SetFocused(true)
			break
		}
	}

	return fm
}

// GetFocusedPane returns which pane is currently focused.
func (fm *FocusManager) GetFocusedPane() Pane {
	return fm.focusedPane
}

// SetFocusedPane sets focus to a specific pane.
func (fm *FocusManager) SetFocusedPane(pane Pane) {
	index := int(pane)
	if index < 0 || index >= len(fm.children) {
		return
	}
	if !fm.children[index].AcceptsFocus() {
		return
	}

	// Clear old focus
	if fm.focusedIndex >= 0 && fm.focusedIndex < len(fm.children) {
		fm.children[fm.focusedIndex].SetFocused(false)
	}

	// Set new focus
	fm.focusedIndex = index
	fm.focusedPane = pane
	fm.children[index].SetFocused(true)
}

// FocusNext moves focus to the next focusable child.
// Returns true if focus was moved, false if already at the last focusable child.
func (fm *FocusManager) FocusNext() bool {
	return fm.moveFocus(1)
}

// FocusPrev moves focus to the previous focusable child.
// Returns true if focus was moved, false if already at the first focusable child.
func (fm *FocusManager) FocusPrev() bool {
	return fm.moveFocus(-1)
}

// moveFocus moves focus by the given delta (positive for next, negative for prev).
// Returns false if there is no next/previous focusable child in the requested direction.
func (fm *FocusManager) moveFocus(delta int) bool {
	if len(fm.children) == 0 {
		return false
	}

	// Build listView of focusable children indices
	var focusableIndices []int
	for i, child := range fm.children {
		if child.AcceptsFocus() {
			focusableIndices = append(focusableIndices, i)
		}
	}

	// Need at least 2 focusable children to move
	if len(focusableIndices) < 2 {
		return false
	}

	// Find current position in focusable listView
	currentPos := -1
	for i, idx := range focusableIndices {
		if idx == fm.focusedIndex {
			currentPos = i
			break
		}
	}

	// Calculate new position, return false if out of bounds
	newPos := currentPos + delta
	if newPos < 0 || newPos >= len(focusableIndices) {
		return false
	}
	newIndex := focusableIndices[newPos]

	// Update focus state
	if fm.focusedIndex >= 0 && fm.focusedIndex < len(fm.children) {
		fm.children[fm.focusedIndex].SetFocused(false)
	}
	fm.focusedIndex = newIndex
	fm.children[fm.focusedIndex].SetFocused(true)

	// Update focusedPane for compatibility
	fm.focusedPane = Pane(fm.focusedIndex)

	return true
}

// GetFocusedChild returns the currently focused child view, or nil if none.
func (fm *FocusManager) GetFocusedChild() pot.View {
	if fm.focusedIndex >= 0 && fm.focusedIndex < len(fm.children) {
		return fm.children[fm.focusedIndex]
	}
	return nil
}

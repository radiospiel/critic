package teapot

import (
	"github.com/radiospiel/critic/simple-go/logger"
	tea "github.com/charmbracelet/bubbletea"
)

// ModalKeyHandler handles keyboard input for modal overlays.
// When a modal is active, it captures all keyboard input before the normal focus chain.
type ModalKeyHandler interface {
	HandleKey(msg tea.KeyMsg) (handled bool, cmd tea.Cmd)
}

// FocusManager handles focus traversal within a widget tree.
type FocusManager struct {
	root       View
	focused    View
	focusChain []View
	modal      ModalKeyHandler // If set, captures all keyboard input
}

// NewFocusManager creates a new focus manager for the given widget tree.
func NewFocusManager(root View) *FocusManager {
	fm := &FocusManager{root: root}
	fm.rebuildFocusChain()
	return fm
}

// Focused returns the currently focused widget.
func (fm *FocusManager) Focused() View {
	return fm.focused
}

// SetFocused sets focus to the given widget.
func (fm *FocusManager) SetFocused(w View) {
	if fm.focused != nil {
		fm.focused.SetFocused(false)
	}
	fm.focused = w
	if w != nil {
		w.SetFocused(true)
	}
}

// SetModal sets a modal key handler that will capture all keyboard input.
// The modal handler receives keys before the normal focus chain.
func (fm *FocusManager) SetModal(m ModalKeyHandler) {
	fm.modal = m
}

// ClearModal removes the current modal key handler.
func (fm *FocusManager) ClearModal() {
	fm.modal = nil
}

// HasModal returns true if a modal handler is currently set.
func (fm *FocusManager) HasModal() bool {
	return fm.modal != nil
}

// FocusNext moves focus to the next focusable widget.
func (fm *FocusManager) FocusNext() {
	if len(fm.focusChain) == 0 {
		return
	}

	currentIdx := -1
	for i, w := range fm.focusChain {
		if w == fm.focused {
			currentIdx = i
			break
		}
	}

	nextIdx := (currentIdx + 1) % len(fm.focusChain)
	fm.SetFocused(fm.focusChain[nextIdx])
}

// FocusPrev moves focus to the previous focusable widget.
func (fm *FocusManager) FocusPrev() {
	if len(fm.focusChain) == 0 {
		return
	}

	currentIdx := -1
	for i, w := range fm.focusChain {
		if w == fm.focused {
			currentIdx = i
			break
		}
	}

	prevIdx := currentIdx - 1
	if prevIdx < 0 {
		prevIdx = len(fm.focusChain) - 1
	}
	fm.SetFocused(fm.focusChain[prevIdx])
}

// RebuildFocusChain rebuilds the list of focusable widgets.
// Call this after adding/removing widgets.
func (fm *FocusManager) RebuildFocusChain() {
	fm.rebuildFocusChain()
}

func (fm *FocusManager) rebuildFocusChain() {
	fm.focusChain = nil
	fm.collectFocusable(fm.root)
}

func (fm *FocusManager) collectFocusable(w View) {
	if w == nil {
		return
	}

	if w.AcceptsFocus() {
		logger.Info("*** collectFocusable")
		fm.focusChain = append(fm.focusChain, w)
	}

	for _, child := range w.Children() {
		fm.collectFocusable(child)
	}
}

// HandleKey routes a key event through the focus system.
// Returns true if the event was handled.
// If a modal is active, all keys are routed to it first.
func (fm *FocusManager) HandleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	// Modal captures all keys when active
	if fm.modal != nil {
		return fm.modal.HandleKey(msg)
	}

	// Tab/Shift+Tab for focus navigation
	switch msg.String() {
	case "tab":
		fm.FocusNext()
		return true, nil
	case "shift+tab":
		fm.FocusPrev()
		return true, nil
	}

	// Route to focused widget
	if fm.focused != nil {
		return fm.focused.HandleKey(msg)
	}

	return false, nil
}

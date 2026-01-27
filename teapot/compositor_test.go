package teapot

import (
	"testing"

	"github.com/radiospiel/critic/simple-go/assert"
	tea "github.com/charmbracelet/bubbletea"
)

func TestCompositorResize(t *testing.T) {
	root := newMockWidget("R", 0, 0, 1)
	comp := NewCompositor(root)

	comp.Resize(80, 24)

	w, h := comp.Size()
	assert.Equals(t, w, 80)
	assert.Equals(t, h, 24)

	// Root should have same size
	assert.Equals(t, root.Bounds().Width, 80)
	assert.Equals(t, root.Bounds().Height, 24)
}

func TestCompositorRender(t *testing.T) {
	root := newMockWidget("X", 0, 0, 1)
	comp := NewCompositor(root)
	comp.Resize(10, 5)

	output := comp.Render()
	// Should have some output
	assert.True(t, len(output) > 0, "render should produce output")
	assert.Contains(t, output, "X")
}

func TestCompositorDirtyFlag(t *testing.T) {
	root := newMockWidget("R", 0, 0, 1)
	comp := NewCompositor(root)
	comp.Resize(10, 5)

	assert.True(t, comp.IsDirty(), "should be dirty after resize")

	comp.Render()
	assert.False(t, comp.IsDirty(), "should not be dirty after render")

	comp.MarkDirty()
	assert.True(t, comp.IsDirty(), "should be dirty after MarkDirty")
}

// TestCompositorFocusManagement removed - focus management is now handled by FocusManager

func TestFocusManager(t *testing.T) {
	widget1 := newMockWidget("A", 0, 0, 0)
	widget2 := newMockWidget("B", 0, 0, 0)
	widget3 := newMockWidget("C", 0, 0, 0)

	vbox := NewVBox(0)
	vbox.AddChild(widget1)
	vbox.AddChild(widget2)
	vbox.AddChild(widget3)

	fm := NewFocusManager(vbox)

	// Set initial focus
	fm.SetFocused(widget1)
	assert.Equals(t, fm.Focused(), View(widget1))

	// Focus next
	fm.FocusNext()
	assert.Equals(t, fm.Focused(), View(widget2))

	fm.FocusNext()
	assert.Equals(t, fm.Focused(), View(widget3))

	// Wrap around
	fm.FocusNext()
	assert.Equals(t, fm.Focused(), View(widget1))

	// Focus prev
	fm.FocusPrev()
	assert.Equals(t, fm.Focused(), View(widget3))
}

func TestStackPushPop(t *testing.T) {
	stack := NewStack()

	widget1 := newMockWidget("A", 0, 0, 0)
	widget2 := newMockWidget("B", 0, 0, 0)

	stack.Push(widget1)
	assert.Equals(t, stack.Top(), View(widget1))
	assert.Equals(t, stack.Base(), View(widget1))

	stack.Push(widget2)
	assert.Equals(t, stack.Top(), View(widget2))
	assert.Equals(t, stack.Base(), View(widget1))

	popped := stack.Pop()
	assert.Equals(t, popped, View(widget2))
	assert.Equals(t, stack.Top(), View(widget1))
}

func TestStackRenderOrder(t *testing.T) {
	stack := NewStack()
	stack.SetBounds(NewRect(0, 0, 10, 5))

	// View that fills with 'A'
	widgetA := newMockWidget("A", 0, 0, 0)
	// View that fills with 'B' (will be on top)
	widgetB := newMockWidget("B", 0, 0, 0)

	stack.Push(widgetA)
	stack.Push(widgetB)

	buf := NewBuffer(10, 5)
	sub := NewSubBuffer(buf, buf.Bounds())
	stack.Render(sub)

	// B should be on top, overwriting A
	assert.Equals(t, buf.GetCell(0, 0).Rune, 'B')
	assert.Equals(t, buf.GetCell(5, 2).Rune, 'B')
}

func TestStackKeyEventRouting(t *testing.T) {
	stack := NewStack()

	var handledBy string

	// View A - handles 'a' key
	widgetA := NewCallbackView(nil)
	widgetA.SetKeyFunc(func(msg tea.KeyMsg) (bool, tea.Cmd) {
		if msg.String() == "a" {
			handledBy = "A"
			return true, nil
		}
		return false, nil
	})

	// View B - handles 'b' key
	widgetB := NewCallbackView(nil)
	widgetB.SetKeyFunc(func(msg tea.KeyMsg) (bool, tea.Cmd) {
		if msg.String() == "b" {
			handledBy = "B"
			return true, nil
		}
		return false, nil
	})

	stack.Push(widgetA)
	stack.Push(widgetB)

	// B is on top, should get events first
	stack.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	assert.Equals(t, handledBy, "B")

	// A doesn't handle 'b', but B does
	handledBy = ""
	stack.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	// B doesn't handle 'a', falls through to A
	assert.Equals(t, handledBy, "A")
}

func TestCompositorUpdate(t *testing.T) {
	root := newMockWidget("R", 0, 0, 1)
	comp := NewCompositor(root)

	// Handle resize message
	cmd := comp.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	assert.Nil(t, cmd)

	w, h := comp.Size()
	assert.Equals(t, w, 100)
	assert.Equals(t, h, 50)
}

func TestCompositorModel(t *testing.T) {
	root := newMockWidget("R", 0, 0, 1)
	model := NewCompositorModel(root)

	// Init returns a tick command for animation (ticking is enabled by default)
	cmd := model.Init()
	assert.NotNil(t, cmd, "Init should return tick command when ticking is enabled")

	// Update with resize
	newModel, cmd := model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	assert.Nil(t, cmd)

	// View should return something
	view := newModel.(CompositorModel).View()
	assert.True(t, len(view) > 0, "view should produce output")

	// Quit on q
	_, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	assert.NotNil(t, cmd, "q should produce quit command")
}

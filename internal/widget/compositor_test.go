package widget

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"git.15b.it/eno/critic/internal/assert"
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

func TestCompositorFocusManagement(t *testing.T) {
	widget1 := newMockWidget("A", 0, 0, 0)
	widget2 := newMockWidget("B", 0, 0, 0)

	vbox := NewVBox(0)
	vbox.AddChild(widget1)
	vbox.AddChild(widget2)

	comp := NewCompositor(vbox)
	comp.Resize(80, 24)

	// Initially no focus
	assert.Nil(t, comp.Focused())

	// Set focus
	comp.SetFocused(widget1)
	assert.Equals(t, comp.Focused(), Widget(widget1))
	assert.True(t, widget1.Focused(), "widget1 should be focused")

	// Switch focus
	comp.SetFocused(widget2)
	assert.Equals(t, comp.Focused(), Widget(widget2))
	assert.False(t, widget1.Focused(), "widget1 should not be focused")
	assert.True(t, widget2.Focused(), "widget2 should be focused")
}

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
	assert.Equals(t, fm.Focused(), Widget(widget1))

	// Focus next
	fm.FocusNext()
	assert.Equals(t, fm.Focused(), Widget(widget2))

	fm.FocusNext()
	assert.Equals(t, fm.Focused(), Widget(widget3))

	// Wrap around
	fm.FocusNext()
	assert.Equals(t, fm.Focused(), Widget(widget1))

	// Focus prev
	fm.FocusPrev()
	assert.Equals(t, fm.Focused(), Widget(widget3))
}

func TestStackPushPop(t *testing.T) {
	stack := NewStack()

	widget1 := newMockWidget("A", 0, 0, 0)
	widget2 := newMockWidget("B", 0, 0, 0)

	stack.Push(widget1)
	assert.Equals(t, stack.Top(), Widget(widget1))
	assert.Equals(t, stack.Base(), Widget(widget1))

	stack.Push(widget2)
	assert.Equals(t, stack.Top(), Widget(widget2))
	assert.Equals(t, stack.Base(), Widget(widget1))

	popped := stack.Pop()
	assert.Equals(t, popped, Widget(widget2))
	assert.Equals(t, stack.Top(), Widget(widget1))
}

func TestStackRenderOrder(t *testing.T) {
	stack := NewStack()
	stack.SetBounds(NewRect(0, 0, 10, 5))

	// Widget that fills with 'A'
	widgetA := newMockWidget("A", 0, 0, 0)
	// Widget that fills with 'B' (will be on top)
	widgetB := newMockWidget("B", 0, 0, 0)

	stack.Push(widgetA)
	stack.Push(widgetB)

	buf := NewBuffer(10, 5)
	sub := buf.Sub(buf.Bounds())
	stack.Render(sub)

	// B should be on top, overwriting A
	assert.Equals(t, buf.GetCell(0, 0).Rune, 'B')
	assert.Equals(t, buf.GetCell(5, 2).Rune, 'B')
}

func TestStackKeyEventRouting(t *testing.T) {
	stack := NewStack()

	var handledBy string

	// Widget A - handles 'a' key
	widgetA := NewCallbackWidget(nil)
	widgetA.SetKeyFunc(func(msg tea.KeyMsg) (bool, tea.Cmd) {
		if msg.String() == "a" {
			handledBy = "A"
			return true, nil
		}
		return false, nil
	})

	// Widget B - handles 'b' key
	widgetB := NewCallbackWidget(nil)
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

	// Init should return nil
	cmd := model.Init()
	assert.Nil(t, cmd)

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

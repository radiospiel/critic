package teapot

import (
	"testing"

	"git.15b.it/eno/critic/simple-go/assert"
)

// mockWidget is a simple widget for testing layouts
type mockWidget struct {
	BaseWidget
	id string
}

func newMockWidget(id string, minW, minH, stretch int) *mockWidget {
	w := &mockWidget{
		BaseWidget: NewBaseWidget(ZOrderDefault),
		id:         id,
	}
	w.SetConstraints(DefaultConstraints().WithMinSize(minW, minH).WithStretch(stretch, stretch))
	return w
}

func (m *mockWidget) Render(buf *SubBuffer) {
	// Fill with first char of id for visual debugging
	if len(m.id) > 0 {
		r := rune(m.id[0])
		row := make([]Cell, buf.Width())
		for i := range row {
			row[i] = Cell{Rune: r}
		}
		for y := 0; y < buf.Height(); y++ {
			buf.SetCells(0, y, row)
		}
	}
}

func TestVBoxLayout(t *testing.T) {
	vbox := NewVBox(0)
	vbox.SetBounds(NewRect(0, 0, 100, 100))

	// Add two children with equal stretch
	child1 := newMockWidget("A", 0, 0, 1)
	child2 := newMockWidget("B", 0, 0, 1)

	vbox.AddChild(child1)
	vbox.AddChild(child2)

	// Trigger layout
	vbox.SetBounds(NewRect(0, 0, 100, 100))

	// Each child should get half the height
	assert.Equals(t, child1.Bounds().Height, 50)
	assert.Equals(t, child2.Bounds().Height, 50)
	assert.Equals(t, child1.Bounds().Width, 100)
	assert.Equals(t, child2.Bounds().Width, 100)

	// Y positions should be stacked
	assert.Equals(t, child1.Bounds().Y, 0)
	assert.Equals(t, child2.Bounds().Y, 50)
}

func TestVBoxWithMinSize(t *testing.T) {
	vbox := NewVBox(0)

	child1 := newMockWidget("A", 0, 30, 0)  // Fixed 30px height
	child2 := newMockWidget("B", 0, 0, 1)   // Takes remaining space

	vbox.AddChild(child1)
	vbox.AddChild(child2)
	vbox.SetBounds(NewRect(0, 0, 100, 100))

	assert.Equals(t, child1.Bounds().Height, 30)
	assert.Equals(t, child2.Bounds().Height, 70)
}

func TestVBoxWithSpacing(t *testing.T) {
	vbox := NewVBox(10) // 10px spacing

	child1 := newMockWidget("A", 0, 0, 1)
	child2 := newMockWidget("B", 0, 0, 1)

	vbox.AddChild(child1)
	vbox.AddChild(child2)
	vbox.SetBounds(NewRect(0, 0, 100, 100))

	// Available: 100 - 10 (spacing) = 90, divided by 2 = 45 each
	assert.Equals(t, child1.Bounds().Height, 45)
	assert.Equals(t, child2.Bounds().Height, 45)
	assert.Equals(t, child1.Bounds().Y, 0)
	assert.Equals(t, child2.Bounds().Y, 55) // 45 + 10 spacing
}

func TestHBoxLayout(t *testing.T) {
	hbox := NewHBox(0)

	child1 := newMockWidget("A", 0, 0, 1)
	child2 := newMockWidget("B", 0, 0, 1)

	hbox.AddChild(child1)
	hbox.AddChild(child2)
	hbox.SetBounds(NewRect(0, 0, 100, 50))

	// Each child should get half the width
	assert.Equals(t, child1.Bounds().Width, 50)
	assert.Equals(t, child2.Bounds().Width, 50)
	assert.Equals(t, child1.Bounds().Height, 50)
	assert.Equals(t, child2.Bounds().Height, 50)

	// X positions should be side by side
	assert.Equals(t, child1.Bounds().X, 0)
	assert.Equals(t, child2.Bounds().X, 50)
}

func TestHBoxWithDifferentStretch(t *testing.T) {
	hbox := NewHBox(0)

	child1 := newMockWidget("A", 0, 0, 1) // stretch 1
	child2 := newMockWidget("B", 0, 0, 2) // stretch 2

	hbox.AddChild(child1)
	hbox.AddChild(child2)
	hbox.SetBounds(NewRect(0, 0, 90, 50))

	// child1 gets 1/3, child2 gets 2/3
	assert.Equals(t, child1.Bounds().Width, 30)
	assert.Equals(t, child2.Bounds().Width, 60)
}

func TestHSplit(t *testing.T) {
	left := newMockWidget("L", 0, 0, 0)
	right := newMockWidget("R", 0, 0, 0)

	split := NewHSplit(left, right, 0.3) // 30% left, 70% right
	split.SetBounds(NewRect(0, 0, 101, 50)) // 101 - 1 divider = 100 available

	// 30% of 100 = 30
	assert.Equals(t, left.Bounds().Width, 30)
	assert.Equals(t, right.Bounds().Width, 70)

	// Positions
	assert.Equals(t, left.Bounds().X, 0)
	assert.Equals(t, right.Bounds().X, 31) // 30 + 1 divider
}

func TestVSplit(t *testing.T) {
	top := newMockWidget("T", 0, 0, 0)
	bottom := newMockWidget("B", 0, 0, 0)

	split := NewVSplit(top, bottom, 0.4) // 40% top, 60% bottom
	split.SetBounds(NewRect(0, 0, 50, 101)) // 101 - 1 divider = 100 available

	// 40% of 100 = 40
	assert.Equals(t, top.Bounds().Height, 40)
	assert.Equals(t, bottom.Bounds().Height, 60)

	// Positions
	assert.Equals(t, top.Bounds().Y, 0)
	assert.Equals(t, bottom.Bounds().Y, 41) // 40 + 1 divider
}

func TestSplitWithFixedSize(t *testing.T) {
	left := newMockWidget("L", 0, 0, 0)
	right := newMockWidget("R", 0, 0, 0)

	split := NewHSplit(left, right, 0.5)
	split.SetFixedSize(25) // Fixed 25px for left pane
	split.SetBounds(NewRect(0, 0, 101, 50))

	assert.Equals(t, left.Bounds().Width, 25)
	assert.Equals(t, right.Bounds().Width, 75)
}

func TestSplitRender(t *testing.T) {
	left := newMockWidget("L", 0, 0, 0)
	right := newMockWidget("R", 0, 0, 0)

	split := NewHSplit(left, right, 0.5)
	split.SetBounds(NewRect(0, 0, 21, 3)) // 21 - 1 divider = 20, split 10|10

	buf := NewBuffer(21, 3)
	sub := buf.Sub(buf.Bounds())
	split.Render(sub)

	// Left side should have 'L'
	assert.Equals(t, buf.GetCell(0, 0).Rune, 'L')
	assert.Equals(t, buf.GetCell(9, 0).Rune, 'L')

	// Divider at position 10
	assert.Equals(t, buf.GetCell(10, 0).Rune, '│')
	assert.Equals(t, buf.GetCell(10, 1).Rune, '│')
	assert.Equals(t, buf.GetCell(10, 2).Rune, '│')

	// Right side should have 'R'
	assert.Equals(t, buf.GetCell(11, 0).Rune, 'R')
	assert.Equals(t, buf.GetCell(20, 0).Rune, 'R')
}

func TestContainerWidgetChildren(t *testing.T) {
	container := NewContainerWidget()

	child1 := newMockWidget("A", 0, 0, 0)
	child2 := newMockWidget("B", 0, 0, 0)

	container.AddChild(child1)
	container.AddChild(child2)

	assert.Equals(t, len(container.Children()), 2)
	assert.Equals(t, child1.Parent(), Widget(&container))
	assert.Equals(t, child2.Parent(), Widget(&container))

	container.RemoveChild(child1)
	assert.Equals(t, len(container.Children()), 1)
	assert.Nil(t, child1.Parent())

	container.ClearChildren()
	assert.Equals(t, len(container.Children()), 0)
	assert.Nil(t, child2.Parent())
}

package teapot

import (
	"testing"

	"git.15b.it/eno/critic/simple-go/assert"
	"github.com/charmbracelet/lipgloss"
)

func TestBufferBasics(t *testing.T) {
	buf := NewBuffer(10, 5)

	assert.Equals(t, buf.Width(), 10)
	assert.Equals(t, buf.Height(), 5)

	// Initially all cells should be spaces
	cell := buf.GetCell(0, 0)
	assert.Equals(t, cell.Rune, ' ')
}

func TestBufferSetCells(t *testing.T) {
	buf := NewBuffer(10, 5)
	style := lipgloss.NewStyle().Bold(true)

	cells := []Cell{{Rune: 'X', Style: style}, {Rune: 'Y', Style: style}}
	buf.SetCells(3, 2, cells)
	assert.Equals(t, buf.GetCell(3, 2).Rune, 'X')
	assert.Equals(t, buf.GetCell(4, 2).Rune, 'Y')

	// Out of bounds writes should be clipped
	buf.SetCells(-1, 0, []Cell{{Rune: 'A'}, {Rune: 'B'}, {Rune: 'C'}})
	assert.Equals(t, buf.GetCell(0, 0).Rune, 'B')
	assert.Equals(t, buf.GetCell(1, 0).Rune, 'C')

	// Writes extending past buffer width should be clipped
	buf.SetCells(8, 0, []Cell{{Rune: 'P'}, {Rune: 'Q'}, {Rune: 'R'}})
	assert.Equals(t, buf.GetCell(8, 0).Rune, 'P')
	assert.Equals(t, buf.GetCell(9, 0).Rune, 'Q')
	// 'R' should not be written
}

func TestBufferSetString(t *testing.T) {
	buf := NewBuffer(10, 5)
	style := lipgloss.NewStyle()

	buf.SetString(2, 1, "Hello", style)

	assert.Equals(t, buf.GetCell(2, 1).Rune, 'H')
	assert.Equals(t, buf.GetCell(3, 1).Rune, 'e')
	assert.Equals(t, buf.GetCell(4, 1).Rune, 'l')
	assert.Equals(t, buf.GetCell(5, 1).Rune, 'l')
	assert.Equals(t, buf.GetCell(6, 1).Rune, 'o')
}

func TestBufferSetStringTruncated(t *testing.T) {
	buf := NewBuffer(10, 5)
	style := lipgloss.NewStyle()

	buf.SetStringTruncated(0, 0, "Hello World", 5, style)

	assert.Equals(t, buf.GetCell(0, 0).Rune, 'H')
	assert.Equals(t, buf.GetCell(1, 0).Rune, 'e')
	assert.Equals(t, buf.GetCell(2, 0).Rune, 'l')
	assert.Equals(t, buf.GetCell(3, 0).Rune, 'l')
	assert.Equals(t, buf.GetCell(4, 0).Rune, '…')
}

func TestBufferFill(t *testing.T) {
	buf := NewBuffer(10, 5)
	style := lipgloss.NewStyle()

	buf.Fill(NewRect(2, 1, 3, 2), Cell{Rune: '#', Style: style})

	assert.Equals(t, buf.GetCell(2, 1).Rune, '#')
	assert.Equals(t, buf.GetCell(3, 1).Rune, '#')
	assert.Equals(t, buf.GetCell(4, 1).Rune, '#')
	assert.Equals(t, buf.GetCell(2, 2).Rune, '#')
	assert.Equals(t, buf.GetCell(3, 2).Rune, '#')
	assert.Equals(t, buf.GetCell(4, 2).Rune, '#')

	// Outside the fill area
	assert.Equals(t, buf.GetCell(1, 1).Rune, ' ')
	assert.Equals(t, buf.GetCell(5, 1).Rune, ' ')
}

func TestBufferBlit(t *testing.T) {
	src := NewBuffer(3, 2)
	dst := NewBuffer(10, 5)

	style := lipgloss.NewStyle()
	src.SetString(0, 0, "ABC", style)
	src.SetString(0, 1, "XYZ", style)

	dst.Blit(src, 2, 1)

	assert.Equals(t, dst.GetCell(2, 1).Rune, 'A')
	assert.Equals(t, dst.GetCell(3, 1).Rune, 'B')
	assert.Equals(t, dst.GetCell(4, 1).Rune, 'C')
	assert.Equals(t, dst.GetCell(2, 2).Rune, 'X')
	assert.Equals(t, dst.GetCell(3, 2).Rune, 'Y')
	assert.Equals(t, dst.GetCell(4, 2).Rune, 'Z')
}

func TestSubBuffer(t *testing.T) {
	buf := NewBuffer(20, 10)
	sub := NewSubBuffer(buf, NewRect(5, 3, 8, 4))

	assert.Equals(t, sub.Width(), 8)
	assert.Equals(t, sub.Height(), 4)

	// Write to sub-buffer
	style := lipgloss.NewStyle()
	sub.SetString(0, 0, "Test", style)

	// Should appear in parent at offset
	assert.Equals(t, buf.GetCell(5, 3).Rune, 'T')
	assert.Equals(t, buf.GetCell(6, 3).Rune, 'e')
	assert.Equals(t, buf.GetCell(7, 3).Rune, 's')
	assert.Equals(t, buf.GetCell(8, 3).Rune, 't')
}

func TestBufferClone(t *testing.T) {
	buf := NewBuffer(5, 5)
	style := lipgloss.NewStyle()
	buf.SetString(0, 0, "Hello", style)

	clone := buf.Clone()

	// Clone should have same content
	assert.Equals(t, clone.GetCell(0, 0).Rune, 'H')
	assert.Equals(t, clone.GetCell(1, 0).Rune, 'e')

	// Modifying clone shouldn't affect original
	clone.SetString(0, 0, "X", style)
	assert.Equals(t, buf.GetCell(0, 0).Rune, 'H')
	assert.Equals(t, clone.GetCell(0, 0).Rune, 'X')
}

func TestBufferEquals(t *testing.T) {
	buf1 := NewBuffer(5, 5)
	buf2 := NewBuffer(5, 5)

	assert.True(t, buf1.Equals(buf2), "empty buffers should be equal")

	style := lipgloss.NewStyle()
	buf1.SetString(0, 0, "Test", style)
	assert.False(t, buf1.Equals(buf2), "different content should not be equal")

	buf2.SetString(0, 0, "Test", style)
	assert.True(t, buf1.Equals(buf2), "same content should be equal")
}

func TestBufferDrawBox(t *testing.T) {
	buf := NewBuffer(10, 5)
	style := lipgloss.NewStyle()

	buf.DrawBox(NewRect(1, 1, 5, 3), style)

	// Corners
	assert.Equals(t, buf.GetCell(1, 1).Rune, '┌')
	assert.Equals(t, buf.GetCell(5, 1).Rune, '┐')
	assert.Equals(t, buf.GetCell(1, 3).Rune, '└')
	assert.Equals(t, buf.GetCell(5, 3).Rune, '┘')

	// Edges
	assert.Equals(t, buf.GetCell(2, 1).Rune, '─')
	assert.Equals(t, buf.GetCell(1, 2).Rune, '│')
}

func TestBufferString(t *testing.T) {
	buf := NewBuffer(5, 2)
	style := lipgloss.NewStyle()

	buf.SetString(0, 0, "Hello", style)
	buf.SetString(0, 1, "World", style)

	output := buf.RenderToString()
	// The output should contain both lines
	assert.Contains(t, output, "H")
	assert.Contains(t, output, "W")
}

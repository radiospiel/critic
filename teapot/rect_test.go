package teapot

import (
	"testing"

	"git.15b.it/eno/critic/internal/assert"
)

func TestRectContains(t *testing.T) {
	r := NewRect(10, 20, 30, 40)

	// Inside
	assert.True(t, r.Contains(10, 20), "top-left corner should be inside")
	assert.True(t, r.Contains(25, 35), "center should be inside")
	assert.True(t, r.Contains(39, 59), "just before bottom-right should be inside")

	// Outside
	assert.False(t, r.Contains(9, 20), "left of rect should be outside")
	assert.False(t, r.Contains(40, 20), "right edge should be outside")
	assert.False(t, r.Contains(10, 60), "bottom edge should be outside")
}

func TestRectIntersect(t *testing.T) {
	r1 := NewRect(0, 0, 10, 10)
	r2 := NewRect(5, 5, 10, 10)

	inter := r1.Intersect(r2)
	assert.Equals(t, inter.X, 5)
	assert.Equals(t, inter.Y, 5)
	assert.Equals(t, inter.Width, 5)
	assert.Equals(t, inter.Height, 5)

	// No intersection
	r3 := NewRect(20, 20, 10, 10)
	noInter := r1.Intersect(r3)
	assert.True(t, noInter.IsEmpty(), "non-overlapping rects should have empty intersection")
}

func TestRectInset(t *testing.T) {
	r := NewRect(0, 0, 100, 50)
	inset := r.Inset(5, 10, 5, 10)

	assert.Equals(t, inset.X, 10)
	assert.Equals(t, inset.Y, 5)
	assert.Equals(t, inset.Width, 80)
	assert.Equals(t, inset.Height, 40)
}

func TestConstraints(t *testing.T) {
	c := DefaultConstraints().
		WithMinSize(10, 20).
		WithPreferredSize(50, 100).
		WithStretch(1, 2)

	assert.Equals(t, c.MinWidth, 10)
	assert.Equals(t, c.MinHeight, 20)
	assert.Equals(t, c.PreferredWidth, 50)
	assert.Equals(t, c.PreferredHeight, 100)
	assert.Equals(t, c.HorizontalStretch, 1)
	assert.Equals(t, c.VerticalStretch, 2)

	assert.Equals(t, c.EffectivePreferredWidth(), 50)
	assert.Equals(t, c.EffectivePreferredHeight(), 100)

	// Test fallback to min when preferred is 0
	c2 := DefaultConstraints().WithMinSize(15, 25)
	assert.Equals(t, c2.EffectivePreferredWidth(), 15)
	assert.Equals(t, c2.EffectivePreferredHeight(), 25)
}

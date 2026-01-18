package styleregistry

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"git.15b.it/eno/critic/simple-go/assert"
)

func TestNewRegistry(t *testing.T) {
	reg := New()
	assert.NotNil(t, reg)
	assert.Equals(t, reg.Len(), 0)
}

func TestNewRegistryWithCapacity(t *testing.T) {
	reg := NewWithCapacity(128)
	assert.NotNil(t, reg)
	assert.Equals(t, reg.Len(), 0)
}

func TestRegisterAndRender_EmptyStyle(t *testing.T) {
	reg := New()
	style := lipgloss.NewStyle()
	id := reg.Register(style)

	assert.Equals(t, reg.Len(), 1)

	// Empty style should return unmodified string
	result := reg.Render(id, "Hello")
	assert.Equals(t, result, "Hello")

	// Compiled style should be marked as empty
	cs := reg.Get(id)
	assert.True(t, cs.IsEmpty(), "empty style should be marked as empty")
	assert.True(t, cs.IsFastPath(), "empty style should use fast path")
}

func TestRegisterAndRender_BoldStyle(t *testing.T) {
	reg := New()
	style := lipgloss.NewStyle().Bold(true)
	id := reg.Register(style)

	result := reg.Render(id, "Hello")

	// Result should contain the text
	assert.Contains(t, result, "Hello")

	// Should match lipgloss output (may or may not have ANSI codes depending on TTY)
	expected := style.Render("Hello")
	assert.Equals(t, result, expected)
}

func TestRegisterAndRender_ForegroundColor(t *testing.T) {
	reg := New()
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
	id := reg.Register(style)

	result := reg.Render(id, "Red")

	// Should match lipgloss output
	expected := style.Render("Red")
	assert.Equals(t, result, expected)
}

func TestRegisterAndRender_BackgroundColor(t *testing.T) {
	reg := New()
	style := lipgloss.NewStyle().Background(lipgloss.Color("#00FF00"))
	id := reg.Register(style)

	result := reg.Render(id, "Green")

	// Should match lipgloss output
	expected := style.Render("Green")
	assert.Equals(t, result, expected)
}

func TestRegisterAndRender_ComplexStyle(t *testing.T) {
	reg := New()
	style := lipgloss.NewStyle().
		Bold(true).
		Italic(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4"))
	id := reg.Register(style)

	result := reg.Render(id, "Complex")

	// Should match lipgloss output
	expected := style.Render("Complex")
	assert.Equals(t, result, expected)
}

func TestRegisterAndRender_StyleWithPadding(t *testing.T) {
	reg := New()
	style := lipgloss.NewStyle().Padding(1, 2)
	id := reg.Register(style)

	result := reg.Render(id, "Padded")

	// Should match lipgloss output (uses slow path due to padding)
	expected := style.Render("Padded")
	assert.Equals(t, result, expected)

	// Should NOT use fast path due to padding
	cs := reg.Get(id)
	assert.False(t, cs.IsFastPath(), "style with padding should not use fast path")
}

func TestRegisterAndRender_StyleWithWidth(t *testing.T) {
	reg := New()
	style := lipgloss.NewStyle().Width(20)
	id := reg.Register(style)

	result := reg.Render(id, "Fixed")

	// Should match lipgloss output (uses slow path due to width)
	expected := style.Render("Fixed")
	assert.Equals(t, result, expected)

	// Should NOT use fast path due to width constraint
	cs := reg.Get(id)
	assert.False(t, cs.IsFastPath(), "style with width should not use fast path")
}

func TestRegisterAndRender_StyleWithBorder(t *testing.T) {
	reg := New()
	style := lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
	id := reg.Register(style)

	result := reg.Render(id, "Bordered")

	// Should match lipgloss output (uses slow path due to border)
	expected := style.Render("Bordered")
	assert.Equals(t, result, expected)

	// Should NOT use fast path due to border
	cs := reg.Get(id)
	assert.False(t, cs.IsFastPath(), "style with border should not use fast path")
}

func TestRegisterAndRender_StyleWithMargin(t *testing.T) {
	reg := New()
	style := lipgloss.NewStyle().Margin(1)
	id := reg.Register(style)

	result := reg.Render(id, "Margin")

	// Should match lipgloss output (uses slow path due to margin)
	expected := style.Render("Margin")
	assert.Equals(t, result, expected)

	// Should NOT use fast path due to margin
	cs := reg.Get(id)
	assert.False(t, cs.IsFastPath(), "style with margin should not use fast path")
}

func TestRender_InvalidID(t *testing.T) {
	reg := New()

	// Rendering with invalid ID should return unmodified string
	result := reg.Render(StyleID(999), "Test")
	assert.Equals(t, result, "Test")

	result = reg.Render(InvalidStyleID, "Test")
	assert.Equals(t, result, "Test")
}

func TestGet_InvalidID(t *testing.T) {
	reg := New()

	// Getting invalid ID should return nil
	cs := reg.Get(StyleID(999))
	assert.Nil(t, cs)

	cs = reg.Get(InvalidStyleID)
	assert.Nil(t, cs)
}

func TestMultipleStyles(t *testing.T) {
	reg := New()

	bold := lipgloss.NewStyle().Bold(true)
	italic := lipgloss.NewStyle().Italic(true)
	underline := lipgloss.NewStyle().Underline(true)

	boldID := reg.Register(bold)
	italicID := reg.Register(italic)
	underlineID := reg.Register(underline)

	assert.Equals(t, reg.Len(), 3)
	assert.Equals(t, boldID, StyleID(0))
	assert.Equals(t, italicID, StyleID(1))
	assert.Equals(t, underlineID, StyleID(2))

	// Each should render correctly
	assert.Equals(t, reg.Render(boldID, "B"), bold.Render("B"))
	assert.Equals(t, reg.Render(italicID, "I"), italic.Render("I"))
	assert.Equals(t, reg.Render(underlineID, "U"), underline.Render("U"))
}

func TestRenderBuilder(t *testing.T) {
	reg := New()
	style := lipgloss.NewStyle().Bold(true)
	id := reg.Register(style)

	var sb strings.Builder
	reg.RenderBuilder(id, "Hello", &sb)
	reg.RenderBuilder(id, " ", &sb)
	reg.RenderBuilder(id, "World", &sb)

	// Each part should be styled
	result := sb.String()
	expected := style.Render("Hello") + style.Render(" ") + style.Render("World")
	assert.Equals(t, result, expected)
}

func TestRenderBuilder_InvalidID(t *testing.T) {
	reg := New()

	var sb strings.Builder
	reg.RenderBuilder(StyleID(999), "Test", &sb)
	assert.Equals(t, sb.String(), "Test")
}

func TestCompiledStyle_Prefix(t *testing.T) {
	reg := New()
	style := lipgloss.NewStyle().Bold(true)
	id := reg.Register(style)

	cs := reg.Get(id)
	assert.NotNil(t, cs)

	// In TTY environment, prefix should contain ANSI escape
	// In non-TTY environment (like tests), it may be empty
	// The key is that prefix + content + suffix should equal lipgloss.Render()
	text := "Test"
	if cs.IsFastPath() {
		reconstructed := cs.Prefix() + text + cs.Suffix()
		expected := style.Render(text)
		assert.Equals(t, reconstructed, expected, "fast path reconstruction should match lipgloss output")
	}
}

func TestCompiledStyle_Suffix(t *testing.T) {
	reg := New()
	style := lipgloss.NewStyle().Bold(true)
	id := reg.Register(style)

	cs := reg.Get(id)
	assert.NotNil(t, cs)

	// In TTY environment, suffix should contain ANSI reset
	// In non-TTY environment (like tests), it may be empty
	// The key is that prefix + content + suffix should equal lipgloss.Render()
	text := "Test"
	if cs.IsFastPath() {
		reconstructed := cs.Prefix() + text + cs.Suffix()
		expected := style.Render(text)
		assert.Equals(t, reconstructed, expected, "fast path reconstruction should match lipgloss output")
	}
}

func TestFastPath_SimpleStyles(t *testing.T) {
	reg := New()

	testCases := []struct {
		name     string
		style    lipgloss.Style
		fastPath bool
	}{
		{"empty", lipgloss.NewStyle(), true},
		{"bold", lipgloss.NewStyle().Bold(true), true},
		{"italic", lipgloss.NewStyle().Italic(true), true},
		{"underline", lipgloss.NewStyle().Underline(true), true},
		{"foreground", lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")), true},
		{"background", lipgloss.NewStyle().Background(lipgloss.Color("#00FF00")), true},
		{"combined", lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF0000")), true},
		{"with padding", lipgloss.NewStyle().Padding(1), false},
		{"with width", lipgloss.NewStyle().Width(10), false},
		{"with border", lipgloss.NewStyle().Border(lipgloss.NormalBorder()), false},
		{"with margin", lipgloss.NewStyle().Margin(1), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			id := reg.Register(tc.style)
			cs := reg.Get(id)
			assert.Equals(t, cs.IsFastPath(), tc.fastPath, "style %q fastPath", tc.name)
		})
	}
}

func TestAdaptiveColor(t *testing.T) {
	reg := New()
	style := lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#FFFFFF"})
	id := reg.Register(style)

	result := reg.Render(id, "Adaptive")

	// Should match lipgloss output
	expected := style.Render("Adaptive")
	assert.Equals(t, result, expected)
}

func TestReverseStyle(t *testing.T) {
	reg := New()
	style := lipgloss.NewStyle().Reverse(true)
	id := reg.Register(style)

	result := reg.Render(id, "Reversed")

	// Should match lipgloss output
	expected := style.Render("Reversed")
	assert.Equals(t, result, expected)
}

func TestFaintStyle(t *testing.T) {
	reg := New()
	style := lipgloss.NewStyle().Faint(true)
	id := reg.Register(style)

	result := reg.Render(id, "Faint")

	// Should match lipgloss output
	expected := style.Render("Faint")
	assert.Equals(t, result, expected)
}

func TestBlinkStyle(t *testing.T) {
	reg := New()
	style := lipgloss.NewStyle().Blink(true)
	id := reg.Register(style)

	result := reg.Render(id, "Blink")

	// Should match lipgloss output
	expected := style.Render("Blink")
	assert.Equals(t, result, expected)
}

func TestStrikethroughStyle(t *testing.T) {
	reg := New()
	style := lipgloss.NewStyle().Strikethrough(true)
	id := reg.Register(style)

	result := reg.Render(id, "Strike")

	// Should match lipgloss output
	expected := style.Render("Strike")
	assert.Equals(t, result, expected)
}

func TestDeduplication_IdenticalStyles(t *testing.T) {
	reg := New()

	// Register the same style twice
	style1 := lipgloss.NewStyle().Bold(true)
	style2 := lipgloss.NewStyle().Bold(true)

	id1 := reg.Register(style1)
	id2 := reg.Register(style2)

	// Should return the same ID (deduplicated)
	assert.Equals(t, id1, id2, "identical styles should return same ID")
	assert.Equals(t, reg.Len(), 1, "registry should only have 1 style")
}

func TestDeduplication_DifferentStyles(t *testing.T) {
	reg := New()

	style1 := lipgloss.NewStyle().Bold(true)
	style2 := lipgloss.NewStyle().Italic(true)

	id1 := reg.Register(style1)
	id2 := reg.Register(style2)

	// Should return different IDs
	assert.NotEquals(t, id1, id2, "different styles should return different IDs")
	assert.Equals(t, reg.Len(), 2, "registry should have 2 styles")
}

func TestDeduplication_ComplexStyles(t *testing.T) {
	reg := New()

	// Two complex styles that are identical
	style1 := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF0000")).
		Background(lipgloss.Color("#00FF00"))
	style2 := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF0000")).
		Background(lipgloss.Color("#00FF00"))

	id1 := reg.Register(style1)
	id2 := reg.Register(style2)

	assert.Equals(t, id1, id2, "identical complex styles should return same ID")
	assert.Equals(t, reg.Len(), 1, "registry should only have 1 style")
}

func TestDeduplication_MultipleRegistrations(t *testing.T) {
	reg := New()

	bold := lipgloss.NewStyle().Bold(true)
	italic := lipgloss.NewStyle().Italic(true)

	// Register multiple times in various orders
	boldID1 := reg.Register(bold)
	italicID1 := reg.Register(italic)
	boldID2 := reg.Register(bold)
	italicID2 := reg.Register(italic)
	boldID3 := reg.Register(bold)

	assert.Equals(t, boldID1, boldID2, "bold IDs should match")
	assert.Equals(t, boldID2, boldID3, "bold IDs should match")
	assert.Equals(t, italicID1, italicID2, "italic IDs should match")
	assert.Equals(t, reg.Len(), 2, "registry should only have 2 unique styles")
}

func TestMaxCapacity(t *testing.T) {
	reg := New()

	// Register MaxStyles unique styles
	for i := 0; i < MaxStyles; i++ {
		// Create unique styles by using different colors (format as hex)
		color := fmt.Sprintf("#%06X", i)
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
		id := reg.Register(style)
		assert.NotEquals(t, id, InvalidStyleID, "style %d should register successfully", i)
	}

	assert.Equals(t, reg.Len(), MaxStyles, "registry should be at max capacity")

	// Try to register one more - should return InvalidStyleID
	extraStyle := lipgloss.NewStyle().Background(lipgloss.Color("#FFFFFF"))
	id := reg.Register(extraStyle)
	assert.Equals(t, id, InvalidStyleID, "should return InvalidStyleID when full")

	// Length should not change
	assert.Equals(t, reg.Len(), MaxStyles, "registry length should not exceed max")
}

func TestMaxCapacity_WithDeduplication(t *testing.T) {
	reg := New()

	// Fill up the registry
	for i := 0; i < MaxStyles; i++ {
		color := fmt.Sprintf("#%06X", i)
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
		reg.Register(style)
	}

	// Registering a duplicate should still work (returns existing ID)
	existingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#000000"))
	id := reg.Register(existingStyle)
	assert.Equals(t, id, StyleID(0), "duplicate should return existing ID even when full")
}

func TestMustRegister_Panic(t *testing.T) {
	reg := New()

	// Fill up the registry
	for i := 0; i < MaxStyles; i++ {
		color := fmt.Sprintf("#%06X", i)
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
		reg.MustRegister(style)
	}

	// MustRegister should panic when full
	defer func() {
		r := recover()
		assert.NotNil(t, r, "MustRegister should panic when registry is full")
	}()

	extraStyle := lipgloss.NewStyle().Background(lipgloss.Color("#FFFFFF"))
	reg.MustRegister(extraStyle)
	t.Error("MustRegister should have panicked")
}

func TestNewWithCapacity_ExceedsMax(t *testing.T) {
	// Creating with capacity > MaxStyles should cap at MaxStyles
	reg := NewWithCapacity(MaxStyles + 1000)
	assert.NotNil(t, reg)
	// The registry should still work and enforce MaxStyles limit
	for i := 0; i < MaxStyles; i++ {
		color := fmt.Sprintf("#%06X", i)
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
		id := reg.Register(style)
		assert.NotEquals(t, id, InvalidStyleID)
	}
	// Should fail at MaxStyles + 1
	extraStyle := lipgloss.NewStyle().Background(lipgloss.Color("#FFFFFF"))
	id := reg.Register(extraStyle)
	assert.Equals(t, id, InvalidStyleID)
}

// Benchmark tests
func BenchmarkLipglossRender_Simple(b *testing.B) {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = style.Render("Hello")
	}
}

func BenchmarkRegistryRender_Simple(b *testing.B) {
	reg := New()
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4"))
	id := reg.Register(style)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = reg.Render(id, "Hello")
	}
}

func BenchmarkLipglossRender_Empty(b *testing.B) {
	style := lipgloss.NewStyle()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = style.Render("Hello")
	}
}

func BenchmarkRegistryRender_Empty(b *testing.B) {
	reg := New()
	style := lipgloss.NewStyle()
	id := reg.Register(style)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = reg.Render(id, "Hello")
	}
}

func BenchmarkLipglossRender_Complex(b *testing.B) {
	style := lipgloss.NewStyle().
		Bold(true).
		Italic(true).
		Underline(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = style.Render("Hello World")
	}
}

func BenchmarkRegistryRender_Complex(b *testing.B) {
	reg := New()
	style := lipgloss.NewStyle().
		Bold(true).
		Italic(true).
		Underline(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4"))
	id := reg.Register(style)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = reg.Render(id, "Hello World")
	}
}

func BenchmarkLipglossRender_WithPadding(b *testing.B) {
	style := lipgloss.NewStyle().
		Bold(true).
		Padding(1, 2)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = style.Render("Hello")
	}
}

func BenchmarkRegistryRender_WithPadding(b *testing.B) {
	reg := New()
	style := lipgloss.NewStyle().
		Bold(true).
		Padding(1, 2)
	id := reg.Register(style)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = reg.Render(id, "Hello")
	}
}

func BenchmarkRenderBuilder(b *testing.B) {
	reg := New()
	style := lipgloss.NewStyle().Bold(true)
	id := reg.Register(style)
	var sb strings.Builder

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sb.Reset()
		reg.RenderBuilder(id, "Hello", &sb)
		_ = sb.String()
	}
}

func BenchmarkMultipleStyles(b *testing.B) {
	reg := New()

	bold := reg.Register(lipgloss.NewStyle().Bold(true))
	italic := reg.Register(lipgloss.NewStyle().Italic(true))
	colored := reg.Register(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = reg.Render(bold, "B")
		_ = reg.Render(italic, "I")
		_ = reg.Render(colored, "C")
	}
}

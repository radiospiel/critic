// Package styleregistry provides optimized style rendering by pre-compiling
// lipgloss styles into ANSI sequences. This avoids the overhead of style
// introspection on every render call.
//
// Usage:
//
//	reg := styleregistry.New()
//	id := reg.Register(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF0000")))
//	output := reg.Render(id, "Hello")
package styleregistry

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// StyleID is a compact identifier for a registered style.
// Using uint16 limits us to 65535 styles, which is plenty for any TUI.
type StyleID uint16

// InvalidStyleID represents an unregistered or invalid style.
const InvalidStyleID StyleID = 0xFFFF

// MaxStyles is the maximum number of styles the registry can hold.
const MaxStyles = 1024

// CompiledStyle holds pre-computed rendering data for a style.
type CompiledStyle struct {
	// Pre-computed ANSI sequences
	prefix string // ANSI codes to apply before text (e.g., "\x1b[1;38;2;255;0;0m")
	suffix string // ANSI codes to apply after text (typically reset: "\x1b[0m")

	// Original style for operations that need full lipgloss capabilities
	original lipgloss.Style

	// Layout parameters (needed at render time for slow path)
	padLeft, padRight, padTop, padBottom int
	width, height                        int
	maxWidth, maxHeight                  int
	alignHorizontal, alignVertical       lipgloss.Position

	// Fast path flag - true if we can skip lipgloss rendering entirely
	fastPath bool

	// Empty style flag - true if no styling is applied
	isEmpty bool
}

// StyleRegistry manages compiled styles with deduplication.
type StyleRegistry struct {
	styles []CompiledStyle
	hashes map[string]StyleID // maps style hash to existing StyleID for deduplication
}

// New creates a new StyleRegistry with pre-allocated capacity.
func New() *StyleRegistry {
	return &StyleRegistry{
		styles: make([]CompiledStyle, 0, 64),
		hashes: make(map[string]StyleID, 64),
	}
}

// NewWithCapacity creates a new StyleRegistry with the specified initial capacity.
func NewWithCapacity(capacity int) *StyleRegistry {
	if capacity > MaxStyles {
		capacity = MaxStyles
	}
	return &StyleRegistry{
		styles: make([]CompiledStyle, 0, capacity),
		hashes: make(map[string]StyleID, capacity),
	}
}

// Register compiles a lipgloss.Style and returns its ID.
// The style is analyzed once during registration, and subsequent renders
// use the pre-computed ANSI sequences.
//
// If an identical style has already been registered, the existing StyleID
// is returned (deduplication). Returns InvalidStyleID if the registry is full.
func (r *StyleRegistry) Register(s lipgloss.Style) StyleID {
	// Compute hash for deduplication
	hash := hashStyle(s)

	// Check for existing identical style
	if existingID, ok := r.hashes[hash]; ok {
		return existingID
	}

	// Check capacity
	if len(r.styles) >= MaxStyles {
		return InvalidStyleID
	}

	// Compile and store new style
	compiled := compile(s)
	id := StyleID(len(r.styles))
	r.styles = append(r.styles, compiled)
	r.hashes[hash] = id
	return id
}

// MustRegister is like Register but panics if the registry is full.
func (r *StyleRegistry) MustRegister(s lipgloss.Style) StyleID {
	id := r.Register(s)
	if id == InvalidStyleID {
		panic("styleregistry: registry is full (max 1024 styles)")
	}
	return id
}

// Render applies a compiled style to a string.
// This is the hot path - optimized for speed.
func (r *StyleRegistry) Render(id StyleID, s string) string {
	if int(id) >= len(r.styles) {
		return s // Invalid ID, return unmodified
	}

	cs := &r.styles[id]

	// Empty style - no transformation needed
	if cs.isEmpty {
		return s
	}

	// Fast path: no layout needed, just wrap with ANSI codes
	if cs.fastPath {
		return cs.prefix + s + cs.suffix
	}

	// Slow path: use original lipgloss style for layout operations
	return cs.original.Render(s)
}

// RenderBuilder appends a styled string to a strings.Builder.
// This is more efficient when building larger strings.
func (r *StyleRegistry) RenderBuilder(id StyleID, s string, sb *strings.Builder) {
	if int(id) >= len(r.styles) {
		sb.WriteString(s)
		return
	}

	cs := &r.styles[id]

	if cs.isEmpty {
		sb.WriteString(s)
		return
	}

	if cs.fastPath {
		sb.WriteString(cs.prefix)
		sb.WriteString(s)
		sb.WriteString(cs.suffix)
		return
	}

	sb.WriteString(cs.original.Render(s))
}

// Get returns the CompiledStyle for a given ID, or nil if invalid.
func (r *StyleRegistry) Get(id StyleID) *CompiledStyle {
	if int(id) >= len(r.styles) {
		return nil
	}
	return &r.styles[id]
}

// Len returns the number of registered styles.
func (r *StyleRegistry) Len() int {
	return len(r.styles)
}

// Prefix returns the pre-computed ANSI prefix for a style.
func (cs *CompiledStyle) Prefix() string {
	return cs.prefix
}

// Suffix returns the pre-computed ANSI suffix for a style.
func (cs *CompiledStyle) Suffix() string {
	return cs.suffix
}

// IsFastPath returns true if this style can use the fast rendering path.
func (cs *CompiledStyle) IsFastPath() bool {
	return cs.fastPath
}

// IsEmpty returns true if this style applies no formatting.
func (cs *CompiledStyle) IsEmpty() bool {
	return cs.isEmpty
}

// compile extracts style properties and pre-builds ANSI sequences.
func compile(s lipgloss.Style) CompiledStyle {
	cs := CompiledStyle{
		original: s,
	}

	// Extract layout properties using lipgloss getters
	cs.padTop = s.GetPaddingTop()
	cs.padBottom = s.GetPaddingBottom()
	cs.padLeft = s.GetPaddingLeft()
	cs.padRight = s.GetPaddingRight()
	cs.width = s.GetWidth()
	cs.height = s.GetHeight()
	cs.maxWidth = s.GetMaxWidth()
	cs.maxHeight = s.GetMaxHeight()
	cs.alignHorizontal = s.GetAlign()
	cs.alignVertical = s.GetAlignVertical()

	// Check if this style has any layout operations
	hasLayout := cs.padTop > 0 || cs.padBottom > 0 || cs.padLeft > 0 || cs.padRight > 0 ||
		cs.width > 0 || cs.height > 0 || cs.maxWidth > 0 || cs.maxHeight > 0

	// Check for borders and margins which also require layout
	hasBorder := s.GetBorderTop() || s.GetBorderBottom() || s.GetBorderLeft() || s.GetBorderRight()
	hasMargin := s.GetMarginTop() > 0 || s.GetMarginBottom() > 0 ||
		s.GetMarginLeft() > 0 || s.GetMarginRight() > 0

	// Extract ANSI codes by rendering an empty string and a test string
	// The difference reveals the prefix and suffix
	emptyRender := s.Render("")

	// Check if the style is empty (no formatting at all)
	if emptyRender == "" {
		cs.isEmpty = true
		cs.fastPath = true
		return cs
	}

	// For fast path determination:
	// We can use fast path if there's no layout (padding, width, border, margin)
	// and the style only applies ANSI formatting codes
	cs.fastPath = !hasLayout && !hasBorder && !hasMargin

	if cs.fastPath {
		// Extract prefix and suffix from empty render
		// For lipgloss styles without layout, Render("") gives us the ANSI codes
		// and Render("x") gives us prefix + "x" + suffix
		cs.prefix, cs.suffix = extractANSICodes(s)
	}

	return cs
}

// extractANSICodes extracts the ANSI prefix and suffix from a style.
// It does this by rendering a test string and extracting the surrounding codes.
func extractANSICodes(s lipgloss.Style) (prefix, suffix string) {
	// Use a unique marker to find where the content goes
	const marker = "\x00"
	rendered := s.Render(marker)

	// Find the marker position
	idx := strings.Index(rendered, marker)
	if idx == -1 {
		// Marker not found (shouldn't happen), fall back to empty
		return "", ""
	}

	prefix = rendered[:idx]
	suffix = rendered[idx+len(marker):]

	return prefix, suffix
}

// hashStyle computes a hash string for a style based on its properties.
// Two styles with identical properties produce the same hash.
func hashStyle(s lipgloss.Style) string {
	// Build hash from all style properties to ensure uniqueness
	// even in non-TTY environments where ANSI codes aren't produced
	var sb strings.Builder

	// Text attributes
	if s.GetBold() {
		sb.WriteString("B")
	}
	if s.GetItalic() {
		sb.WriteString("I")
	}
	if s.GetUnderline() {
		sb.WriteString("U")
	}
	if s.GetStrikethrough() {
		sb.WriteString("S")
	}
	if s.GetReverse() {
		sb.WriteString("R")
	}
	if s.GetBlink() {
		sb.WriteString("K")
	}
	if s.GetFaint() {
		sb.WriteString("F")
	}

	// Colors - include the rendered output which captures color values
	// This works because even without ANSI output, the color values differ
	sb.WriteString("|fg:")
	sb.WriteString(fmt.Sprintf("%v", s.GetForeground()))
	sb.WriteString("|bg:")
	sb.WriteString(fmt.Sprintf("%v", s.GetBackground()))

	// Layout properties
	sb.WriteString(fmt.Sprintf("|p:%d,%d,%d,%d",
		s.GetPaddingTop(), s.GetPaddingRight(), s.GetPaddingBottom(), s.GetPaddingLeft()))
	sb.WriteString(fmt.Sprintf("|m:%d,%d,%d,%d",
		s.GetMarginTop(), s.GetMarginRight(), s.GetMarginBottom(), s.GetMarginLeft()))
	sb.WriteString(fmt.Sprintf("|w:%d|h:%d|mw:%d|mh:%d",
		s.GetWidth(), s.GetHeight(), s.GetMaxWidth(), s.GetMaxHeight()))

	// Border
	if s.GetBorderTop() || s.GetBorderBottom() || s.GetBorderLeft() || s.GetBorderRight() {
		sb.WriteString(fmt.Sprintf("|border:%v,%v,%v,%v",
			s.GetBorderTop(), s.GetBorderRight(), s.GetBorderBottom(), s.GetBorderLeft()))
		sb.WriteString(fmt.Sprintf("|bstyle:%v", s.GetBorderStyle()))
	}

	// Alignment
	sb.WriteString(fmt.Sprintf("|align:%v,%v", s.GetAlign(), s.GetAlignVertical()))

	return sb.String()
}

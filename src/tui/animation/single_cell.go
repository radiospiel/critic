// Package animation provides terminal spinner/loading animation definitions
// for use with bubbletea and lipgloss.
package animation

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Type identifies different animation styles.
type Type int

const (
	BrailleSpinner Type = iota
	CircleQuarters
	DotPulse
	VerticalBounce
	StarBurst
	ExpandingRings
	BrailleSnake
	BlockQuadrants
)

// Animation defines the frames, colors, speed, and current state for an animation.
type Animation struct {
	Frames  []string
	Colors  []string
	Speed   time.Duration
	Colored bool
	Frame   int // current frame index
}

// Render returns the current frame, with or without color based on the Colored field.
func (a *Animation) Render() string {
	frame := a.Frames[a.Frame%len(a.Frames)]
	if !a.Colored {
		return frame
	}
	colorIdx := a.Frame % len(a.Colors)
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(a.Colors[colorIdx]))
	return style.Render(frame)
}

// Rune returns the current frame as a rune (for cell-based rendering).
func (a *Animation) Rune() rune {
	frame := a.Frames[a.Frame%len(a.Frames)]
	runes := []rune(frame)
	if len(runes) > 0 {
		return runes[0]
	}
	return ' '
}

// Style returns the lipgloss style for the current frame.
func (a *Animation) Style() lipgloss.Style {
	if !a.Colored {
		return lipgloss.NewStyle()
	}
	colorIdx := a.Frame % len(a.Colors)
	return lipgloss.NewStyle().Foreground(lipgloss.Color(a.Colors[colorIdx]))
}

// Tick advances the animation to the next frame.
func (a *Animation) Tick() {
	a.Frame = (a.Frame + 1) % len(a.Frames)
}

// FrameAt computes the current frame index from a global tick count.
// This allows animations to be stateless - the frame is computed from the global tick.
// tickInterval should be the compositor's tick interval (e.g., 40ms).
func (a *Animation) FrameAt(globalTick int64, tickInterval time.Duration) int {
	return int(int64(float64(globalTick)*float64(tickInterval)/float64(a.Speed)) % int64(len(a.Frames)))
}

// RenderAt returns the rendered frame at the given global tick.
func (a *Animation) RenderAt(globalTick int64, tickInterval time.Duration) string {
	frameIdx := a.FrameAt(globalTick, tickInterval)
	frame := a.Frames[frameIdx]
	if !a.Colored {
		return frame
	}
	colorIdx := frameIdx % len(a.Colors)
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(a.Colors[colorIdx]))
	return style.Render(frame)
}

// RuneAt returns the frame rune at the given global tick.
func (a *Animation) RuneAt(globalTick int64, tickInterval time.Duration) rune {
	frameIdx := a.FrameAt(globalTick, tickInterval)
	frame := a.Frames[frameIdx]
	runes := []rune(frame)
	if len(runes) > 0 {
		return runes[0]
	}
	return ' '
}

// StyleAt returns the lipgloss style at the given global tick.
func (a *Animation) StyleAt(globalTick int64, tickInterval time.Duration) lipgloss.Style {
	if !a.Colored {
		return lipgloss.NewStyle()
	}
	frameIdx := a.FrameAt(globalTick, tickInterval)
	colorIdx := frameIdx % len(a.Colors)
	return lipgloss.NewStyle().Foreground(lipgloss.Color(a.Colors[colorIdx]))
}

// animations contains all available animation definitions.
var animations = map[Type]Animation{
	BrailleSpinner: {
		Frames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		Colors:  []string{"#61AFEF", "#61AFEF", "#61AFEF", "#61AFEF", "#61AFEF"},
		Speed:   80 * time.Millisecond,
		Colored: true,
	},
	CircleQuarters: {
		Frames:  []string{"◐", "◓", "◑", "◒"},
		Colors:  []string{"#E06C75", "#E5C07B", "#98C379", "#61AFEF"},
		Speed:   100 * time.Millisecond,
		Colored: true,
	},
	DotPulse: {
		Frames:  []string{"·", "∘", "○", "◯", "◉", "●", "◉", "◯", "○", "∘"},
		Colors:  []string{"#3B4048", "#4B5058", "#5B6068", "#6B7078", "#7B8088", "#8B9098", "#7B8088", "#6B7078", "#5B6068", "#4B5058"},
		Speed:   60 * time.Millisecond,
		Colored: true,
	},
	VerticalBounce: {
		Frames:  []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█", "▇", "▆", "▅", "▄", "▃", "▂"},
		Colors:  []string{"#98C379", "#98C379", "#98C379", "#98C379", "#98C379", "#98C379", "#98C379"},
		Speed:   50 * time.Millisecond,
		Colored: true,
	},
	StarBurst: {
		Frames:  []string{"·", "∗", "✦", "✧", "✶", "✷", "✸", "✹", "✺", "✻", "✼", "✽", "✾", "✿", " "},
		Colors:  []string{"#3B4048", "#E5C07B", "#E5C07B", "#E06C75", "#E06C75", "#C678DD", "#C678DD", "#61AFEF", "#61AFEF", "#56B6C2", "#56B6C2", "#98C379", "#98C379", "#FFFFFF", "#000000"},
		Speed:   40 * time.Millisecond,
		Colored: true,
	},
	ExpandingRings: {
		Frames:  []string{"∘", "○", "◎", "⊙", "⊚", "⊛", " "},
		Colors:  []string{"#61AFEF", "#56B6C2", "#98C379", "#E5C07B", "#E06C75", "#C678DD", "#000000"},
		Speed:   60 * time.Millisecond,
		Colored: true,
	},
	BrailleSnake: {
		Frames:  []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"},
		Colors:  []string{"#C678DD", "#C678DD", "#C678DD", "#C678DD"},
		Speed:   80 * time.Millisecond,
		Colored: true,
	},
	BlockQuadrants: {
		Frames:  []string{"▖", "▘", "▝", "▗"},
		Colors:  []string{"#E06C75", "#E5C07B", "#98C379", "#61AFEF"},
		Speed:   120 * time.Millisecond,
		Colored: true,
	},
}

// Animation type names (indexed by Type)
var typeNames = []string{
	"Braille Spinner",
	"Circle Quarters",
	"Dot Pulse",
	"Vertical Bounce",
	"Star Burst",
	"Expanding Rings",
	"Braille Snake",
	"Block Quadrants",
}

// Name returns the human-readable name of the animation type.
func (t Type) Name() string {
	if int(t) < len(typeNames) {
		return typeNames[t]
	}
	return "Unknown"
}

// Get returns a copy of the Animation for the given type.
func Get(t Type) Animation {
	return animations[t]
}

// NewSingleCellAnimation returns a new Animation configured for the given type
// with the specified color mode and speed factor.
// Speed factor: 1.0 = normal, <1.0 = faster, >1.0 = slower.
func NewSingleCellAnimation(t Type, colored bool, speedFactor float64) *Animation {
	base := animations[t]
	return &Animation{
		Frames:  base.Frames,
		Colors:  base.Colors,
		Speed:   time.Duration(float64(base.Speed) * speedFactor),
		Colored: colored,
		Frame:   0,
	}
}

// TickMsg is sent when it's time to update the animation frame.
type TickMsg struct {
	Time time.Time
	Anim *Animation
}

// TickCmd returns a tea.Cmd that sends a TickMsg after the animation's speed interval.
func (a *Animation) TickCmd() tea.Cmd {
	return tea.Tick(a.Speed, func(tm time.Time) tea.Msg {
		return TickMsg{Time: tm, Anim: a}
	})
}

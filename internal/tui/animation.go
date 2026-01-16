package tui

import (
	"strings"
	"sync"

	"git.15b.it/eno/critic/pkg/critic"
	"git.15b.it/eno/critic/simple-tui/animation"
	"git.15b.it/eno/critic/teapot"
	"github.com/charmbracelet/lipgloss"
)

// AnimationState represents the current animation state for conversations
type AnimationState int

const (
	// NoAnimation - conversation is not read by AI or is resolved
	NoAnimation AnimationState = iota
	// ThinkingAnimation - read by AI but not answered (last message is human)
	ThinkingAnimation
	// LookHereAnimation - read by AI and answered (last message is AI) but not resolved
	LookHereAnimation
)

// Animation configuration
const (
	// ThinkingAnimationType - BrailleSnake for when AI is thinking
	ThinkingAnimationType = animation.BrailleSnake
	// LookHereAnimationType - StarBurst for when AI is waiting for user
	LookHereAnimationType = animation.StarBurst
	// LookHereSpeedFactor - 71% speed (1/0.71 ≈ 1.41 multiplier on duration)
	LookHereSpeedFactor = 1.41
	// SeparatorAnimationType - Snake for separator line in diff view
	SeparatorAnimationType = animation.Snake
	// SeparatorSpeedFactor - 62% speed (1/0.62 ≈ 1.61 multiplier on duration)
	SeparatorSpeedFactor = 1.61
)

// tickInterval is the base tick rate for animations (use fastest animation speed)
// Note: This now uses the ComposerTickInterval from the teapot package.
const tickInterval = teapot.ComposerTickInterval

// AnimationTicker holds the animations and provides animation state.
// It implements teapot.Ticker interface.
type AnimationTicker struct {
	mu             sync.RWMutex
	thinking       *animation.Animation
	lookHere       *animation.Animation
	separator      *animation.Animation
	thinkingTicks  int // accumulator for thinking animation
	lookHereTicks  int // accumulator for look here animation
	separatorTicks int // accumulator for separator animation
}

// NewAnimationTicker creates a new animation ticker
func NewAnimationTicker() *AnimationTicker {
	return &AnimationTicker{
		thinking:  animation.NewSingleCellAnimation(ThinkingAnimationType, true, 1.0),
		lookHere:  animation.NewSingleCellAnimation(LookHereAnimationType, true, LookHereSpeedFactor),
		separator: animation.NewShortAnimation(SeparatorAnimationType, true, SeparatorSpeedFactor),
	}
}

// GetFrame returns the current animation frame for the given state (with color)
func (at *AnimationTicker) GetFrame(state AnimationState, long bool) string {
	at.mu.RLock()
	defer at.mu.RUnlock()

	switch state {
	case ThinkingAnimation:
		frame := at.thinking.Render()
		if long {
			return padToWidth(frame, 10)
		}
		return frame
	case LookHereAnimation:
		frame := at.lookHere.Render()
		if long {
			return padToWidth(frame, 10)
		}
		return frame
	default:
		if long {
			return "          " // 10 spaces
		}
		return " "
	}
}

// GetFrameRune returns the current animation frame character (without color)
func (at *AnimationTicker) GetFrameRune(state AnimationState) rune {
	at.mu.RLock()
	defer at.mu.RUnlock()

	switch state {
	case ThinkingAnimation:
		return at.thinking.Rune()
	case LookHereAnimation:
		return at.lookHere.Rune()
	default:
		return ' '
	}
}

// GetFrameStyle returns the lipgloss style for the animation state
func (at *AnimationTicker) GetFrameStyle(state AnimationState) lipgloss.Style {
	at.mu.RLock()
	defer at.mu.RUnlock()

	switch state {
	case ThinkingAnimation:
		return at.thinking.Style()
	case LookHereAnimation:
		return at.lookHere.Style()
	default:
		return lipgloss.NewStyle()
	}
}

// GetSeparatorFrame returns the current separator animation frame (12-char snake)
func (at *AnimationTicker) GetSeparatorFrame() string {
	at.mu.RLock()
	defer at.mu.RUnlock()
	return at.separator.Render()
}

// padToWidth pads a string to exactly width characters (accounts for ANSI codes)
func padToWidth(s string, width int) string {
	// The animation frames are single characters, so we need to pad
	// But they may have ANSI color codes, so we count visible width
	visibleLen := 1 // animation frames are single chars
	if visibleLen < width {
		return s + strings.Repeat(" ", width-visibleLen)
	}
	return s
}

// Tick advances the animation frames based on their respective speeds
func (at *AnimationTicker) Tick() {
	at.mu.Lock()
	defer at.mu.Unlock()

	// Thinking animation: BrailleSnake at 80ms per frame
	// At 40ms tick rate, advance every 2 ticks
	thinkingTicksNeeded := int(at.thinking.Speed / tickInterval)
	if thinkingTicksNeeded < 1 {
		thinkingTicksNeeded = 1
	}
	at.thinkingTicks++
	if at.thinkingTicks >= thinkingTicksNeeded {
		at.thinkingTicks = 0
		at.thinking.Tick()
	}

	// LookHere animation: StarBurst at adjusted speed
	lookHereTicksNeeded := int(at.lookHere.Speed / tickInterval)
	if lookHereTicksNeeded < 1 {
		lookHereTicksNeeded = 1
	}
	at.lookHereTicks++
	if at.lookHereTicks >= lookHereTicksNeeded {
		at.lookHereTicks = 0
		at.lookHere.Tick()
	}

	// Separator animation: Snake short animation at 62% speed
	separatorTicksNeeded := int(at.separator.Speed / tickInterval)
	if separatorTicksNeeded < 1 {
		separatorTicksNeeded = 1
	}
	at.separatorTicks++
	if at.separatorTicks >= separatorTicksNeeded {
		at.separatorTicks = 0
		at.separator.Tick()
	}
}

// GetConversationAnimationState determines the animation state for a conversation
// (A) ReadByAI and not answered (last message is human) => ThinkingAnimation
// (B) ReadByAI and answered (last message is AI) but not resolved => LookHereAnimation
// Otherwise => NoAnimation
func GetConversationAnimationState(conv *critic.Conversation) AnimationState {
	// Not read by AI - no animation
	if !conv.ReadByAI {
		return NoAnimation
	}

	// Resolved - no animation
	if conv.Status == critic.StatusResolved {
		return NoAnimation
	}

	// Check last message author
	if len(conv.Messages) == 0 {
		return NoAnimation
	}

	lastMsg := conv.Messages[len(conv.Messages)-1]
	if lastMsg.Author == critic.AuthorAI {
		// Last message is AI - look here animation (call to action for user)
		return LookHereAnimation
	}

	// Last message is human - thinking animation (AI is working on it)
	return ThinkingAnimation
}

// FileAnimationSummary holds animation info for a file
type FileAnimationSummary struct {
	HasThinking bool
	HasLookHere bool
}

// GetFileAnimationState returns the animation state for a file
// If file has any LookHere conversations, return LookHere (higher priority)
// If file has any Thinking conversations, return Thinking
// Otherwise return NoAnimation
func GetFileAnimationState(summary FileAnimationSummary) AnimationState {
	if summary.HasLookHere {
		return LookHereAnimation
	}
	if summary.HasThinking {
		return ThinkingAnimation
	}
	return NoAnimation
}

// GlobalAnimationSummary holds animation info for the entire app
type GlobalAnimationSummary struct {
	HasThinking bool
	HasLookHere bool
}

// GetGlobalAnimationState returns the animation state for the status bar
func GetGlobalAnimationState(summary GlobalAnimationSummary) AnimationState {
	if summary.HasLookHere {
		return LookHereAnimation
	}
	if summary.HasThinking {
		return ThinkingAnimation
	}
	return NoAnimation
}

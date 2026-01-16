package tui

import (
	"strings"

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

// Static animation instances - these are stateless, frames are computed from GlobalTickCount
var (
	thinkingAnim  = animation.NewSingleCellAnimation(ThinkingAnimationType, true, 1.0)
	lookHereAnim  = animation.NewSingleCellAnimation(LookHereAnimationType, true, LookHereSpeedFactor)
	separatorAnim = animation.NewShortAnimation(SeparatorAnimationType, true, SeparatorSpeedFactor)
)

// GetFrame returns the current animation frame for the given state (with color)
func GetFrame(state AnimationState, long bool) string {
	tick := teapot.GlobalTickCount
	interval := teapot.ComposerTickInterval

	switch state {
	case ThinkingAnimation:
		frame := thinkingAnim.RenderAt(tick, interval)
		if long {
			return padToWidth(frame, 10)
		}
		return frame
	case LookHereAnimation:
		frame := lookHereAnim.RenderAt(tick, interval)
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
func GetFrameRune(state AnimationState) rune {
	tick := teapot.GlobalTickCount
	interval := teapot.ComposerTickInterval

	switch state {
	case ThinkingAnimation:
		return thinkingAnim.RuneAt(tick, interval)
	case LookHereAnimation:
		return lookHereAnim.RuneAt(tick, interval)
	default:
		return ' '
	}
}

// GetFrameStyle returns the lipgloss style for the animation state
func GetFrameStyle(state AnimationState) lipgloss.Style {
	tick := teapot.GlobalTickCount
	interval := teapot.ComposerTickInterval

	switch state {
	case ThinkingAnimation:
		return thinkingAnim.StyleAt(tick, interval)
	case LookHereAnimation:
		return lookHereAnim.StyleAt(tick, interval)
	default:
		return lipgloss.NewStyle()
	}
}

// GetSeparatorFrame returns the current separator animation frame (12-char snake)
func GetSeparatorFrame() string {
	tick := teapot.GlobalTickCount
	interval := teapot.ComposerTickInterval
	return separatorAnim.RenderAt(tick, interval)
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

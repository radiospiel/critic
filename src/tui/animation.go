package tui

import (
	"github.com/radiospiel/critic/src/pkg/critic"
	"github.com/radiospiel/critic/src/tui/animation"
	"github.com/radiospiel/critic/teapot"
	"github.com/charmbracelet/lipgloss"
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

// GetAnimatedIndicator returns the animated indicator rune and style.
// LookHere has priority over Thinking.
// Returns (' ', empty style) if no animation is active.
func GetAnimatedIndicator(hasThinking, hasLookHere bool) (rune, lipgloss.Style) {
	tick := teapot.GlobalTickCount
	interval := teapot.ComposerTickInterval

	if hasLookHere {
		return lookHereAnim.RuneAt(tick, interval), lookHereAnim.StyleAt(tick, interval)
	}
	if hasThinking {
		return thinkingAnim.RuneAt(tick, interval), thinkingAnim.StyleAt(tick, interval)
	}
	return ' ', lipgloss.NewStyle()
}

// GetSeparatorFrame returns the current separator animation frame (12-char snake)
func GetSeparatorFrame() string {
	tick := teapot.GlobalTickCount
	interval := teapot.ComposerTickInterval
	return separatorAnim.RenderAt(tick, interval)
}

// FileAnimationSummary holds animation info for a file, aggregated from its conversations
type FileAnimationSummary struct {
	HasThinking bool
	HasLookHere bool
}

// UpdateFromConversation updates the summary based on a conversation's state.
// Returns true if the summary changed.
func (s *FileAnimationSummary) UpdateFromConversation(conv *critic.Conversation) bool {
	// Not read by AI or resolved - no animation
	if !conv.ReadByAI || conv.Status == critic.StatusResolved || len(conv.Messages) == 0 {
		return false
	}

	lastMsg := conv.Messages[len(conv.Messages)-1]
	if lastMsg.Author == critic.AuthorAI {
		// Last message is AI - look here animation
		if !s.HasLookHere {
			s.HasLookHere = true
			return true
		}
	} else {
		// Last message is human - thinking animation
		if !s.HasThinking {
			s.HasThinking = true
			return true
		}
	}
	return false
}

// HasAnimation returns true if any animation is active
func (s *FileAnimationSummary) HasAnimation() bool {
	return s.HasThinking || s.HasLookHere
}

package ui

import (
	"sync"
	"time"

	"git.15b.it/eno/critic/pkg/critic"
	tea "github.com/charmbracelet/bubbletea"
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

// Animation frames for different states
// Each frame should be single-width ASCII/Unicode characters

// ThinkingFrames - dots animation for "thinking" state (10 chars)
var ThinkingFrames = []string{
	".         ",
	"..        ",
	"...       ",
	"....      ",
	".....     ",
	"......    ",
	".......   ",
	"........  ",
	"......... ",
	"..........",
	"......... ",
	"........  ",
	".......   ",
	"......    ",
	".....     ",
	"....      ",
	"...       ",
	"..        ",
}

// ThinkingFramesShort - single char thinking animation
var ThinkingFramesShort = []string{
	".",
	"o",
	"O",
	"o",
}

// LookHereFrames - jumping animation for "look here" state (10 chars)
var LookHereFrames = []string{
	">>        ",
	" >>       ",
	"  >>      ",
	"   >>     ",
	"    >>    ",
	"     >>   ",
	"      >>  ",
	"       >> ",
	"        >>",
	"       >> ",
	"      >>  ",
	"     >>   ",
	"    >>    ",
	"   >>     ",
	"  >>      ",
	" >>       ",
}

// LookHereFramesShort - single char look here animation
var LookHereFramesShort = []string{
	">",
	"*",
	"<",
	"*",
}

// AnimationTicker holds the current frame index and provides animation state
type AnimationTicker struct {
	mu           sync.RWMutex
	frameIndex   int
	tickInterval time.Duration
}

// NewAnimationTicker creates a new animation ticker
func NewAnimationTicker() *AnimationTicker {
	return &AnimationTicker{
		tickInterval: 200 * time.Millisecond, // 200ms per frame
	}
}

// GetFrame returns the current animation frame for the given state
func (at *AnimationTicker) GetFrame(state AnimationState, long bool) string {
	at.mu.RLock()
	defer at.mu.RUnlock()

	switch state {
	case ThinkingAnimation:
		if long {
			return ThinkingFrames[at.frameIndex%len(ThinkingFrames)]
		}
		return ThinkingFramesShort[at.frameIndex%len(ThinkingFramesShort)]
	case LookHereAnimation:
		if long {
			return LookHereFrames[at.frameIndex%len(LookHereFrames)]
		}
		return LookHereFramesShort[at.frameIndex%len(LookHereFramesShort)]
	default:
		if long {
			return "          " // 10 spaces
		}
		return " "
	}
}

// Tick advances the animation frame
func (at *AnimationTicker) Tick() {
	at.mu.Lock()
	defer at.mu.Unlock()
	at.frameIndex++
}

// AnimationTickMsg is sent when it's time to update animations
type AnimationTickMsg struct{}

// StartAnimationTicker returns a command that sends ticks every 200ms
func StartAnimationTicker() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return AnimationTickMsg{}
	})
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
	HasThinking  bool
	HasLookHere  bool
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
	HasThinking  bool
	HasLookHere  bool
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

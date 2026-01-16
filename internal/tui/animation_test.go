package tui

import (
	"testing"

	"git.15b.it/eno/critic/pkg/critic"
	"git.15b.it/eno/critic/simple-go/assert"
	"git.15b.it/eno/critic/simple-tui/animation"
)

func TestGetConversationAnimationState_NoAnimation(t *testing.T) {
	// Test NoAnimation when not read by AI
	conv := &critic.Conversation{
		ReadByAI: false,
		Status:   critic.StatusUnresolved,
		Messages: []critic.Message{{Author: critic.AuthorHuman}},
	}
	state := GetConversationAnimationState(conv)
	assert.Equals(t, state, NoAnimation, "expected NoAnimation when ReadByAI is false")

	// Test NoAnimation when resolved
	conv = &critic.Conversation{
		ReadByAI: true,
		Status:   critic.StatusResolved,
		Messages: []critic.Message{{Author: critic.AuthorHuman}},
	}
	state = GetConversationAnimationState(conv)
	assert.Equals(t, state, NoAnimation, "expected NoAnimation when Status is resolved")

	// Test NoAnimation when no messages
	conv = &critic.Conversation{
		ReadByAI: true,
		Status:   critic.StatusUnresolved,
		Messages: []critic.Message{},
	}
	state = GetConversationAnimationState(conv)
	assert.Equals(t, state, NoAnimation, "expected NoAnimation when no messages")
}

func TestGetConversationAnimationState_ThinkingAnimation(t *testing.T) {
	// Test ThinkingAnimation when last message is human
	conv := &critic.Conversation{
		ReadByAI: true,
		Status:   critic.StatusUnresolved,
		Messages: []critic.Message{
			{Author: critic.AuthorAI},
			{Author: critic.AuthorHuman}, // Last message is human
		},
	}
	state := GetConversationAnimationState(conv)
	assert.Equals(t, state, ThinkingAnimation, "expected ThinkingAnimation when last message is human")

	// Test with single human message
	conv = &critic.Conversation{
		ReadByAI: true,
		Status:   critic.StatusUnresolved,
		Messages: []critic.Message{
			{Author: critic.AuthorHuman},
		},
	}
	state = GetConversationAnimationState(conv)
	assert.Equals(t, state, ThinkingAnimation, "expected ThinkingAnimation with single human message")
}

func TestGetConversationAnimationState_LookHereAnimation(t *testing.T) {
	// Test LookHereAnimation when last message is AI
	conv := &critic.Conversation{
		ReadByAI: true,
		Status:   critic.StatusUnresolved,
		Messages: []critic.Message{
			{Author: critic.AuthorHuman},
			{Author: critic.AuthorAI}, // Last message is AI
		},
	}
	state := GetConversationAnimationState(conv)
	assert.Equals(t, state, LookHereAnimation, "expected LookHereAnimation when last message is AI")

	// Test with single AI message
	conv = &critic.Conversation{
		ReadByAI: true,
		Status:   critic.StatusUnresolved,
		Messages: []critic.Message{
			{Author: critic.AuthorAI},
		},
	}
	state = GetConversationAnimationState(conv)
	assert.Equals(t, state, LookHereAnimation, "expected LookHereAnimation with single AI message")
}

func TestGetFrame(t *testing.T) {
	// Get the animation definitions
	thinkingAnim := animation.Get(ThinkingAnimationType)
	lookHereAnim := animation.Get(LookHereAnimationType)

	// Test NoAnimation returns spaces
	noAnimFrame := GetFrame(NoAnimation, false)
	assert.Equals(t, noAnimFrame, " ", "expected space for NoAnimation short")

	noAnimFrameLong := GetFrame(NoAnimation, true)
	assert.Equals(t, noAnimFrameLong, "          ", "expected 10 spaces for NoAnimation long")

	// Test ThinkingAnimation returns valid BrailleSnake frames
	thinkingFrame := GetFrameRune(ThinkingAnimation)
	validThinkingRune := false
	for _, f := range thinkingAnim.Frames {
		runes := []rune(f)
		if len(runes) > 0 && runes[0] == thinkingFrame {
			validThinkingRune = true
			break
		}
	}
	assert.True(t, validThinkingRune, "expected valid BrailleSnake frame rune")

	// Test LookHereAnimation returns valid StarBurst frames
	lookHereFrame := GetFrameRune(LookHereAnimation)
	validLookHereRune := false
	for _, f := range lookHereAnim.Frames {
		runes := []rune(f)
		if len(runes) > 0 && runes[0] == lookHereFrame {
			validLookHereRune = true
			break
		}
	}
	assert.True(t, validLookHereRune, "expected valid StarBurst frame rune")
}

func TestGetFileAnimationState(t *testing.T) {
	// Test NoAnimation
	summary := FileAnimationSummary{
		HasThinking:  false,
		HasLookHere:  false,
	}
	state := GetFileAnimationState(summary)
	assert.Equals(t, state, NoAnimation, "expected NoAnimation when no flags set")

	// Test ThinkingAnimation
	summary = FileAnimationSummary{
		HasThinking:  true,
		HasLookHere:  false,
	}
	state = GetFileAnimationState(summary)
	assert.Equals(t, state, ThinkingAnimation, "expected ThinkingAnimation when HasThinking is true")

	// Test LookHereAnimation takes priority
	summary = FileAnimationSummary{
		HasThinking:  true,
		HasLookHere:  true,
	}
	state = GetFileAnimationState(summary)
	assert.Equals(t, state, LookHereAnimation, "expected LookHereAnimation when both flags set (higher priority)")

	// Test LookHereAnimation alone
	summary = FileAnimationSummary{
		HasThinking:  false,
		HasLookHere:  true,
	}
	state = GetFileAnimationState(summary)
	assert.Equals(t, state, LookHereAnimation, "expected LookHereAnimation when HasLookHere is true")
}

func TestGetGlobalAnimationState(t *testing.T) {
	// Test NoAnimation
	summary := GlobalAnimationSummary{
		HasThinking:  false,
		HasLookHere:  false,
	}
	state := GetGlobalAnimationState(summary)
	assert.Equals(t, state, NoAnimation, "expected NoAnimation when no flags set")

	// Test ThinkingAnimation
	summary = GlobalAnimationSummary{
		HasThinking:  true,
		HasLookHere:  false,
	}
	state = GetGlobalAnimationState(summary)
	assert.Equals(t, state, ThinkingAnimation, "expected ThinkingAnimation when HasThinking is true")

	// Test LookHereAnimation takes priority
	summary = GlobalAnimationSummary{
		HasThinking:  true,
		HasLookHere:  true,
	}
	state = GetGlobalAnimationState(summary)
	assert.Equals(t, state, LookHereAnimation, "expected LookHereAnimation when both flags set (higher priority)")
}

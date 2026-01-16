package tui

import (
	"testing"

	"git.15b.it/eno/critic/pkg/critic"
	"git.15b.it/eno/critic/simple-go/assert"
	"git.15b.it/eno/critic/simple-tui/animation"
)

func TestUpdateFromConversation_NoChange(t *testing.T) {
	// Test no change when not read by AI
	conv := &critic.Conversation{
		ReadByAI: false,
		Status:   critic.StatusUnresolved,
		Messages: []critic.Message{{Author: critic.AuthorHuman}},
	}
	var summary FileAnimationSummary
	changed := summary.UpdateFromConversation(conv)
	assert.False(t, changed, "expected no change when ReadByAI is false")
	assert.False(t, summary.HasAnimation(), "expected no animation when ReadByAI is false")

	// Test no change when resolved
	conv = &critic.Conversation{
		ReadByAI: true,
		Status:   critic.StatusResolved,
		Messages: []critic.Message{{Author: critic.AuthorHuman}},
	}
	summary = FileAnimationSummary{}
	changed = summary.UpdateFromConversation(conv)
	assert.False(t, changed, "expected no change when Status is resolved")
	assert.False(t, summary.HasAnimation(), "expected no animation when resolved")

	// Test no change when no messages
	conv = &critic.Conversation{
		ReadByAI: true,
		Status:   critic.StatusUnresolved,
		Messages: []critic.Message{},
	}
	summary = FileAnimationSummary{}
	changed = summary.UpdateFromConversation(conv)
	assert.False(t, changed, "expected no change when no messages")
	assert.False(t, summary.HasAnimation(), "expected no animation when no messages")
}

func TestUpdateFromConversation_Thinking(t *testing.T) {
	// Test HasThinking set when last message is human
	conv := &critic.Conversation{
		ReadByAI: true,
		Status:   critic.StatusUnresolved,
		Messages: []critic.Message{
			{Author: critic.AuthorAI},
			{Author: critic.AuthorHuman}, // Last message is human
		},
	}
	var summary FileAnimationSummary
	changed := summary.UpdateFromConversation(conv)
	assert.True(t, changed, "expected change when last message is human")
	assert.True(t, summary.HasThinking, "expected HasThinking when last message is human")
	assert.False(t, summary.HasLookHere, "expected no HasLookHere")

	// Test no change when HasThinking already set
	changed = summary.UpdateFromConversation(conv)
	assert.False(t, changed, "expected no change when HasThinking already set")
}

func TestUpdateFromConversation_LookHere(t *testing.T) {
	// Test HasLookHere set when last message is AI
	conv := &critic.Conversation{
		ReadByAI: true,
		Status:   critic.StatusUnresolved,
		Messages: []critic.Message{
			{Author: critic.AuthorHuman},
			{Author: critic.AuthorAI}, // Last message is AI
		},
	}
	var summary FileAnimationSummary
	changed := summary.UpdateFromConversation(conv)
	assert.True(t, changed, "expected change when last message is AI")
	assert.True(t, summary.HasLookHere, "expected HasLookHere when last message is AI")
	assert.False(t, summary.HasThinking, "expected no HasThinking")

	// Test no change when HasLookHere already set
	changed = summary.UpdateFromConversation(conv)
	assert.False(t, changed, "expected no change when HasLookHere already set")
}

func TestGetAnimatedIndicator(t *testing.T) {
	// Get the animation definitions for validation
	thinkingAnimDef := animation.Get(ThinkingAnimationType)
	lookHereAnimDef := animation.Get(LookHereAnimationType)

	// Test no animation returns space
	r, _ := GetAnimatedIndicator(false, false)
	assert.Equals(t, r, ' ', "expected space for no animation")

	// Test HasThinking returns valid BrailleSnake frame
	thinkingRune, _ := GetAnimatedIndicator(true, false)
	validThinkingRune := false
	for _, f := range thinkingAnimDef.Frames {
		runes := []rune(f)
		if len(runes) > 0 && runes[0] == thinkingRune {
			validThinkingRune = true
			break
		}
	}
	assert.True(t, validThinkingRune, "expected valid BrailleSnake frame rune")

	// Test HasLookHere returns valid StarBurst frame
	lookHereRune, _ := GetAnimatedIndicator(false, true)
	validLookHereRune := false
	for _, f := range lookHereAnimDef.Frames {
		runes := []rune(f)
		if len(runes) > 0 && runes[0] == lookHereRune {
			validLookHereRune = true
			break
		}
	}
	assert.True(t, validLookHereRune, "expected valid StarBurst frame rune")

	// Test LookHere takes priority over Thinking
	priorityRune, _ := GetAnimatedIndicator(true, true)
	validPriorityRune := false
	for _, f := range lookHereAnimDef.Frames {
		runes := []rune(f)
		if len(runes) > 0 && runes[0] == priorityRune {
			validPriorityRune = true
			break
		}
	}
	assert.True(t, validPriorityRune, "expected LookHere to take priority when both set")
}

func TestFileAnimationSummary_HasAnimation(t *testing.T) {
	// Test no animation
	summary := FileAnimationSummary{}
	assert.False(t, summary.HasAnimation(), "expected no animation for empty summary")

	// Test HasThinking
	summary = FileAnimationSummary{HasThinking: true}
	assert.True(t, summary.HasAnimation(), "expected animation when HasThinking")

	// Test HasLookHere
	summary = FileAnimationSummary{HasLookHere: true}
	assert.True(t, summary.HasAnimation(), "expected animation when HasLookHere")

	// Test both
	summary = FileAnimationSummary{HasThinking: true, HasLookHere: true}
	assert.True(t, summary.HasAnimation(), "expected animation when both set")
}

func TestGetSeparatorFrame(t *testing.T) {
	// Test that GetSeparatorFrame returns a non-empty string
	frame := GetSeparatorFrame()
	assert.True(t, len(frame) > 0, "expected non-empty separator frame")
}

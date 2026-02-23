package server

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/simple-go/assert"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/pkg/critic"
)

func TestGetConversationsSummary_ReturnsEmptyForNoConversations(t *testing.T) {
	messaging := critic.NewDummyMessaging()

	s := &Server{
		config: Config{
			Messaging:     messaging,
			ProjectConfig: testProjectConfig(),
		},
	}

	req := connect.NewRequest(&api.GetConversationsSummaryRequest{})
	resp, err := s.GetConversationsSummary(context.Background(), req)

	assert.NoError(t, err, "GetConversationsSummary should not return error")
	assert.Equals(t, len(resp.Msg.GetSummaries()), 0, "should return empty summaries")
}

func TestGetConversationsSummary_ReturnsSummariesForFiles(t *testing.T) {
	messaging := critic.NewDummyMessaging()
	messaging.Summaries["src/main.go"] = &critic.FileConversationSummary{
		FilePath:              "src/main.go",
		TotalCount:            3,
		UnresolvedCount:       2,
		ResolvedCount:         1,
		HasUnresolvedComments: true,
		HasResolvedComments:   true,
		HasUnreadAIMessages:   false,
	}
	messaging.Summaries["src/utils.go"] = &critic.FileConversationSummary{
		FilePath:              "src/utils.go",
		TotalCount:            1,
		UnresolvedCount:       0,
		ResolvedCount:         1,
		HasUnresolvedComments: false,
		HasResolvedComments:   true,
		HasUnreadAIMessages:   true,
	}

	s := &Server{
		config: Config{
			Messaging:     messaging,
			ProjectConfig: testProjectConfig(),
		},
	}

	req := connect.NewRequest(&api.GetConversationsSummaryRequest{})
	resp, err := s.GetConversationsSummary(context.Background(), req)

	assert.NoError(t, err, "GetConversationsSummary should not return error")
	assert.Equals(t, len(resp.Msg.GetSummaries()), 2, "should return two summaries")

	// Check that both files are present (order may vary)
	summaryMap := make(map[string]*api.FileConversationSummary)
	for _, s := range resp.Msg.GetSummaries() {
		summaryMap[s.GetFilePath()] = s
	}

	mainSummary, ok := summaryMap["src/main.go"]
	assert.True(t, ok, "should have summary for src/main.go")
	assert.Equals(t, mainSummary.GetTotalCount(), int32(3), "total count should be 3")
	assert.Equals(t, mainSummary.GetUnresolvedCount(), int32(2), "unresolved count should be 2")
	assert.Equals(t, mainSummary.GetResolvedCount(), int32(1), "resolved count should be 1")
	assert.Equals(t, mainSummary.GetHasUnreadAiMessages(), false, "should not have unread AI messages")
	assert.Equals(t, mainSummary.GetCategory(), "source", "src/main.go should be source category")

	utilsSummary, ok := summaryMap["src/utils.go"]
	assert.True(t, ok, "should have summary for src/utils.go")
	assert.Equals(t, utilsSummary.GetTotalCount(), int32(1), "total count should be 1")
	assert.Equals(t, utilsSummary.GetUnresolvedCount(), int32(0), "unresolved count should be 0")
	assert.Equals(t, utilsSummary.GetResolvedCount(), int32(1), "resolved count should be 1")
	assert.Equals(t, utilsSummary.GetHasUnreadAiMessages(), true, "should have unread AI messages")
	assert.Equals(t, utilsSummary.GetCategory(), "source", "src/utils.go should be source category")
}

func TestGetConversationsSummary_AnnotatesCategories(t *testing.T) {
	messaging := critic.NewDummyMessaging()
	messaging.Summaries["src/main.go"] = &critic.FileConversationSummary{
		FilePath:   "src/main.go",
		TotalCount: 1,
	}
	messaging.Summaries["src/main_test.go"] = &critic.FileConversationSummary{
		FilePath:   "src/main_test.go",
		TotalCount: 1,
	}
	messaging.Summaries[".gitignore"] = &critic.FileConversationSummary{
		FilePath:   ".gitignore",
		TotalCount: 1,
	}

	s := &Server{
		config: Config{
			Messaging:     messaging,
			ProjectConfig: testProjectConfig(),
		},
	}

	req := connect.NewRequest(&api.GetConversationsSummaryRequest{})
	resp, err := s.GetConversationsSummary(context.Background(), req)

	assert.NoError(t, err, "GetConversationsSummary should not return error")

	summaryMap := make(map[string]*api.FileConversationSummary)
	for _, s := range resp.Msg.GetSummaries() {
		summaryMap[s.GetFilePath()] = s
	}

	assert.Equals(t, summaryMap["src/main.go"].GetCategory(), "source", "src/main.go should be source")
	assert.Equals(t, summaryMap["src/main_test.go"].GetCategory(), "test", "src/main_test.go should be test")
	assert.Equals(t, summaryMap[".gitignore"].GetCategory(), "hidden", ".gitignore should be hidden")
}

func TestGetConversationsSummary_DefaultConfigWhenNil(t *testing.T) {
	messaging := critic.NewDummyMessaging()
	messaging.Summaries["src/main.go"] = &critic.FileConversationSummary{
		FilePath:   "src/main.go",
		TotalCount: 1,
	}

	s := &Server{
		config: Config{
			Messaging:     messaging,
			ProjectConfig: nil, // no project config
		},
	}

	req := connect.NewRequest(&api.GetConversationsSummaryRequest{})
	resp, err := s.GetConversationsSummary(context.Background(), req)

	assert.NoError(t, err, "should not error with nil ProjectConfig")
	assert.Equals(t, len(resp.Msg.GetSummaries()), 1, "should return one summary")
	// With default config, src/main.go should still get "source" category
	assert.Equals(t, resp.Msg.GetSummaries()[0].GetCategory(), "source", "should use default config categorization")
}

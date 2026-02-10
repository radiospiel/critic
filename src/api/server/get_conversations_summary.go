package server

import (
	"context"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/pkg/critic"
	"github.com/samber/lo"
)

// GetConversationsSummary returns conversation counts and status per file path.
func (s *Server) GetConversationsSummary(
	ctx context.Context,
	req *connect.Request[api.GetConversationsSummaryRequest],
) (*connect.Response[api.GetConversationsSummaryResponse], error) {
	response := depanic(func() (*api.GetConversationsSummaryResponse, error) {
		return getConversationsSummaryImpl(s, req.Msg)
	})
	return connect.NewResponse(response), nil
}

func getConversationsSummaryImpl(server *Server, req *api.GetConversationsSummaryRequest) (*api.GetConversationsSummaryResponse, error) {
	m := server.config.Messaging
	criticSummaries, err := m.GetConversationsSummary()
	if err != nil {
		return nil, err
	}
	apiSummaries := lo.Map(criticSummaries, criticToApiFileConversationSummary)

	return &api.GetConversationsSummaryResponse{
		Summaries: apiSummaries,
	}, nil
}

func criticToApiFileConversationSummary(summary *critic.FileConversationSummary, index int) *api.FileConversationSummary {
	return &api.FileConversationSummary{
		FilePath:            summary.FilePath,
		TotalCount:          int32(summary.TotalCount),
		UnresolvedCount:     int32(summary.UnresolvedCount),
		ResolvedCount:       int32(summary.ResolvedCount),
		ExplanationCount:    int32(summary.ExplanationCount),
		HasUnreadAiMessages: summary.HasUnreadAIMessages,
	}
}

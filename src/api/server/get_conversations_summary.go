package server

import (
	"context"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/simple-go/must"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/pkg/critic"
	"github.com/samber/lo"
)

// GetConversationsSummary returns conversation counts and status per file path.
func (s *Server) GetConversationsSummary(
	ctx context.Context,
	req *connect.Request[api.GetConversationsSummaryRequest],
) (*connect.Response[api.GetConversationsSummaryResponse], error) {
	return depanic(func() *connect.Response[api.GetConversationsSummaryResponse] {
		response := getConversationsSummaryImpl(s, req.Msg)
		return connect.NewResponse(response)
	})
}

func getConversationsSummaryImpl(server *Server, req *api.GetConversationsSummaryRequest) *api.GetConversationsSummaryResponse {
	m := server.config.Messaging
	criticSummaries := must.Must2(m.GetAllFileConversationSummaries())
	apiSummaries := lo.Map(criticSummaries, criticToApiFileConversationSummary)

	return &api.GetConversationsSummaryResponse{
		Summaries: apiSummaries,
	}
}

func criticToApiFileConversationSummary(summary *critic.FileConversationSummary, index int) *api.FileConversationSummary {
	return &api.FileConversationSummary{
		FilePath:             summary.FilePath,
		TotalCount:           int32(summary.TotalCount),
		UnresolvedCount:      int32(summary.UnresolvedCount),
		ResolvedCount:        int32(summary.ResolvedCount),
		HasUnreadAiMessages:  summary.HasUnreadAIMessages,
	}
}

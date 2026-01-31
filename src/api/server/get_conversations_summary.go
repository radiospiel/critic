package server

import (
	"context"
	"encoding/json"
	"net/http"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/simple-go/must"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/pkg/critic"
	"github.com/samber/lo"
)

// GetConversationsSummary returns conversation summaries for all files.
// This is a custom endpoint added alongside the Connect-RPC generated handlers.
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
	summaries := must.Must2(m.GetAllConversationsSummary())

	apiSummaries := lo.Map(summaries, func(s *critic.FileConversationSummaryWithCounts, index int) *api.FileConversationSummary {
		return &api.FileConversationSummary{
			FilePath:            s.FilePath,
			TotalCount:          int32(s.TotalCount),
			UnresolvedCount:     int32(s.UnresolvedCount),
			ResolvedCount:       int32(s.ResolvedCount),
			HasUnreadAiMessages: s.HasUnreadAIMessages,
		}
	})

	return &api.GetConversationsSummaryResponse{
		Summaries: apiSummaries,
	}
}

// GetConversationsSummaryHTTPHandler returns an HTTP handler for the GetConversationsSummary endpoint.
// This is needed because the endpoint is not yet in the generated Connect-RPC code.
func (s *Server) GetConversationsSummaryHTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		response := getConversationsSummaryImpl(s, &api.GetConversationsSummaryRequest{})

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

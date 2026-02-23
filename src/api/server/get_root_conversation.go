package server

import (
	"context"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/pkg/critic"
)

// GetRootConversation returns the root conversation if it exists and is unresolved.
func (s *Server) GetRootConversation(
	ctx context.Context,
	req *connect.Request[api.GetRootConversationRequest],
) (*connect.Response[api.GetRootConversationResponse], error) {
	response := depanic(func() (*api.GetRootConversationResponse, error) {
		return getRootConversationImpl(s)
	})
	return connect.NewResponse(response), nil
}

func getRootConversationImpl(server *Server) (*api.GetRootConversationResponse, error) {
	conv, err := server.config.Messaging.LoadRootConversation()
	if err != nil {
		return nil, err
	}

	// Only return the conversation if it's unresolved and has messages beyond the empty root
	if conv == nil || conv.Status == critic.StatusResolved || len(conv.Messages) <= 1 {
		return &api.GetRootConversationResponse{}, nil
	}

	// Check if the only message is the empty root message
	if len(conv.Messages) == 1 && conv.Messages[0].Message == "" {
		return &api.GetRootConversationResponse{}, nil
	}

	return &api.GetRootConversationResponse{
		Conversation: criticToApiConversation(conv, 0, server.categorizeFile),
	}, nil
}

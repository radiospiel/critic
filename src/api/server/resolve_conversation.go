package server

import (
	"context"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/simple-go/logger"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/pkg/critic"
)

// ResolveConversation marks a conversation as resolved.
func (s *Server) ResolveConversation(
	ctx context.Context,
	req *connect.Request[api.ResolveConversationRequest],
) (*connect.Response[api.ResolveConversationResponse], error) {
	response := depanic(func() (*api.ResolveConversationResponse, error) {
		return resolveConversationImpl(s, req.Msg)
	})
	return connect.NewResponse(response), nil
}

func resolveConversationImpl(server *Server, req *api.ResolveConversationRequest) (*api.ResolveConversationResponse, error) {
	logger.Info("ResolveConversation: conversation_id=%s", req.GetConversationId())

	err := server.config.Messaging.MarkConversationAs(req.GetConversationId(), critic.ConversationResolved)
	if err != nil {
		return nil, err
	}

	logger.Info("Resolved conversation %s", req.GetConversationId())
	return &api.ResolveConversationResponse{}, nil
}

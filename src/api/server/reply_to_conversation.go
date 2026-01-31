package server

import (
	"context"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/simple-go/logger"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/pkg/critic"
)

// ReplyToConversation adds a reply message to an existing conversation.
func (s *Server) ReplyToConversation(
	ctx context.Context,
	req *connect.Request[api.ReplyToConversationRequest],
) (*connect.Response[api.ReplyToConversationResponse], error) {
	response := depanic(func() (*api.ReplyToConversationResponse, error) {
		return replyToConversationImpl(s, req.Msg)
	})
	return connect.NewResponse(response), nil
}

func replyToConversationImpl(server *Server, req *api.ReplyToConversationRequest) (*api.ReplyToConversationResponse, error) {
	logger.Info("ReplyToConversation: conversation_id=%s, message=%q",
		req.GetConversationId(),
		req.GetMessage(),
	)

	// Add the reply using the messaging interface
	message, err := server.config.Messaging.ReplyToConversation(
		req.GetConversationId(),
		req.GetMessage(),
		critic.AuthorHuman,
	)
	if err != nil {
		return nil, err
	}

	logger.Info("Added reply %s to conversation %s", message.UUID, req.GetConversationId())
	return &api.ReplyToConversationResponse{
		Success: true,
	}, nil
}

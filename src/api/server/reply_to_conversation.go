package server

import (
	"context"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/simple-go/logger"
	"github.com/radiospiel/critic/simple-go/must"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/pkg/critic"
)

// ReplyToConversation adds a reply message to an existing conversation.
func (s *Server) ReplyToConversation(
	ctx context.Context,
	req *connect.Request[api.ReplyToConversationRequest],
) (*connect.Response[api.ReplyToConversationResponse], error) {
	return depanic(func() *connect.Response[api.ReplyToConversationResponse] {
		response := replyToConversationImpl(s, req.Msg)
		return connect.NewResponse(response)
	})
}

func replyToConversationImpl(server *Server, req *api.ReplyToConversationRequest) *api.ReplyToConversationResponse {
	logger.Info("ReplyToConversation: conversation_id=%s, message=%q",
		req.GetConversationId(),
		req.GetMessage(),
	)

	// Add the reply using the messaging interface
	message := must.Must2(server.config.Messaging.ReplyToConversation(
		req.GetConversationId(),
		req.GetMessage(),
		critic.AuthorHuman,
	))

	logger.Info("Added reply %s to conversation %s", message.UUID, req.GetConversationId())
	return &api.ReplyToConversationResponse{
		Success: true,
	}
}

package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/simple-go/logger"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/pkg/critic"
)

// MarkConversationAs updates the status of a conversation.
func (s *Server) MarkConversationAs(
	ctx context.Context,
	req *connect.Request[api.MarkConversationAsRequest],
) (*connect.Response[api.MarkConversationAsResponse], error) {
	response := depanic(func() (*api.MarkConversationAsResponse, error) {
		return markConversationAsImpl(s, req.Msg)
	})
	return connect.NewResponse(response), nil
}

func apiStatusToConversationUpdate(status api.ConversationStatus) (critic.ConversationUpdate, error) {
	switch status {
	case api.ConversationStatus_CONVERSATION_STATUS_RESOLVED:
		return critic.ConversationResolved, nil
	case api.ConversationStatus_CONVERSATION_STATUS_UNRESOLVED:
		return critic.ConversationUnresolved, nil
	case api.ConversationStatus_CONVERSATION_STATUS_ARCHIVED:
		return critic.ConversationArchived, nil
	default:
		return "", fmt.Errorf("unsupported status transition: %s", status)
	}
}

func markConversationAsImpl(server *Server, req *api.MarkConversationAsRequest) (*api.MarkConversationAsResponse, error) {
	logger.Info("MarkConversationAs: conversation_id=%s status=%s", req.GetConversationId(), req.GetStatus())

	update, err := apiStatusToConversationUpdate(req.GetStatus())
	if err != nil {
		return nil, err
	}

	err = server.config.Messaging.MarkConversationAs(req.GetConversationId(), update)
	if err != nil {
		return nil, err
	}

	logger.Info("Marked conversation %s as %s", req.GetConversationId(), update)
	return &api.MarkConversationAsResponse{}, nil
}

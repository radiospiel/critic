package server

import (
	"context"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/pkg/critic"
	"github.com/samber/lo"
)

// GetConversations returns all conversations for a file at a specific path.
func (s *Server) GetConversations(
	ctx context.Context,
	req *connect.Request[api.GetConversationsRequest],
) (*connect.Response[api.GetConversationsResponse], error) {
	response := depanic(func() (*api.GetConversationsResponse, error) {
		return getConversationsImpl(s, req.Msg)
	})
	return connect.NewResponse(response), nil
}

func getConversationsImpl(server *Server, req *api.GetConversationsRequest) (*api.GetConversationsResponse, error) {
	path := req.GetPath()

	m := server.config.Messaging
	criticConversations, err := m.GetConversationsForFile(path)
	if err != nil {
		return nil, err
	}
	apiConversations := lo.Map(criticConversations, criticToApiConversation)

	return &api.GetConversationsResponse{
		Conversations: apiConversations,
	}, nil
}

func criticToApiMessage(msg critic.Message, index int) *api.Message {
	return &api.Message{
		Id:        msg.UUID,
		Author:    string(msg.Author),
		Content:   msg.Message,
		CreatedAt: msg.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: msg.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		IsUnread:  msg.IsUnread,
	}
}

func criticStatusToApiStatus(status critic.ConversationStatus) api.ConversationStatus {
	switch status {
	case critic.StatusResolved:
		return api.ConversationStatus_CONVERSATION_STATUS_RESOLVED
	case critic.StatusUnresolved:
		return api.ConversationStatus_CONVERSATION_STATUS_UNRESOLVED
	case critic.StatusActive:
		return api.ConversationStatus_CONVERSATION_STATUS_ACTIVE
	case critic.StatusWaitingForResponse:
		return api.ConversationStatus_CONVERSATION_STATUS_WAITING_FOR_RESPONSE
	default:
		return api.ConversationStatus_CONVERSATION_STATUS_INVALID
	}
}

func criticToApiConversation(conv *critic.Conversation, index int) *api.Conversation {
	messages := lo.Map(conv.Messages, criticToApiMessage)
	return &api.Conversation{
		Id:          conv.UUID,
		Status:      criticStatusToApiStatus(conv.Status),
		FilePath:    conv.FilePath,
		LineNumber:  int32(conv.LineNumber),
		CodeVersion: conv.CodeVersion,
		Context:     conv.Context,
		Messages:    messages,
		CreatedAt:   conv.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   conv.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

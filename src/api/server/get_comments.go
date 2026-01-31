package server

import (
	"context"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/simple-go/must"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/pkg/critic"
	"github.com/samber/lo"
)

// TODO(bot) reimplement GRPC implementations following the pattern of GetComments:
// - set up a xxxImpl function which accepts the server and the request, and that
//   returns the response, and that panics on error.
// - have a generic wrapper that depanics the call to the impl.

// TODO(bot) adjust the webui to fetch comments from the grpc call.

// GetComments returns comments on a file at a specific path.
func (s *Server) GetComments(
	ctx context.Context,
	req *connect.Request[api.GetCommentsRequest],
) (*connect.Response[api.GetCommentsResponse], error) {
	return depanic(func() *connect.Response[api.GetCommentsResponse] {
		response := getCommentsImpl(s, req.Msg)
		return connect.NewResponse(response)
	})
}

func getCommentsImpl(server *Server, req *api.GetCommentsRequest) *api.GetCommentsResponse {
	path := req.GetPath()

	m := server.config.Messaging
	criticConversations := must.Must2(m.GetConversationsForFile(path))
	apiConversations := lo.Map(criticConversations, criticToApiConversation)

	return &api.GetCommentsResponse{
		Conversations: apiConversations,
	}
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

func criticToApiConversation(conv *critic.Conversation, index int) *api.Conversation {
	messages := lo.Map(conv.Messages, criticToApiMessage)
	return &api.Conversation{
		Id:          conv.UUID,
		Status:      string(conv.Status),
		FilePath:    conv.FilePath,
		LineNumber:  int32(conv.LineNumber),
		CodeVersion: conv.CodeVersion,
		Context:     conv.Context,
		Messages:    messages,
		CreatedAt:   conv.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   conv.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

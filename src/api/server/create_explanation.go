package server

import (
	"context"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/src/api"
)

// CreateExplanation creates a new explanation (informal annotation) on a code line.
// It delegates to createConversationImpl with ConversationType set to EXPLANATION.
func (s *Server) CreateExplanation(
	ctx context.Context,
	req *connect.Request[api.CreateExplanationRequest],
) (*connect.Response[api.CreateExplanationResponse], error) {
	response := depanic(func() (*api.CreateExplanationResponse, error) {
		return createExplanationImpl(s, req.Msg)
	})
	return connect.NewResponse(response), nil
}

func createExplanationImpl(server *Server, req *api.CreateExplanationRequest) (*api.CreateExplanationResponse, error) {
	// Delegate to createConversationImpl with the explanation type
	createReq := &api.CreateConversationRequest{
		NewFile:          req.GetFile(),
		NewLine:          req.GetLine(),
		Comment:          req.GetComment(),
		ConversationType: api.ConversationType_CONVERSATION_TYPE_EXPLANATION,
	}

	result, err := createConversationImpl(server, createReq)
	if err != nil {
		return nil, err
	}

	return &api.CreateExplanationResponse{
		Success: result.GetSuccess(),
	}, nil
}

package server

import (
	"context"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/simple-go/logger"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/pkg/critic"
)

// CreateExplanation creates a new explanation (informal annotation) on a code line.
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
	logger.Info("CreateExplanation: file=%s, line=%d, comment=%q",
		req.GetFile(),
		req.GetLine(),
		req.GetComment(),
	)

	filePath := req.GetFile()
	lineNo := int(req.GetLine())

	// Get the current commit SHA from the session
	codeVersion := server.session.HeadCommit()

	// Create the explanation using the messaging interface
	conversation, err := server.config.Messaging.CreateExplanation(
		critic.AuthorHuman,
		req.GetComment(),
		filePath,
		lineNo,
		codeVersion,
		"", // context
	)
	if err != nil {
		return nil, err
	}

	logger.Info("Created explanation %s at %s:%d", conversation.UUID, filePath, lineNo)
	return &api.CreateExplanationResponse{
		Success: true,
	}, nil
}

package server

import (
	"context"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/simple-go/logger"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/pkg/critic"
)

// CreateConversation creates a new conversation (comment thread) on a diff line.
func (s *Server) CreateConversation(
	ctx context.Context,
	req *connect.Request[api.CreateConversationRequest],
) (*connect.Response[api.CreateConversationResponse], error) {
	response := depanic(func() (*api.CreateConversationResponse, error) {
		return createConversationImpl(s, req.Msg)
	})
	return connect.NewResponse(response), nil
}

func createConversationImpl(server *Server, req *api.CreateConversationRequest) (*api.CreateConversationResponse, error) {
	logger.Info("CreateConversation: old_file=%s, old_line=%d, new_file=%s, new_line=%d, comment=%q",
		req.GetOldFile(),
		req.GetOldLine(),
		req.GetNewFile(),
		req.GetNewLine(),
		req.GetComment(),
	)

	// Determine file path and line number to use
	// Prefer new_file/new_line for added/modified lines, fall back to old_file/old_line for deleted lines
	// Note: Validation (comment required, file paths, line numbers >= 0) is handled by JSON schema
	filePath := req.GetNewFile()
	lineNo := int(req.GetNewLine())
	if filePath == "" || lineNo == 0 {
		filePath = req.GetOldFile()
		lineNo = int(req.GetOldLine())
	}

	// Get the current commit SHA from the session
	codeVersion := server.session.HeadCommit()

	// Create the conversation using the messaging interface
	conversation, err := server.config.Messaging.CreateConversation(
		critic.AuthorHuman,
		req.GetComment(),
		filePath,
		lineNo,
		codeVersion,
		"", // context - could be enhanced to include surrounding code
	)
	if err != nil {
		return nil, err
	}

	logger.Info("Created conversation %s at %s:%d", conversation.UUID, filePath, lineNo)
	return &api.CreateConversationResponse{
		Success: true,
	}, nil
}

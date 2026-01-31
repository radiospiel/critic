package server

import (
	"context"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/simple-go/logger"
	"github.com/radiospiel/critic/simple-go/must"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/pkg/critic"
)

// CreateComment creates a new comment on a diff line.
func (s *Server) CreateComment(
	ctx context.Context,
	req *connect.Request[api.CreateCommentRequest],
) (*connect.Response[api.CreateCommentResponse], error) {
	return depanic(func() *connect.Response[api.CreateCommentResponse] {
		response := createCommentImpl(s, req.Msg)
		return connect.NewResponse(response)
	})
}

func createCommentImpl(server *Server, req *api.CreateCommentRequest) *api.CreateCommentResponse {
	logger.Info("CreateComment: old_file=%s, old_line=%d, new_file=%s, new_line=%d, comment=%q",
		req.GetOldFile(),
		req.GetOldLine(),
		req.GetNewFile(),
		req.GetNewLine(),
		req.GetComment(),
	)

	// Determine file path and line number to use
	// Prefer new_file/new_line for added/modified lines, fall back to old_file/old_line for deleted lines
	// Note: Basic validation (comment required, at least one file path) is handled by JSON schema
	filePath := req.GetNewFile()
	lineNo := int(req.GetNewLine())
	if filePath == "" || lineNo == 0 {
		filePath = req.GetOldFile()
		lineNo = int(req.GetOldLine())
	}

	// Validate derived line number - JSON schema validates basic fields,
	// but the derived line number from the fallback logic needs manual validation
	if lineNo <= 0 {
		return &api.CreateCommentResponse{
			Success: false,
			Error:   api.InvalidArgument("line number must be positive"),
		}
	}

	// Get the current commit SHA from the session
	codeVersion := server.session.HeadCommit()

	// Create the conversation using the messaging interface
	conversation := must.Must2(server.config.Messaging.CreateConversation(
		critic.AuthorHuman,
		req.GetComment(),
		filePath,
		lineNo,
		codeVersion,
		"", // context - could be enhanced to include surrounding code
	))

	logger.Info("Created comment %s at %s:%d", conversation.UUID, filePath, lineNo)
	return &api.CreateCommentResponse{
		Success: true,
	}
}

package server

import (
	"context"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/simple-go/logger"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/pkg/critic"
)

// CreateComment creates a new comment on a diff line.
func (s *Server) CreateComment(
	ctx context.Context,
	req *connect.Request[api.CreateCommentRequest],
) (*connect.Response[api.CreateCommentResponse], error) {
	logger.Info("CreateComment: old_file=%s, old_line=%d, new_file=%s, new_line=%d, comment=%q",
		req.Msg.GetOldFile(),
		req.Msg.GetOldLine(),
		req.Msg.GetNewFile(),
		req.Msg.GetNewLine(),
		req.Msg.GetComment(),
	)

	// Determine file path and line number to use
	// Prefer new_file/new_line for added/modified lines, fall back to old_file/old_line for deleted lines
	filePath := req.Msg.GetNewFile()
	lineNo := int(req.Msg.GetNewLine())
	if filePath == "" || lineNo == 0 {
		filePath = req.Msg.GetOldFile()
		lineNo = int(req.Msg.GetOldLine())
	}

	// Validate required fields
	// TODO(bot): replace this validation by a validation with a JSON schema.
	if filePath == "" {
		return connect.NewResponse(&api.CreateCommentResponse{
			Success: false,
			Error:   api.InvalidArgument("file path is required"),
		}), nil
	}
	if lineNo <= 0 {
		return connect.NewResponse(&api.CreateCommentResponse{
			Success: false,
			Error:   api.InvalidArgument("line number must be positive"),
		}), nil
	}
	if req.Msg.GetComment() == "" {
		return connect.NewResponse(&api.CreateCommentResponse{
			Success: false,
			Error:   api.InvalidArgument("comment is required"),
		}), nil
	}

	// Get the current commit SHA from the session
	codeVersion := s.session.HeadCommit()

	// Create the conversation using the messaging interface
	conversation, err := s.config.Messaging.CreateConversation(
		critic.AuthorHuman,
		req.Msg.GetComment(),
		filePath,
		lineNo,
		codeVersion,
		"", // context - could be enhanced to include surrounding code
	)
	if err != nil {
		logger.Error("Failed to create comment: %v", err)
		return connect.NewResponse(&api.CreateCommentResponse{
			Success: false,
			Error:   api.InternalError("failed to create comment: " + err.Error()),
		}), nil
	}

	logger.Info("Created comment %s at %s:%d", conversation.UUID, filePath, lineNo)
	res := connect.NewResponse(&api.CreateCommentResponse{
		Success: true,
	})
	return res, nil
}

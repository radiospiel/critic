package server

import (
	"context"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/simple-go/logger"
	"github.com/radiospiel/critic/src/api"
)

// CreateComment creates a new comment on a diff line.
// Currently, it just logs the comment without persisting it.
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

	res := connect.NewResponse(&api.CreateCommentResponse{
		Success: true,
	})
	return res, nil
}

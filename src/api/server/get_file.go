package server

import (
	"context"
	"path/filepath"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/git"
)

// GetFile returns the content of a file at a specific path.
func (s *Server) GetFile(
	ctx context.Context,
	req *connect.Request[api.GetFileRequest],
) (*connect.Response[api.GetFileResponse], error) {
	path := req.Msg.GetPath()

	// Resolve path relative to git root
	fullPath := filepath.Join(s.config.GitRoot, path)

	// Read file content from working directory
	content, err := git.GetFileContent(fullPath, "")
	if err != nil {
		res := connect.NewResponse(&api.GetFileResponse{
			Error: api.NotFound("file not found: " + path),
		})
		return res, nil
	}

	res := connect.NewResponse(&api.GetFileResponse{
		Content: content,
	})
	return res, nil
}

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
	return depanic(func() *connect.Response[api.GetFileResponse] {
		response := getFileImpl(s, req.Msg)
		return connect.NewResponse(response)
	})
}

func getFileImpl(server *Server, req *api.GetFileRequest) *api.GetFileResponse {
	path := req.GetPath()

	// Resolve path relative to git root
	fullPath := filepath.Join(server.config.GitRoot, path)

	// Read file content from working directory
	content, err := git.GetFileContent(fullPath, "")
	if err != nil {
		return &api.GetFileResponse{
			Error: api.NotFound("file not found: " + path),
		}
	}

	return &api.GetFileResponse{
		Content: content,
	}
}

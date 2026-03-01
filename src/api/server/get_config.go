package server

import (
	"context"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/src/api"
)

// GetConfig returns server configuration for the frontend.
func (s *Server) GetConfig(ctx context.Context, req *connect.Request[api.GetConfigRequest]) (*connect.Response[api.GetConfigResponse], error) {
	return connect.NewResponse(depanic(func() (*api.GetConfigResponse, error) {
		return &api.GetConfigResponse{
			GitRoot: s.config.GitRoot,
			DevMode: s.config.Dev,
		}, nil
	})), nil
}

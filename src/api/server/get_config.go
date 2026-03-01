package server

import (
	"context"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/messagedb"
)

// GetConfig returns server configuration for the frontend.
func (s *Server) GetConfig(ctx context.Context, req *connect.Request[api.GetConfigRequest]) (*connect.Response[api.GetConfigResponse], error) {
	return connect.NewResponse(depanic(func() (*api.GetConfigResponse, error) {
		resp := &api.GetConfigResponse{
			GitRoot: s.config.GitRoot,
			DevMode: s.config.Dev,
		}

		// Include Claude session ID if available
		if db, ok := s.config.Messaging.(*messagedb.DB); ok {
			if sessionID, err := db.GetSetting("claude_session_id"); err == nil {
				resp.ClaudeSessionId = sessionID
			}
		}

		return resp, nil
	})), nil
}

package server

import (
	"context"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/messagedb"
)

// SetClaudeSession stores the Claude Code session ID in the database.
func (s *Server) SetClaudeSession(ctx context.Context, req *connect.Request[api.SetClaudeSessionRequest]) (*connect.Response[api.SetClaudeSessionResponse], error) {
	return connect.NewResponse(depanic(func() (*api.SetClaudeSessionResponse, error) {
		db, ok := s.config.Messaging.(*messagedb.DB)
		if !ok {
			return nil, api.UnavailableError("database not available")
		}

		if err := db.SetSetting("claude_session_id", req.Msg.SessionId); err != nil {
			return nil, err
		}

		return &api.SetClaudeSessionResponse{Success: true}, nil
	})), nil
}

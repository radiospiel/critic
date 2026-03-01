package server

import (
	"context"
	"fmt"
	"os/exec"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/simple-go/logger"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/messagedb"
)

// InjectPrompt sends a prompt to the connected Claude Code session.
func (s *Server) InjectPrompt(ctx context.Context, req *connect.Request[api.InjectPromptRequest]) (*connect.Response[api.InjectPromptResponse], error) {
	return connect.NewResponse(depanic(func() (*api.InjectPromptResponse, error) {
		db, ok := s.config.Messaging.(*messagedb.DB)
		if !ok {
			return nil, api.UnavailableError("database not available")
		}

		sessionID, err := db.GetSetting("claude_session_id")
		if err != nil || sessionID == "" {
			return nil, api.UnavailableError("no Claude Code session connected", "run /critic:activate in Claude Code first")
		}

		prompt := req.Msg.Prompt
		if prompt == "" {
			return nil, api.InvalidArgumentError("prompt is required")
		}

		cmd := exec.CommandContext(ctx, "claude", "--resume", sessionID, "-p", prompt, "--output-format", "json")
		logger.Info("Running: %s", cmd.String())
		output, err := cmd.CombinedOutput()
		if err != nil {
			logger.Error("Claude prompt injection failed: %v, output: %s", err, string(output))
			return nil, api.InternalServerError(fmt.Sprintf("claude command failed: %v", err), string(output))
		}

		return &api.InjectPromptResponse{
			Success: true,
			Output:  string(output),
		}, nil
	})), nil
}

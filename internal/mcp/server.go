package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

const (
	// ServerName is the name of the MCP server
	ServerName = "critic-hitl"
	// ServerVersion is the version of the MCP server
	ServerVersion = "1.0.0"
	// ProtocolVersion is the MCP protocol version we support
	ProtocolVersion = "2024-11-05"
	// DefaultFeedbackTimeout is the default timeout for waiting for feedback
	DefaultFeedbackTimeout = 5 * time.Minute
)

// Server represents the MCP server for HITL interactions
type Server struct {
	reader          *bufio.Reader
	writer          io.Writer
	reviewerIPC     *ReviewerIPC
	feedbackTimeout time.Duration
	initialized     bool
}

// NewServer creates a new MCP server
func NewServer(socketPath string) *Server {
	return &Server{
		reader:          bufio.NewReader(os.Stdin),
		writer:          os.Stdout,
		reviewerIPC:     NewReviewerIPC(socketPath),
		feedbackTimeout: DefaultFeedbackTimeout,
	}
}

// SetFeedbackTimeout sets the timeout for waiting for reviewer feedback
func (s *Server) SetFeedbackTimeout(timeout time.Duration) {
	s.feedbackTimeout = timeout
}

// Run starts the MCP server and processes messages
func (s *Server) Run() error {
	// Start IPC server for reviewer communication
	if err := s.reviewerIPC.Start(); err != nil {
		return fmt.Errorf("failed to start reviewer IPC: %w", err)
	}
	defer s.reviewerIPC.Stop()

	s.logToStderr("HITL MCP server started, socket: %s", s.reviewerIPC.GetSocketPath())

	// Process messages from stdin
	for {
		line, err := s.reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				s.logToStderr("EOF received, shutting down")
				return nil
			}
			return fmt.Errorf("failed to read message: %w", err)
		}

		if len(line) == 0 {
			continue
		}

		if err := s.handleMessage(line); err != nil {
			s.logToStderr("Error handling message: %v", err)
		}
	}
}

// handleMessage processes a single JSON-RPC message
func (s *Server) handleMessage(data []byte) error {
	var req Request
	if err := json.Unmarshal(data, &req); err != nil {
		return s.sendError(nil, ParseError, "Invalid JSON", nil)
	}

	if req.JSONRPC != "2.0" {
		return s.sendError(req.ID, InvalidRequest, "Invalid JSON-RPC version", nil)
	}

	s.logToStderr("Received request: %s", req.Method)

	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "initialized":
		// Notification, no response needed
		return nil
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(req)
	case "ping":
		return s.sendResult(req.ID, map[string]interface{}{})
	default:
		return s.sendError(req.ID, MethodNotFound, fmt.Sprintf("Unknown method: %s", req.Method), nil)
	}
}

// handleInitialize handles the initialize request
func (s *Server) handleInitialize(req Request) error {
	result := InitializeResult{
		ProtocolVersion: ProtocolVersion,
		Capabilities: Capabilities{
			Tools: &ToolsCapability{},
		},
		ServerInfo: ServerInfo{
			Name:    ServerName,
			Version: ServerVersion,
		},
	}

	s.initialized = true
	return s.sendResult(req.ID, result)
}

// handleToolsList returns the list of available tools
func (s *Server) handleToolsList(req Request) error {
	tools := []Tool{
		{
			Name:        "get_review_feedback",
			Description: "Wait for and retrieve feedback from the human reviewer. Call this before completing significant changes. The tool will block until the reviewer responds or timeout occurs.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"summary": {
						Type:        "string",
						Description: "Brief summary of what you've done for the reviewer to review",
					},
				},
				Required: []string{"summary"},
			},
		},
		{
			Name:        "notify_reviewer",
			Description: "Send a notification to the reviewer without waiting for a response. Use this for status updates or non-blocking communication.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"message": {
						Type:        "string",
						Description: "The message to send to the reviewer",
					},
				},
				Required: []string{"message"},
			},
		},
	}

	return s.sendResult(req.ID, ToolsListResult{Tools: tools})
}

// handleToolsCall handles a tool call request
func (s *Server) handleToolsCall(req Request) error {
	// Parse params
	paramsJSON, err := json.Marshal(req.Params)
	if err != nil {
		return s.sendError(req.ID, InvalidParams, "Invalid params", nil)
	}

	var params CallToolParams
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		return s.sendError(req.ID, InvalidParams, "Invalid params", nil)
	}

	s.logToStderr("Tool call: %s", params.Name)

	switch params.Name {
	case "get_review_feedback":
		return s.handleGetReviewFeedback(req, params)
	case "notify_reviewer":
		return s.handleNotifyReviewer(req, params)
	default:
		return s.sendToolError(req.ID, fmt.Sprintf("Unknown tool: %s", params.Name))
	}
}

// handleGetReviewFeedback handles the get_review_feedback tool
func (s *Server) handleGetReviewFeedback(req Request, params CallToolParams) error {
	summary, _ := params.Arguments["summary"].(string)
	if summary == "" {
		return s.sendToolError(req.ID, "summary is required")
	}

	s.logToStderr("Awaiting review for: %s", summary)

	// Notify reviewers that we're waiting
	s.reviewerIPC.NotifyReviewer(NotificationMessage{
		Type:    "waiting",
		Summary: summary,
	})

	// Wait for feedback
	msg, err := s.reviewerIPC.WaitForFeedback(s.feedbackTimeout)
	if err != nil {
		return s.sendToolResult(req.ID, fmt.Sprintf("No feedback received: %v", err))
	}

	// Format response based on message type
	var responseText string
	switch msg.Type {
	case "approved":
		responseText = "APPROVED: " + msg.Feedback
	case "rejected":
		responseText = "REJECTED: " + msg.Feedback
	default:
		responseText = msg.Feedback
	}

	return s.sendToolResult(req.ID, responseText)
}

// handleNotifyReviewer handles the notify_reviewer tool
func (s *Server) handleNotifyReviewer(req Request, params CallToolParams) error {
	message, _ := params.Arguments["message"].(string)
	if message == "" {
		return s.sendToolError(req.ID, "message is required")
	}

	s.logToStderr("Notification: %s", message)

	// Send notification to reviewers
	s.reviewerIPC.NotifyReviewer(NotificationMessage{
		Type:    "notification",
		Summary: message,
	})

	return s.sendToolResult(req.ID, "Reviewer notified")
}

// sendResult sends a successful result response
func (s *Server) sendResult(id interface{}, result interface{}) error {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	return s.sendResponse(resp)
}

// sendError sends an error response
func (s *Server) sendError(id interface{}, code int, message string, data interface{}) error {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &Error{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	return s.sendResponse(resp)
}

// sendToolResult sends a successful tool result
func (s *Server) sendToolResult(id interface{}, text string) error {
	result := CallToolResult{
		Content: []ContentBlock{TextContent(text)},
	}
	return s.sendResult(id, result)
}

// sendToolError sends a tool error result
func (s *Server) sendToolError(id interface{}, message string) error {
	result := CallToolResult{
		Content: []ContentBlock{TextContent(message)},
		IsError: true,
	}
	return s.sendResult(id, result)
}

// sendResponse sends a JSON-RPC response
func (s *Server) sendResponse(resp Response) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	_, err = fmt.Fprintf(s.writer, "%s\n", data)
	return err
}

// logToStderr logs a message to stderr (visible to the user, not to MCP client)
func (s *Server) logToStderr(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[HITL] "+format+"\n", args...)
}

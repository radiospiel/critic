package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"git.15b.it/eno/critic/internal/git"
	"git.15b.it/eno/critic/internal/logger"
	"git.15b.it/eno/critic/internal/messagedb"
	"git.15b.it/eno/critic/pkg/critic"
)

const (
	// ServerName is the name of the MCP server
	ServerName = "critic-hitl"
	// ServerVersion is the version of the MCP server
	ServerVersion = "1.0.0"
	// ProtocolVersion is the MCP protocol version we support
	ProtocolVersion = "2024-11-05"
)

// Server represents the MCP server for HITL interactions
type Server struct {
	reader      *bufio.Reader
	writer      io.Writer
	messaging   critic.Messaging
	initialized bool
}

// NewServer creates a new MCP server
func NewServer() *Server {
	// Initialize message database
	gitRoot, err := git.GetGitRoot()
	if err != nil {
		logger.Error("Failed to get git root: %v", err)
		return nil
	}

	mdb, err := messagedb.New(gitRoot)
	if err != nil {
		logger.Error("Failed to initialize message database: %v", err)
		return nil
	}

	return &Server{
		reader:    bufio.NewReader(os.Stdin),
		writer:    os.Stdout,
		messaging: mdb,
	}
}

// SetMessaging sets the messaging interface
func (s *Server) SetMessaging(messaging critic.Messaging) {
	s.messaging = messaging
}

// Run starts the MCP server and processes messages
func (s *Server) Run() error {
	s.logToStderr("HITL MCP server started")

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
			Name:        "get_critic_conversations",
			Description: "Get a list of conversation UUIDs. Optionally filter by status ('unresolved' or 'resolved'). Use this to check for reviewer feedback.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"status": {
						Type:        "string",
						Description: "Optional filter: 'unresolved' or 'resolved'. If omitted, returns all conversations.",
					},
				},
			},
		},
		{
			Name:        "get_full_critic_conversation",
			Description: "Get the complete conversation including all messages and replies. Returns conversation metadata and all messages ordered chronologically.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"uuid": {
						Type:        "string",
						Description: "The UUID of the conversation to retrieve",
					},
				},
				Required: []string{"uuid"},
			},
		},
		{
			Name:        "reply_to_critic_conversation",
			Description: "Add a reply to an existing conversation. Use this to respond to reviewer feedback.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"uuid": {
						Type:        "string",
						Description: "The UUID of the conversation to reply to",
					},
					"message": {
						Type:        "string",
						Description: "Your reply message",
					},
				},
				Required: []string{"uuid", "message"},
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
	case "get_critic_conversations":
		return s.handleGetCriticConversations(req, params)
	case "get_full_critic_conversation":
		return s.handleGetFullCriticConversation(req, params)
	case "reply_to_critic_conversation":
		return s.handleReplyToCriticConversation(req, params)
	default:
		return s.sendToolError(req.ID, fmt.Sprintf("Unknown tool: %s", params.Name))
	}
}

// handleGetCriticConversations handles the get_critic_conversations tool
func (s *Server) handleGetCriticConversations(req Request, params CallToolParams) error {
	if s.messaging == nil {
		s.logToStderr("Messaging not initialized")
		return s.sendToolResult(req.ID, "[]")
	}

	// Get status filter (optional)
	status, _ := params.Arguments["status"].(string)
	s.logToStderr("Getting conversations with status filter: %s", status)

	conversations, err := s.messaging.GetConversations(status)
	if err != nil {
		s.logToStderr("Failed to get conversations: %v", err)
		return s.sendToolError(req.ID, fmt.Sprintf("Error getting conversations: %v", err))
	}

	s.logToStderr("Found %d conversations", len(conversations))

	// Extract UUIDs for the response
	uuids := make([]string, len(conversations))
	for i, conv := range conversations {
		uuids[i] = conv.UUID
	}

	// Format as JSON array
	result, err := json.Marshal(uuids)
	if err != nil {
		return s.sendToolError(req.ID, fmt.Sprintf("Error encoding result: %v", err))
	}

	return s.sendToolResult(req.ID, string(result))
}

// handleGetFullCriticConversation handles the get_full_critic_conversation tool
func (s *Server) handleGetFullCriticConversation(req Request, params CallToolParams) error {
	if s.messaging == nil {
		s.logToStderr("Messaging not initialized")
		return s.sendToolError(req.ID, "Messaging not initialized")
	}

	uuid, ok := params.Arguments["uuid"].(string)
	if !ok || uuid == "" {
		return s.sendToolError(req.ID, "uuid is required")
	}

	s.logToStderr("Getting full conversation: %s", uuid)

	conversation, err := s.messaging.GetFullConversation(uuid)
	if err != nil {
		s.logToStderr("Failed to get conversation: %v", err)
		return s.sendToolError(req.ID, fmt.Sprintf("Error getting conversation: %v", err))
	}

	// Format conversation as human-readable text
	response := s.formatConversation(conversation)
	s.logToStderr("Returning conversation with %d messages", len(conversation.Messages))

	return s.sendToolResult(req.ID, response)
}

// handleReplyToCriticConversation handles the reply_to_critic_conversation tool
func (s *Server) handleReplyToCriticConversation(req Request, params CallToolParams) error {
	if s.messaging == nil {
		s.logToStderr("Messaging not initialized")
		return s.sendToolError(req.ID, "Messaging not initialized")
	}

	uuid, ok := params.Arguments["uuid"].(string)
	if !ok || uuid == "" {
		return s.sendToolError(req.ID, "uuid is required")
	}

	message, ok := params.Arguments["message"].(string)
	if !ok || message == "" {
		return s.sendToolError(req.ID, "message is required")
	}

	s.logToStderr("Adding reply to conversation %s", uuid)

	reply, err := s.messaging.ReplyToConversation(uuid, message, critic.AuthorAI)
	if err != nil {
		s.logToStderr("Failed to create reply: %v", err)
		return s.sendToolError(req.ID, fmt.Sprintf("Error creating reply: %v", err))
	}

	s.logToStderr("Created reply: %s", reply.UUID)
	return s.sendToolResult(req.ID, fmt.Sprintf("Reply created successfully: %s", reply.UUID))
}

// formatConversation formats a conversation for display
func (s *Server) formatConversation(conv *critic.Conversation) string {
	var builder strings.Builder

	// Header with metadata
	builder.WriteString(fmt.Sprintf("Conversation: %s\n", conv.UUID))
	builder.WriteString(fmt.Sprintf("Status: %s\n", conv.Status))
	builder.WriteString(fmt.Sprintf("Location: %s:%d\n", conv.FilePath, conv.LineNumber))
	builder.WriteString(fmt.Sprintf("Code Version: %s\n", conv.CodeVersion))
	builder.WriteString("\n")

	// Messages
	for i, msg := range conv.Messages {
		if i > 0 {
			builder.WriteString("\n")
		}

		prefix := "human"
		if msg.Author == critic.AuthorAI {
			prefix = "ai"
		}

		builder.WriteString(fmt.Sprintf("[%s] %s\n", prefix, msg.CreatedAt.Format("2006-01-02 15:04:05")))
		builder.WriteString(msg.Message)
		builder.WriteString("\n")
	}

	return builder.String()
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

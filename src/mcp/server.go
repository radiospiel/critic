package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/radiospiel/critic/simple-go/logger"
	"github.com/radiospiel/critic/src/git"
	"github.com/radiospiel/critic/src/messagedb"
	"github.com/radiospiel/critic/src/pkg/critic"
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
	gitRoot := git.GetGitRoot()

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
	case "prompts/list":
		return s.handlePromptsList(req)
	case "prompts/get":
		return s.handlePromptsGet(req)
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
			Tools:   &ToolsCapability{},
			Prompts: &PromptsCapability{},
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
			Description: "Get a list of conversation UUIDs. Optionally filter by status ('unresolved', 'resolved', or 'actionable'). Use this to check for reviewer feedback. The 'actionable' filter returns only unresolved conversations where the last message is from a human reviewer (i.e., conversations that need AI attention).",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"status": {
						Type:        "string",
						Description: "Optional filter: 'unresolved', 'resolved', or 'actionable'. 'actionable' returns unresolved conversations where the last message is from a human. If omitted, returns all conversations.",
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
		{
			Name:        "critic_announce",
			Description: "Post an announcement visible in the Critic UI. Creates a message on the root conversation and marks it as unresolved.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"message": {
						Type:        "string",
						Description: "The announcement message",
					},
				},
				Required: []string{"message"},
			},
		},
		{
			Name:        "critic_explain",
			Description: "Post an explanation on a specific code line. Explanations are informal annotations shown with a lightbulb icon in the Critic UI.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"file": {
						Type:        "string",
						Description: "The git-relative file path",
					},
					"line": {
						Type:        "number",
						Description: "The line number in the file",
					},
					"comment": {
						Type:        "string",
						Description: "The explanation text",
					},
				},
				Required: []string{"file", "line", "comment"},
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
	case "critic_announce":
		return s.handleCriticAnnounce(req, params)
	case "critic_explain":
		return s.handleCriticExplain(req, params)
	default:
		return s.sendToolError(req.ID, fmt.Sprintf("Unknown tool: %s", params.Name))
	}
}

// ConversationSummary represents a summary of a conversation for the MCP list response
type ConversationSummary struct {
	UUID       string `json:"uuid"`
	Status     string `json:"status"`
	FilePath   string `json:"file_path"`
	LineNumber int    `json:"line_number"`
	Context    string `json:"context,omitempty"`
}

// MessageResponse represents a message in JSON format for MCP responses
type MessageResponse struct {
	UUID      string `json:"uuid"`
	Author    string `json:"author"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
	IsUnread  bool   `json:"is_unread,omitempty"`
}

// ConversationResponse represents a full conversation in JSON format for MCP responses
type ConversationResponse struct {
	UUID        string            `json:"uuid"`
	Status      string            `json:"status"`
	FilePath    string            `json:"file_path"`
	LineNumber  int               `json:"line_number"`
	CodeVersion string            `json:"code_version"`
	Context     string            `json:"context,omitempty"`
	Messages    []MessageResponse `json:"messages"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
}

// ReplyResponse represents the result of creating a reply
type ReplyResponse struct {
	Success   bool   `json:"success"`
	UUID      string `json:"uuid"`
	Author    string `json:"author"`
	CreatedAt string `json:"created_at"`
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

	conversations, err := s.messaging.GetConversations(status, nil)
	if err != nil {
		s.logToStderr("Failed to get conversations: %v", err)
		return s.sendToolError(req.ID, fmt.Sprintf("Error getting conversations: %v", err))
	}

	s.logToStderr("Found %d conversations", len(conversations))

	// Build summaries with context
	summaries := make([]ConversationSummary, len(conversations))
	for i, conv := range conversations {
		summaries[i] = ConversationSummary{
			UUID:       conv.UUID,
			Status:     string(conv.Status),
			FilePath:   conv.FilePath,
			LineNumber: conv.LineNumber,
			Context:    conv.Context,
		}
	}

	// Format as JSON array
	result, err := json.Marshal(summaries)
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

	// Mark the conversation as read by AI
	if err := s.messaging.MarkConversationAs(uuid, critic.ConversationReadByAI); err != nil {
		s.logToStderr("Failed to mark conversation as read by AI: %v", err)
		// Don't fail the request, just log the error
	}

	// Convert to JSON response format
	messages := make([]MessageResponse, len(conversation.Messages))
	for i, msg := range conversation.Messages {
		messages[i] = MessageResponse{
			UUID:      msg.UUID,
			Author:    string(msg.Author),
			Message:   msg.Message,
			CreatedAt: msg.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			IsUnread:  msg.IsUnread,
		}
	}

	response := ConversationResponse{
		UUID:        conversation.UUID,
		Status:      string(conversation.Status),
		FilePath:    conversation.FilePath,
		LineNumber:  conversation.LineNumber,
		CodeVersion: conversation.CodeVersion,
		Context:     conversation.Context,
		Messages:    messages,
		CreatedAt:   conversation.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   conversation.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	result, err := json.Marshal(response)
	if err != nil {
		return s.sendToolError(req.ID, fmt.Sprintf("Error encoding result: %v", err))
	}

	s.logToStderr("Returning conversation with %d messages", len(conversation.Messages))
	return s.sendToolResult(req.ID, string(result))
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

	response := ReplyResponse{
		Success:   true,
		UUID:      reply.UUID,
		Author:    string(reply.Author),
		CreatedAt: reply.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	result, err := json.Marshal(response)
	if err != nil {
		return s.sendToolError(req.ID, fmt.Sprintf("Error encoding result: %v", err))
	}

	s.logToStderr("Created reply: %s", reply.UUID)
	return s.sendToolResult(req.ID, string(result))
}

// handleCriticAnnounce handles the critic_announce tool
func (s *Server) handleCriticAnnounce(req Request, params CallToolParams) error {
	if s.messaging == nil {
		s.logToStderr("Messaging not initialized")
		return s.sendToolError(req.ID, "Messaging not initialized")
	}

	message, ok := params.Arguments["message"].(string)
	if !ok || message == "" {
		return s.sendToolError(req.ID, "message is required")
	}

	s.logToStderr("Creating announcement: %s", message)

	// Get or create the root conversation
	rootConv, err := s.messaging.LoadRootConversation()
	if err != nil {
		s.logToStderr("Failed to load root conversation: %v", err)
		return s.sendToolError(req.ID, fmt.Sprintf("Error loading root conversation: %v", err))
	}

	// Add the announcement message as a reply
	reply, err := s.messaging.ReplyToConversation(rootConv.UUID, message, critic.AuthorAI)
	if err != nil {
		s.logToStderr("Failed to create announcement: %v", err)
		return s.sendToolError(req.ID, fmt.Sprintf("Error creating announcement: %v", err))
	}

	// Ensure the root conversation is marked as unresolved
	if err := s.messaging.MarkConversationAs(rootConv.UUID, critic.ConversationUnresolved); err != nil {
		s.logToStderr("Failed to mark root conversation as unresolved: %v", err)
		// Don't fail the request, the message was already created
	}

	response := ReplyResponse{
		Success:   true,
		UUID:      reply.UUID,
		Author:    string(reply.Author),
		CreatedAt: reply.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	result, err := json.Marshal(response)
	if err != nil {
		return s.sendToolError(req.ID, fmt.Sprintf("Error encoding result: %v", err))
	}

	s.logToStderr("Created announcement: %s", reply.UUID)
	return s.sendToolResult(req.ID, string(result))
}

// handleCriticExplain handles the critic_explain tool
func (s *Server) handleCriticExplain(req Request, params CallToolParams) error {
	if s.messaging == nil {
		s.logToStderr("Messaging not initialized")
		return s.sendToolError(req.ID, "Messaging not initialized")
	}

	file, ok := params.Arguments["file"].(string)
	if !ok || file == "" {
		return s.sendToolError(req.ID, "file is required")
	}

	lineFloat, ok := params.Arguments["line"].(float64)
	if !ok || lineFloat <= 0 {
		return s.sendToolError(req.ID, "line is required and must be a positive number")
	}
	line := int(lineFloat)

	comment, ok := params.Arguments["comment"].(string)
	if !ok || comment == "" {
		return s.sendToolError(req.ID, "comment is required")
	}

	s.logToStderr("Creating explanation at %s:%d", file, line)

	codeVersion := git.ResolveRef("HEAD")

	conversation, err := s.messaging.CreateConversation(critic.AuthorAI, comment, file, line, codeVersion, "", critic.TypeExplanation)
	if err != nil {
		s.logToStderr("Failed to create explanation: %v", err)
		return s.sendToolError(req.ID, fmt.Sprintf("Error creating explanation: %v", err))
	}

	response := ReplyResponse{
		Success:   true,
		UUID:      conversation.UUID,
		Author:    string(critic.AuthorAI),
		CreatedAt: conversation.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	result, err := json.Marshal(response)
	if err != nil {
		return s.sendToolError(req.ID, fmt.Sprintf("Error encoding result: %v", err))
	}

	s.logToStderr("Created explanation: %s", conversation.UUID)
	return s.sendToolResult(req.ID, string(result))
}

// handlePromptsList returns the list of available prompts
func (s *Server) handlePromptsList(req Request) error {
	prompts := []Prompt{
		{
			Name:        "summarize",
			Description: "Summarize all uncommitted changes and post via the critic_announce tool",
		},
		{
			Name:        "step",
			Description: "Get unresolved critic conversations, address critical feedback, reply or make adjustments as necessary",
		},
		{
			Name:        "loop",
			Description: "Repeat /critic:step until all critic conversations are resolved",
		},
		{
			Name:        "explain",
			Description: "Identify the changes and post an explanation on all non-obvious changes",
		},
	}

	return s.sendResult(req.ID, PromptsListResult{Prompts: prompts})
}

// handlePromptsGet returns the prompt messages for a given prompt
func (s *Server) handlePromptsGet(req Request) error {
	paramsJSON, err := json.Marshal(req.Params)
	if err != nil {
		return s.sendError(req.ID, InvalidParams, "Invalid params", nil)
	}

	var params GetPromptParams
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		return s.sendError(req.ID, InvalidParams, "Invalid params", nil)
	}

	s.logToStderr("Prompt get: %s", params.Name)

	switch params.Name {
	case "summarize":
		return s.sendResult(req.ID, GetPromptResult{
			Description: "Summarize all uncommitted changes and post via critic_announce",
			Messages: []PromptMessage{
				{
					Role:    "user",
					Content: TextContent("Summarize all uncommitted changes, and post these via the critic_announce MCP tool."),
				},
			},
		})
	case "step":
		return s.sendResult(req.ID, GetPromptResult{
			Description: "Address unresolved critic feedback",
			Messages: []PromptMessage{
				{
					Role:    "user",
					Content: TextContent("Get unresolved critic conversations via the get_critic_conversations MCP tool. Address critical feedback, reply or make adjustments as necessary. After each tool call, re-check for new unresolved messages and address any new feedback before continuing."),
				},
			},
		})
	case "loop":
		return s.sendResult(req.ID, GetPromptResult{
			Description: "Resolve all critic conversations iteratively",
			Messages: []PromptMessage{
				{
					Role:    "user",
					Content: TextContent("Resolve all unresolved critic conversations. After each tool call, check for new unresolved messages using get_critic_conversations. Address any new feedback before continuing. Repeat until all conversations are resolved."),
				},
			},
		})
	case "explain":
		return s.sendResult(req.ID, GetPromptResult{
			Description: "Explain non-obvious code changes",
			Messages: []PromptMessage{
				{
					Role:    "user",
					Content: TextContent("Review all uncommitted changes in the diff. For each non-obvious change — anything where the intent, reason, or mechanism isn't immediately clear from the code alone — post an explanation using the critic_explain MCP tool. Skip trivial or self-explanatory changes (renames, formatting, obvious bug fixes). Focus on: why a change was made, subtle implications, non-obvious design decisions, and tricky logic."),
				},
			},
		})
	default:
		return s.sendError(req.ID, InvalidParams, fmt.Sprintf("Unknown prompt: %s", params.Name), nil)
	}
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

package webui

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/radiospiel/critic/simple-go/logger"
	"github.com/radiospiel/critic/src/git"
	"github.com/radiospiel/critic/src/highlight"
	"github.com/radiospiel/critic/src/pkg/critic"
	"github.com/radiospiel/critic/src/pkg/types"
)

// FileListItem represents a file in the list
type FileListItem struct {
	Path        string `json:"path"`
	Status      string `json:"status"`
	HasComments bool   `json:"hasComments"`
	Unresolved  int    `json:"unresolved"`
}

// FileListResponse is the JSON response for the file list API
type FileListResponse struct {
	Files []FileListItem `json:"files"`
}

// handleFileList returns the file list as JSON
func (s *Server) handleFileList(w http.ResponseWriter, r *http.Request) {
	diff := s.getDiff()
	if diff == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(FileListResponse{Files: []FileListItem{}})
		return
	}

	items := make([]FileListItem, 0, len(diff.Files))
	for _, f := range diff.Files {
		path := f.NewPath
		if f.IsDeleted {
			path = f.OldPath
		}

		status := "M"
		if f.IsNew {
			status = "A"
		} else if f.IsDeleted {
			status = "D"
		} else if f.IsRenamed {
			status = "R"
		}

		// Get conversation summary for this file
		summary, _ := s.messaging.GetFileConversationSummary(path)
		hasComments := summary != nil && (summary.HasUnresolvedComments || summary.HasResolvedComments)
		hasUnresolved := summary != nil && summary.HasUnresolvedComments

		items = append(items, FileListItem{
			Path:        path,
			Status:      status,
			HasComments: hasComments,
			Unresolved:  boolToInt(hasUnresolved),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(FileListResponse{Files: items})
}

// DiffLine represents a line in the diff view
type DiffLine struct {
	Type        string `json:"type"` // "context", "added", "deleted", "header"
	Content     string `json:"content"`
	HTMLContent string `json:"htmlContent"` // Syntax highlighted HTML
	OldNum      int    `json:"oldNum"`
	NewNum      int    `json:"newNum"`
	LineIndex   int    `json:"lineIndex"` // Index in the hunk for commenting
}

// DiffHunk represents a hunk in the diff
type DiffHunk struct {
	Header string     `json:"header"`
	Lines  []DiffLine `json:"lines"`
}

// DiffData holds diff data for a file
type DiffData struct {
	FilePath     string                 `json:"filePath"`
	Hunks        []DiffHunk             `json:"hunks"`
	Conversations []*critic.Conversation `json:"conversations"`
	IsNew        bool                   `json:"isNew"`
	IsDeleted    bool                   `json:"isDeleted"`
	IsRenamed    bool                   `json:"isRenamed"`
	OldPath      string                 `json:"oldPath"`
}

// handleDiff returns the diff for a file as JSON
func (s *Server) handleDiff(w http.ResponseWriter, r *http.Request) {
	filePath := r.PathValue("path")
	if filePath == "" {
		http.Error(w, "File path required", http.StatusBadRequest)
		return
	}

	diff := s.getDiff()
	if diff == nil {
		http.Error(w, "No diff available", http.StatusNotFound)
		return
	}

	var file *types.FileDiff
	for _, f := range diff.Files {
		path := f.NewPath
		if f.IsDeleted {
			path = f.OldPath
		}
		if path == filePath {
			file = f
			break
		}
	}

	if file == nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Build diff data with syntax highlighting
	highlighter := highlight.NewHTMLHighlighter()
	hunks := make([]DiffHunk, 0, len(file.Hunks))
	for _, h := range file.Hunks {
		lines := make([]DiffLine, 0, len(h.Lines))
		for i, l := range h.Lines {
			lineType := "context"
			switch l.Type {
			case types.LineAdded:
				lineType = "added"
			case types.LineDeleted:
				lineType = "deleted"
			}

			// Get syntax-highlighted HTML for the content
			htmlContent := highlighter.HighlightLineHTML(l.Content, filePath)

			lines = append(lines, DiffLine{
				Type:        lineType,
				Content:     l.Content,
				HTMLContent: htmlContent,
				OldNum:      l.OldNum,
				NewNum:      l.NewNum,
				LineIndex:   i,
			})
		}

		hunks = append(hunks, DiffHunk{
			Header: h.Header,
			Lines:  lines,
		})
	}

	// Get conversations for this file
	conversations, err := s.messaging.GetConversations("")
	if err != nil {
		logger.Warn("Failed to get conversations: %v", err)
	}
	fileConvs := make([]*critic.Conversation, 0)
	for _, c := range conversations {
		if c.FilePath == filePath {
			fullConv, err := s.messaging.GetFullConversation(c.UUID)
			if err == nil {
				fileConvs = append(fileConvs, fullConv)
			}
		}
	}

	data := DiffData{
		FilePath:      filePath,
		Hunks:         hunks,
		Conversations: fileConvs,
		IsNew:         file.IsNew,
		IsDeleted:     file.IsDeleted,
		IsRenamed:     file.IsRenamed,
		OldPath:       file.OldPath,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// ConversationsResponse is the JSON response for conversations
type ConversationsResponse struct {
	Conversations []*critic.Conversation `json:"conversations"`
}

// handleConversations returns conversations for a file as JSON
func (s *Server) handleConversations(w http.ResponseWriter, r *http.Request) {
	filePath := r.PathValue("path")
	if filePath == "" {
		http.Error(w, "File path required", http.StatusBadRequest)
		return
	}

	conversations, err := s.messaging.GetConversations("")
	if err != nil {
		http.Error(w, "Failed to get conversations", http.StatusInternalServerError)
		return
	}

	fileConvs := make([]*critic.Conversation, 0)
	for _, c := range conversations {
		if c.FilePath == filePath {
			fullConv, err := s.messaging.GetFullConversation(c.UUID)
			if err == nil {
				fileConvs = append(fileConvs, fullConv)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ConversationsResponse{Conversations: fileConvs})
}

// CommentRequest represents a new comment request
type CommentRequest struct {
	FilePath   string `json:"filePath"`
	LineNumber int    `json:"lineNumber"`
	Message    string `json:"message"`
}

// handleCreateComment creates a new conversation
func (s *Server) handleCreateComment(w http.ResponseWriter, r *http.Request) {
	var req CommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	req.Message = strings.TrimSpace(req.Message)
	if req.FilePath == "" || req.LineNumber == 0 || req.Message == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Get current code version
	codeVersion, _ := git.ResolveRef("HEAD")
	context := git.GetLineContext(req.FilePath, req.LineNumber, codeVersion)

	conv, err := s.messaging.CreateConversation(
		critic.AuthorHuman,
		req.Message,
		req.FilePath,
		req.LineNumber,
		codeVersion,
		context,
	)
	if err != nil {
		logger.Error("Failed to create conversation: %v", err)
		http.Error(w, "Failed to create comment", http.StatusInternalServerError)
		return
	}

	// Broadcast update to all clients
	s.broadcastUpdate("conversation", conv.UUID)

	// Return the new conversation as JSON
	fullConv, _ := s.messaging.GetFullConversation(conv.UUID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fullConv)
}

// ReplyRequest represents a reply request
type ReplyRequest struct {
	ConversationID string `json:"conversationId"`
	Message        string `json:"message"`
}

// handleReply adds a reply to a conversation
func (s *Server) handleReply(w http.ResponseWriter, r *http.Request) {
	var req ReplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	req.Message = strings.TrimSpace(req.Message)
	if req.ConversationID == "" || req.Message == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	_, err := s.messaging.ReplyToConversation(req.ConversationID, req.Message, critic.AuthorHuman)
	if err != nil {
		logger.Error("Failed to create reply: %v", err)
		http.Error(w, "Failed to create reply", http.StatusInternalServerError)
		return
	}

	// Broadcast update
	s.broadcastUpdate("conversation", req.ConversationID)

	// Return the updated conversation as JSON
	fullConv, _ := s.messaging.GetFullConversation(req.ConversationID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fullConv)
}

// handleResolve marks a conversation as resolved
func (s *Server) handleResolve(w http.ResponseWriter, r *http.Request) {
	uuid := r.PathValue("uuid")
	if uuid == "" {
		http.Error(w, "UUID required", http.StatusBadRequest)
		return
	}

	if err := s.messaging.MarkAsResolved(uuid); err != nil {
		logger.Error("Failed to resolve conversation: %v", err)
		http.Error(w, "Failed to resolve", http.StatusInternalServerError)
		return
	}

	// Broadcast update
	s.broadcastUpdate("conversation", uuid)

	// Return updated conversation as JSON
	fullConv, _ := s.messaging.GetFullConversation(uuid)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fullConv)
}

// handleUnresolve marks a conversation as unresolved
func (s *Server) handleUnresolve(w http.ResponseWriter, r *http.Request) {
	uuid := r.PathValue("uuid")
	if uuid == "" {
		http.Error(w, "UUID required", http.StatusBadRequest)
		return
	}

	if err := s.messaging.MarkAsUnresolved(uuid); err != nil {
		logger.Error("Failed to unresolve conversation: %v", err)
		http.Error(w, "Failed to unresolve", http.StatusInternalServerError)
		return
	}

	// Broadcast update
	s.broadcastUpdate("conversation", uuid)

	// Return updated conversation as JSON
	fullConv, _ := s.messaging.GetFullConversation(uuid)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fullConv)
}

// broadcastUpdate sends an update notification to all WebSocket clients
func (s *Server) broadcastUpdate(updateType, id string) {
	msg := map[string]string{
		"type": updateType,
		"id":   id,
	}
	data, _ := json.Marshal(msg)
	s.Broadcast(data)
}

// boolToInt converts a bool to an int (0 or 1)
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

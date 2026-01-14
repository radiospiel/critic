package webui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"git.15b.it/eno/critic/internal/git"
	"git.15b.it/eno/critic/pkg/critic"
	"git.15b.it/eno/critic/pkg/types"
	"git.15b.it/eno/critic/simple-go/logger"
)

// PageData holds data for page templates
type PageData struct {
	Title       string
	Files       []*types.FileDiff
	CurrentFile *types.FileDiff
	FilePath    string
	Theme       string
}

// handleIndex renders the main page
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	diff := s.getDiff()
	var files []*types.FileDiff
	if diff != nil {
		files = diff.Files
	}

	data := PageData{
		Title: "Critic - Code Review",
		Files: files,
		Theme: "", // Theme is managed client-side via localStorage
	}

	if err := s.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		logger.Error("Failed to render index: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleFile renders a specific file's diff
func (s *Server) handleFile(w http.ResponseWriter, r *http.Request) {
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

	var currentFile *types.FileDiff
	for _, f := range diff.Files {
		path := f.NewPath
		if f.IsDeleted {
			path = f.OldPath
		}
		if path == filePath {
			currentFile = f
			break
		}
	}

	if currentFile == nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	data := PageData{
		Title:       fmt.Sprintf("Critic - %s", filePath),
		Files:       diff.Files,
		CurrentFile: currentFile,
		FilePath:    filePath,
		Theme:       "", // Theme is managed client-side via localStorage
	}

	if err := s.templates.ExecuteTemplate(w, "file.html", data); err != nil {
		logger.Error("Failed to render file: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// FileListItem represents a file in the list
type FileListItem struct {
	Path        string `json:"path"`
	Status      string `json:"status"`
	HasComments bool   `json:"hasComments"`
	Unresolved  int    `json:"unresolved"`
}

// handleFileList returns the file list as HTML (for htmx)
func (s *Server) handleFileList(w http.ResponseWriter, r *http.Request) {
	diff := s.getDiff()
	if diff == nil {
		http.Error(w, "No diff available", http.StatusNotFound)
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

	if err := s.templates.ExecuteTemplate(w, "filelist.html", items); err != nil {
		logger.Error("Failed to render file list: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// DiffLine represents a line in the diff view
type DiffLine struct {
	Type      string `json:"type"` // "context", "added", "deleted", "header"
	Content   string `json:"content"`
	OldNum    int    `json:"oldNum"`
	NewNum    int    `json:"newNum"`
	LineIndex int    `json:"lineIndex"` // Index in the hunk for commenting
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

// handleDiff returns the diff for a file as HTML (for htmx)
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

	// Build diff data
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

			lines = append(lines, DiffLine{
				Type:      lineType,
				Content:   l.Content,
				OldNum:    l.OldNum,
				NewNum:    l.NewNum,
				LineIndex: i,
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

	if err := s.templates.ExecuteTemplate(w, "diff.html", data); err != nil {
		logger.Error("Failed to render diff: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleConversations returns conversations for a file
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

	if err := s.templates.ExecuteTemplate(w, "conversations.html", fileConvs); err != nil {
		logger.Error("Failed to render conversations: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// CommentRequest represents a new comment request
type CommentRequest struct {
	FilePath   string `json:"filePath"`
	LineNumber int    `json:"lineNumber"`
	Message    string `json:"message"`
}

// handleCreateComment creates a new conversation
func (s *Server) handleCreateComment(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	filePath := r.FormValue("filePath")
	lineNumber := 0
	fmt.Sscanf(r.FormValue("lineNumber"), "%d", &lineNumber)
	message := strings.TrimSpace(r.FormValue("message"))

	if filePath == "" || lineNumber == 0 || message == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Get current code version
	codeVersion, _ := git.ResolveRef("HEAD")
	context := git.GetLineContext(filePath, lineNumber, codeVersion)

	conv, err := s.messaging.CreateConversation(
		critic.AuthorHuman,
		message,
		filePath,
		lineNumber,
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

	// Return the new conversation HTML
	fullConv, _ := s.messaging.GetFullConversation(conv.UUID)
	if err := s.templates.ExecuteTemplate(w, "conversation.html", fullConv); err != nil {
		logger.Error("Failed to render conversation: %v", err)
	}
}

// ReplyRequest represents a reply request
type ReplyRequest struct {
	ConversationID string `json:"conversationId"`
	Message        string `json:"message"`
}

// handleReply adds a reply to a conversation
func (s *Server) handleReply(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	conversationID := r.FormValue("conversationId")
	message := strings.TrimSpace(r.FormValue("message"))

	if conversationID == "" || message == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	_, err := s.messaging.ReplyToConversation(conversationID, message, critic.AuthorHuman)
	if err != nil {
		logger.Error("Failed to create reply: %v", err)
		http.Error(w, "Failed to create reply", http.StatusInternalServerError)
		return
	}

	// Broadcast update
	s.broadcastUpdate("conversation", conversationID)

	// Return the updated conversation HTML
	fullConv, _ := s.messaging.GetFullConversation(conversationID)
	if err := s.templates.ExecuteTemplate(w, "conversation.html", fullConv); err != nil {
		logger.Error("Failed to render conversation: %v", err)
	}
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

	// Return updated conversation
	fullConv, _ := s.messaging.GetFullConversation(uuid)
	if err := s.templates.ExecuteTemplate(w, "conversation.html", fullConv); err != nil {
		logger.Error("Failed to render conversation: %v", err)
	}
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

	// Return updated conversation
	fullConv, _ := s.messaging.GetFullConversation(uuid)
	if err := s.templates.ExecuteTemplate(w, "conversation.html", fullConv); err != nil {
		logger.Error("Failed to render conversation: %v", err)
	}
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

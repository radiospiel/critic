package server

import (
	"encoding/json"
	"net/http"

	"github.com/radiospiel/critic/simple-go/logger"
)

// CommentMessage represents a single message in a conversation (for JSON response).
type CommentMessage struct {
	ID        string `json:"id"`
	Author    string `json:"author"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	IsUnread  bool   `json:"isUnread"`
}

// CommentConversation represents a comment thread at a specific line (for JSON response).
type CommentConversation struct {
	ID          string           `json:"id"`
	Status      string           `json:"status"`
	FilePath    string           `json:"filePath"`
	LineNumber  int              `json:"lineNumber"`
	CodeVersion string           `json:"codeVersion"`
	Context     string           `json:"context"`
	Messages    []CommentMessage `json:"messages"`
	CreatedAt   string           `json:"createdAt"`
	UpdatedAt   string           `json:"updatedAt"`
}

// GetCommentsResponse is the JSON response for the GetComments endpoint.
type GetCommentsResponse struct {
	Conversations []CommentConversation `json:"conversations"`
	Error         string                `json:"error,omitempty"`
}

// GetCommentsHandler returns an HTTP handler for getting comments for a file.
// This is a temporary HTTP/JSON endpoint until protobuf code can be regenerated.
func (s *Server) GetCommentsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get file path from query parameter
		filePath := r.URL.Query().Get("path")
		if filePath == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(GetCommentsResponse{
				Error: "path query parameter is required",
			})
			return
		}

		logger.Info("GetComments: path=%s", filePath)

		// Get conversations for the file
		conversations, err := s.config.Messaging.GetConversationsForFile(filePath)
		if err != nil {
			logger.Error("Failed to get conversations for file %s: %v", filePath, err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(GetCommentsResponse{
				Error: "failed to get comments: " + err.Error(),
			})
			return
		}

		// Convert to response format
		response := GetCommentsResponse{
			Conversations: make([]CommentConversation, 0, len(conversations)),
		}

		for _, conv := range conversations {
			messages := make([]CommentMessage, 0, len(conv.Messages))
			for _, msg := range conv.Messages {
				messages = append(messages, CommentMessage{
					ID:        msg.UUID,
					Author:    string(msg.Author),
					Content:   msg.Message,
					CreatedAt: msg.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
					UpdatedAt: msg.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
					IsUnread:  msg.IsUnread,
				})
			}

			response.Conversations = append(response.Conversations, CommentConversation{
				ID:          conv.UUID,
				Status:      string(conv.Status),
				FilePath:    conv.FilePath,
				LineNumber:  conv.LineNumber,
				CodeVersion: conv.CodeVersion,
				Context:     conv.Context,
				Messages:    messages,
				CreatedAt:   conv.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				UpdatedAt:   conv.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			})
		}

		logger.Info("GetComments: returning %d conversations for %s", len(response.Conversations), filePath)
		json.NewEncoder(w).Encode(response)
	}
}

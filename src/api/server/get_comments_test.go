package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/radiospiel/critic/src/pkg/critic"
)

func TestGetCommentsHandler(t *testing.T) {
	// Create a server with a dummy messaging implementation that has some test data
	messaging := critic.NewDummyMessaging()

	// Add test conversation
	testConv := &critic.Conversation{
		UUID:        "test-conv-1",
		Status:      critic.StatusUnresolved,
		FilePath:    "test.go",
		LineNumber:  10,
		CodeVersion: "abc123",
		Context:     "",
		Messages: []critic.Message{
			{
				UUID:      "test-msg-1",
				Author:    critic.AuthorHuman,
				Message:   "Test comment",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				IsUnread:  false,
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	messaging.Conversations["test.go"] = []*critic.Conversation{testConv}

	s := &Server{
		config: Config{
			Messaging: messaging,
		},
	}

	// Test with valid path
	t.Run("valid path returns comments", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/comments?path=test.go", nil)
		w := httptest.NewRecorder()

		handler := s.GetCommentsHandler()
		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status OK, got %d", w.Code)
		}

		var response GetCommentsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(response.Conversations) != 1 {
			t.Errorf("Expected 1 conversation, got %d", len(response.Conversations))
		}

		if response.Conversations[0].ID != "test-conv-1" {
			t.Errorf("Expected conversation ID test-conv-1, got %s", response.Conversations[0].ID)
		}
	})

	// Test with missing path
	t.Run("missing path returns error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/comments", nil)
		w := httptest.NewRecorder()

		handler := s.GetCommentsHandler()
		handler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status BadRequest, got %d", w.Code)
		}

		var response GetCommentsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response.Error == "" {
			t.Error("Expected error message, got empty")
		}
	})

	// Test with non-existent file
	t.Run("non-existent file returns empty list", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/comments?path=nonexistent.go", nil)
		w := httptest.NewRecorder()

		handler := s.GetCommentsHandler()
		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status OK, got %d", w.Code)
		}

		var response GetCommentsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(response.Conversations) != 0 {
			t.Errorf("Expected 0 conversations, got %d", len(response.Conversations))
		}
	})
}

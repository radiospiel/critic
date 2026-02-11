package server

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/simple-go/assert"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/pkg/critic"
)

func TestGetConversations_ReturnsConversationsForFile(t *testing.T) {
	now := time.Now()
	messaging := critic.NewDummyMessaging()
	messaging.Conversations["src/main.go"] = []*critic.Conversation{
		{
			UUID:        "conv-1",
			Status:      critic.StatusUnresolved,
			FilePath:    "src/main.go",
			LineNumber:  42,
			CodeVersion: "abc123",
			Context:     "func main() {",
			Messages: []critic.Message{
				{
					UUID:      "msg-1",
					Author:    critic.AuthorHuman,
					Message:   "This needs refactoring",
					CreatedAt: now,
					UpdatedAt: now,
					IsUnread:  false,
				},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	s := &Server{
		config: Config{
			Messaging: messaging,
		},
	}

	req := connect.NewRequest(&api.GetConversationsRequest{Paths: []string{"src/main.go"}})
	resp, err := s.GetConversations(context.Background(), req)

	assert.NoError(t, err, "GetConversations should not return error")
	assert.Equals(t, len(resp.Msg.GetConversations()), 1, "should return one conversation")

	conv := resp.Msg.GetConversations()[0]
	assert.Equals(t, conv.GetId(), "conv-1", "conversation ID should match")
	assert.Equals(t, conv.GetStatus(), api.ConversationStatus_CONVERSATION_STATUS_UNRESOLVED, "status should be unresolved")
	assert.Equals(t, conv.GetFilePath(), "src/main.go", "file path should match")
	assert.Equals(t, conv.GetLineNumber(), int32(42), "line number should match")
	assert.Equals(t, conv.GetCodeVersion(), "abc123", "code version should match")
	assert.Equals(t, conv.GetContext(), "func main() {", "context should match")
	assert.Equals(t, len(conv.GetMessages()), 1, "should have one message")

	msg := conv.GetMessages()[0]
	assert.Equals(t, msg.GetId(), "msg-1", "message ID should match")
	assert.Equals(t, msg.GetAuthor(), "human", "author should be human")
	assert.Equals(t, msg.GetContent(), "This needs refactoring", "message content should match")
	assert.Equals(t, msg.GetIsUnread(), false, "message should not be unread")
}

func TestGetConversations_ReturnsEmptyForNoConversations(t *testing.T) {
	messaging := critic.NewDummyMessaging()

	s := &Server{
		config: Config{
			Messaging: messaging,
		},
	}

	req := connect.NewRequest(&api.GetConversationsRequest{Paths: []string{"nonexistent.go"}})
	resp, err := s.GetConversations(context.Background(), req)

	assert.NoError(t, err, "GetConversations should not return error")
	assert.Equals(t, len(resp.Msg.GetConversations()), 0, "should return empty conversations")
}

func TestGetConversations_ReturnsMultipleConversations(t *testing.T) {
	now := time.Now()
	messaging := critic.NewDummyMessaging()
	messaging.Conversations["src/utils.go"] = []*critic.Conversation{
		{
			UUID:       "conv-1",
			Status:     critic.StatusUnresolved,
			FilePath:   "src/utils.go",
			LineNumber: 10,
			Messages:   []critic.Message{{UUID: "msg-1", Author: critic.AuthorHuman, Message: "First comment", CreatedAt: now, UpdatedAt: now}},
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		{
			UUID:       "conv-2",
			Status:     critic.StatusResolved,
			FilePath:   "src/utils.go",
			LineNumber: 25,
			Messages:   []critic.Message{{UUID: "msg-2", Author: critic.AuthorAI, Message: "AI response", CreatedAt: now, UpdatedAt: now, IsUnread: true}},
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}

	s := &Server{
		config: Config{
			Messaging: messaging,
		},
	}

	req := connect.NewRequest(&api.GetConversationsRequest{Paths: []string{"src/utils.go"}})
	resp, err := s.GetConversations(context.Background(), req)

	assert.NoError(t, err, "GetConversations should not return error")
	assert.Equals(t, len(resp.Msg.GetConversations()), 2, "should return two conversations")

	conv1 := resp.Msg.GetConversations()[0]
	assert.Equals(t, conv1.GetId(), "conv-1", "first conversation ID should match")
	assert.Equals(t, conv1.GetStatus(), api.ConversationStatus_CONVERSATION_STATUS_UNRESOLVED, "first status should be unresolved")

	conv2 := resp.Msg.GetConversations()[1]
	assert.Equals(t, conv2.GetId(), "conv-2", "second conversation ID should match")
	assert.Equals(t, conv2.GetStatus(), api.ConversationStatus_CONVERSATION_STATUS_RESOLVED, "second status should be resolved")
	assert.True(t, conv2.GetMessages()[0].GetIsUnread(), "AI message should be unread")
}

func TestGetConversations_HandlesMultipleMessages(t *testing.T) {
	now := time.Now()
	later := now.Add(time.Hour)

	messaging := critic.NewDummyMessaging()
	messaging.Conversations["src/handler.go"] = []*critic.Conversation{
		{
			UUID:       "conv-1",
			Status:     critic.StatusUnresolved,
			FilePath:   "src/handler.go",
			LineNumber: 100,
			Messages: []critic.Message{
				{UUID: "msg-1", Author: critic.AuthorHuman, Message: "Why is this slow?", CreatedAt: now, UpdatedAt: now},
				{UUID: "msg-2", Author: critic.AuthorAI, Message: "Consider using caching", CreatedAt: later, UpdatedAt: later, IsUnread: true},
				{UUID: "msg-3", Author: critic.AuthorHuman, Message: "Good idea!", CreatedAt: later.Add(time.Minute), UpdatedAt: later.Add(time.Minute)},
			},
			CreatedAt: now,
			UpdatedAt: later.Add(time.Minute),
		},
	}

	s := &Server{
		config: Config{
			Messaging: messaging,
		},
	}

	req := connect.NewRequest(&api.GetConversationsRequest{Paths: []string{"src/handler.go"}})
	resp, err := s.GetConversations(context.Background(), req)

	assert.NoError(t, err, "GetConversations should not return error")
	assert.Equals(t, len(resp.Msg.GetConversations()), 1, "should return one conversation")

	conv := resp.Msg.GetConversations()[0]
	assert.Equals(t, len(conv.GetMessages()), 3, "should have three messages")

	assert.Equals(t, conv.GetMessages()[0].GetAuthor(), "human", "first message author")
	assert.Equals(t, conv.GetMessages()[1].GetAuthor(), "ai", "second message author")
	assert.Equals(t, conv.GetMessages()[2].GetAuthor(), "human", "third message author")
}

func TestGetConversations_FiltersByStatus(t *testing.T) {
	now := time.Now()
	messaging := critic.NewDummyMessaging()
	messaging.Conversations["src/app.go"] = []*critic.Conversation{
		{
			UUID:       "conv-1",
			Status:     critic.StatusUnresolved,
			FilePath:   "src/app.go",
			LineNumber: 10,
			Messages:   []critic.Message{{UUID: "msg-1", Author: critic.AuthorHuman, Message: "Fix this", CreatedAt: now, UpdatedAt: now}},
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		{
			UUID:       "conv-2",
			Status:     critic.StatusResolved,
			FilePath:   "src/app.go",
			LineNumber: 20,
			Messages:   []critic.Message{{UUID: "msg-2", Author: critic.AuthorHuman, Message: "Done", CreatedAt: now, UpdatedAt: now}},
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		{
			UUID:       "conv-3",
			Status:     critic.StatusArchived,
			FilePath:   "src/app.go",
			LineNumber: 30,
			Messages:   []critic.Message{{UUID: "msg-3", Author: critic.AuthorHuman, Message: "Old", CreatedAt: now, UpdatedAt: now}},
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}

	s := &Server{
		config: Config{
			Messaging: messaging,
		},
	}

	// Filter by unresolved only
	req := connect.NewRequest(&api.GetConversationsRequest{
		Paths:    []string{"src/app.go"},
		Statuses: []api.ConversationStatus{api.ConversationStatus_CONVERSATION_STATUS_UNRESOLVED},
	})
	resp, err := s.GetConversations(context.Background(), req)

	assert.NoError(t, err, "GetConversations should not return error")
	assert.Equals(t, len(resp.Msg.GetConversations()), 1, "should return one unresolved conversation")
	assert.Equals(t, resp.Msg.GetConversations()[0].GetId(), "conv-1", "should be the unresolved conversation")

	// Filter by multiple statuses
	req2 := connect.NewRequest(&api.GetConversationsRequest{
		Paths: []string{"src/app.go"},
		Statuses: []api.ConversationStatus{
			api.ConversationStatus_CONVERSATION_STATUS_RESOLVED,
			api.ConversationStatus_CONVERSATION_STATUS_ARCHIVED,
		},
	})
	resp2, err := s.GetConversations(context.Background(), req2)

	assert.NoError(t, err, "GetConversations should not return error")
	assert.Equals(t, len(resp2.Msg.GetConversations()), 2, "should return resolved and archived conversations")
}

func TestGetConversations_EmptyPathsReturnsAll(t *testing.T) {
	now := time.Now()
	messaging := critic.NewDummyMessaging()
	messaging.Conversations["src/a.go"] = []*critic.Conversation{
		{
			UUID:       "conv-1",
			Status:     critic.StatusUnresolved,
			FilePath:   "src/a.go",
			LineNumber: 1,
			Messages:   []critic.Message{{UUID: "msg-1", Author: critic.AuthorHuman, Message: "A", CreatedAt: now, UpdatedAt: now}},
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}
	messaging.Conversations["src/b.go"] = []*critic.Conversation{
		{
			UUID:       "conv-2",
			Status:     critic.StatusResolved,
			FilePath:   "src/b.go",
			LineNumber: 2,
			Messages:   []critic.Message{{UUID: "msg-2", Author: critic.AuthorHuman, Message: "B", CreatedAt: now, UpdatedAt: now}},
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}

	s := &Server{
		config: Config{
			Messaging: messaging,
		},
	}

	// Empty paths = return all
	req := connect.NewRequest(&api.GetConversationsRequest{})
	resp, err := s.GetConversations(context.Background(), req)

	assert.NoError(t, err, "GetConversations should not return error")
	assert.Equals(t, len(resp.Msg.GetConversations()), 2, "should return conversations from all files")
}

func TestCriticToApiMessage(t *testing.T) {
	now := time.Now()
	msg := critic.Message{
		UUID:      "test-uuid",
		Author:    critic.AuthorAI,
		Message:   "Test message content",
		CreatedAt: now,
		UpdatedAt: now.Add(time.Hour),
		IsUnread:  true,
	}

	apiMsg := criticToApiMessage(msg, 0)

	assert.Equals(t, apiMsg.GetId(), "test-uuid", "ID should match")
	assert.Equals(t, apiMsg.GetAuthor(), "ai", "author should match")
	assert.Equals(t, apiMsg.GetContent(), "Test message content", "content should match")
	assert.True(t, apiMsg.GetIsUnread(), "IsUnread should match")
	assert.Equals(t, apiMsg.GetCreatedAt(), now.Format("2006-01-02T15:04:05Z07:00"), "CreatedAt format should match")
	assert.Equals(t, apiMsg.GetUpdatedAt(), now.Add(time.Hour).Format("2006-01-02T15:04:05Z07:00"), "UpdatedAt format should match")
}

func TestCriticToApiConversation(t *testing.T) {
	now := time.Now()
	conv := &critic.Conversation{
		UUID:        "conv-uuid",
		Status:      critic.StatusResolved,
		FilePath:    "path/to/file.go",
		LineNumber:  55,
		CodeVersion: "sha256abc",
		Context:     "// comment context",
		Messages: []critic.Message{
			{UUID: "msg-1", Author: critic.AuthorHuman, Message: "Hello", CreatedAt: now, UpdatedAt: now},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	apiConv := criticToApiConversation(conv, 0)

	assert.Equals(t, apiConv.GetId(), "conv-uuid", "ID should match")
	assert.Equals(t, apiConv.GetStatus(), api.ConversationStatus_CONVERSATION_STATUS_RESOLVED, "status should match")
	assert.Equals(t, apiConv.GetFilePath(), "path/to/file.go", "file path should match")
	assert.Equals(t, apiConv.GetLineNumber(), int32(55), "line number should match")
	assert.Equals(t, apiConv.GetCodeVersion(), "sha256abc", "code version should match")
	assert.Equals(t, apiConv.GetContext(), "// comment context", "context should match")
	assert.Equals(t, len(apiConv.GetMessages()), 1, "should have one message")
}

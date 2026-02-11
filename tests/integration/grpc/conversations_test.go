package grpc_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/radiospiel/critic/simple-go/assert"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/api/server"
	"github.com/radiospiel/critic/src/pkg/critic"
)

// TestMessaging implements critic.Messaging with proper storage for integration testing.
// Unlike DummyMessaging, this properly stores and retrieves conversations through all operations.
type TestMessaging struct {
	conversations map[string][]*critic.Conversation // key: file path
}

func NewTestMessaging() *TestMessaging {
	return &TestMessaging{
		conversations: make(map[string][]*critic.Conversation),
	}
}

func (m *TestMessaging) GetConversations(status string, paths []string) ([]*critic.Conversation, error) {
	pathSet := make(map[string]bool, len(paths))
	for _, p := range paths {
		pathSet[p] = true
	}

	var all []*critic.Conversation
	for filePath, convs := range m.conversations {
		if len(paths) > 0 && !pathSet[filePath] {
			continue
		}
		for _, c := range convs {
			if status == "" || string(c.Status) == status {
				all = append(all, c)
			}
		}
	}
	return all, nil
}

func (m *TestMessaging) GetFullConversations(uuids []string) ([]*critic.Conversation, error) {
	uuidSet := make(map[string]bool, len(uuids))
	for _, u := range uuids {
		uuidSet[u] = true
	}

	var result []*critic.Conversation
	for _, convs := range m.conversations {
		for _, c := range convs {
			if uuidSet[c.UUID] {
				result = append(result, c)
			}
		}
	}
	return result, nil
}

func (m *TestMessaging) GetFullConversation(uuid string) (*critic.Conversation, error) {
	for _, convs := range m.conversations {
		for _, c := range convs {
			if c.UUID == uuid {
				return c, nil
			}
		}
	}
	return nil, nil
}

func (m *TestMessaging) GetConversationsForFile(filePath string) ([]*critic.Conversation, error) {
	return m.conversations[filePath], nil
}

func (m *TestMessaging) getFileConversationSummary(filePath string) (*critic.FileConversationSummary, error) {
	convs := m.conversations[filePath]
	if len(convs) == 0 {
		return nil, nil
	}

	summary := &critic.FileConversationSummary{
		FilePath:   filePath,
		TotalCount: len(convs),
	}

	for _, c := range convs {
		switch c.Status {
		case critic.StatusResolved:
			summary.ResolvedCount++
			summary.HasResolvedComments = true
		case critic.StatusUnresolved:
			summary.UnresolvedCount++
			summary.HasUnresolvedComments = true
		}
		// Check for unread AI messages
		for _, msg := range c.Messages {
			if msg.Author == critic.AuthorAI && msg.IsUnread {
				summary.HasUnreadAIMessages = true
			}
		}
	}

	return summary, nil
}

func (m *TestMessaging) GetConversationsSummary() ([]*critic.FileConversationSummary, error) {
	var summaries []*critic.FileConversationSummary
	for filePath := range m.conversations {
		summary, err := m.getFileConversationSummary(filePath)
		if err != nil {
			return nil, err
		}
		if summary != nil {
			summaries = append(summaries, summary)
		}
	}
	return summaries, nil
}

func (m *TestMessaging) ReplyToConversation(conversationUUID string, message string, author critic.Author) (*critic.Message, error) {
	for _, convs := range m.conversations {
		for _, c := range convs {
			if c.UUID == conversationUUID {
				now := time.Now()
				msg := critic.Message{
					UUID:      uuid.New().String(),
					Author:    author,
					Message:   message,
					CreatedAt: now,
					UpdatedAt: now,
					IsUnread:  author == critic.AuthorAI,
				}
				c.Messages = append(c.Messages, msg)
				c.UpdatedAt = now
				return &msg, nil
			}
		}
	}
	return nil, api.NotFoundError("conversation not found", conversationUUID)
}

func (m *TestMessaging) CreateConversation(author critic.Author, message, filePath string, lineNumber int, codeVersion string, context string) (*critic.Conversation, error) {
	now := time.Now()
	conv := &critic.Conversation{
		UUID:        uuid.New().String(),
		Status:      critic.StatusUnresolved,
		FilePath:    filePath,
		LineNumber:  lineNumber,
		CodeVersion: codeVersion,
		Context:     context,
		Messages: []critic.Message{
			{
				UUID:      uuid.New().String(),
				Author:    author,
				Message:   message,
				CreatedAt: now,
				UpdatedAt: now,
				IsUnread:  false,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	m.conversations[filePath] = append(m.conversations[filePath], conv)
	return conv, nil
}

func (m *TestMessaging) MarkConversationAs(conversationUUID string, update critic.ConversationUpdate) error {
	for _, convs := range m.conversations {
		for _, c := range convs {
			if c.UUID == conversationUUID {
				switch update {
				case critic.ConversationResolved:
					c.Status = critic.StatusResolved
				case critic.ConversationUnresolved:
					c.Status = critic.StatusUnresolved
				case critic.ConversationReadByAI:
					c.ReadByAI = true
				}
				return nil
			}
		}
	}
	return nil
}

func (m *TestMessaging) MarkMessageAs(messageUUID string, status critic.MessageReadStatus) error {
	return nil
}
func (m *TestMessaging) LoadRootConversation() (*critic.Conversation, error) {
	return &critic.Conversation{UUID: "root-conv"}, nil
}

func (m *TestMessaging) CreateExplanation(author critic.Author, comment, filePath string, lineNumber int, codeVersion string, context string) (*critic.Conversation, error) {
	return &critic.Conversation{UUID: "expl-1", ConversationType: critic.TypeExplanation, Status: critic.StatusInformal}, nil
}

func (m *TestMessaging) Close() error { return nil }

// TestConversationsScenario runs through all conversation-related GRPC endpoints
// in a realistic scenario: create a conversation, retrieve it, get summaries,
// reply to it, and verify the results.
func TestConversationsScenario(t *testing.T) {
	messaging := NewTestMessaging()
	srv := server.NewServer(server.Config{
		Messaging: messaging,
	})

	ctx := context.Background()
	filePath := "src/main.go"

	// Step 1: Create a conversation
	createReq := connect.NewRequest(&api.CreateConversationRequest{
		OldFile: filePath,
		OldLine: 10,
		NewFile: filePath,
		NewLine: 15,
		Comment: "This function needs better error handling",
	})

	createResp, err := srv.CreateConversation(ctx, createReq)
	assert.NoError(t, err, "CreateConversation should not return error")
	assert.True(t, createResp.Msg.GetSuccess(), "CreateConversation should succeed")
	assert.Nil(t, createResp.Msg.GetError(), "CreateConversation should have no error")

	// Step 2: Get conversations for the file
	getReq := connect.NewRequest(&api.GetConversationsRequest{Paths: []string{filePath}})
	getResp, err := srv.GetConversations(ctx, getReq)

	assert.NoError(t, err, "GetConversations should not return error")
	assert.Equals(t, len(getResp.Msg.GetConversations()), 1, "should have one conversation")

	conv := getResp.Msg.GetConversations()[0]
	assert.Equals(t, conv.GetFilePath(), filePath, "conversation file path should match")
	assert.Equals(t, conv.GetStatus(), api.ConversationStatus_CONVERSATION_STATUS_UNRESOLVED, "new conversation should be unresolved")
	assert.Equals(t, len(conv.GetMessages()), 1, "conversation should have one message")
	assert.Equals(t, conv.GetMessages()[0].GetContent(), "This function needs better error handling", "message content should match")
	assert.Equals(t, conv.GetMessages()[0].GetAuthor(), "human", "message author should be human")

	conversationID := conv.GetId()
	assert.True(t, conversationID != "", "conversation should have an ID")

	// Step 3: Get conversation summary
	summaryReq := connect.NewRequest(&api.GetConversationsSummaryRequest{})
	summaryResp, err := srv.GetConversationsSummary(ctx, summaryReq)

	assert.NoError(t, err, "GetConversationsSummary should not return error")
	assert.Equals(t, len(summaryResp.Msg.GetSummaries()), 1, "should have one file summary")

	summary := summaryResp.Msg.GetSummaries()[0]
	assert.Equals(t, summary.GetFilePath(), filePath, "summary file path should match")
	assert.Equals(t, summary.GetTotalCount(), int32(1), "total count should be 1")
	assert.Equals(t, summary.GetUnresolvedCount(), int32(1), "unresolved count should be 1")
	assert.Equals(t, summary.GetResolvedCount(), int32(0), "resolved count should be 0")

	// Step 4: Reply to the conversation
	replyReq := connect.NewRequest(&api.ReplyToConversationRequest{
		ConversationId: conversationID,
		Message:        "Good point, I'll add try-catch blocks",
	})

	replyResp, err := srv.ReplyToConversation(ctx, replyReq)
	assert.NoError(t, err, "ReplyToConversation should not return error")
	assert.True(t, replyResp.Msg.GetSuccess(), "ReplyToConversation should succeed")

	// Step 5: Get the conversation again to verify the reply was added
	getResp2, err := srv.GetConversations(ctx, getReq)

	assert.NoError(t, err, "GetConversations after reply should not return error")
	assert.Equals(t, len(getResp2.Msg.GetConversations()), 1, "should still have one conversation")

	conv2 := getResp2.Msg.GetConversations()[0]
	assert.Equals(t, len(conv2.GetMessages()), 2, "conversation should now have two messages")
	assert.Equals(t, conv2.GetMessages()[1].GetContent(), "Good point, I'll add try-catch blocks", "reply content should match")
}

// TestConversationsMultipleFiles tests conversations across multiple files
func TestConversationsMultipleFiles(t *testing.T) {
	messaging := NewTestMessaging()
	srv := server.NewServer(server.Config{
		Messaging: messaging,
	})

	ctx := context.Background()

	// Create conversations on multiple files
	files := []string{"src/main.go", "src/utils.go", "src/handler.go"}
	for _, file := range files {
		req := connect.NewRequest(&api.CreateConversationRequest{
			OldFile: file,
			OldLine: 1,
			NewFile: file,
			NewLine: 1,
			Comment: "Comment on " + file,
		})
		resp, err := srv.CreateConversation(ctx, req)
		assert.NoError(t, err, "CreateConversation should not fail for %s", file)
		assert.True(t, resp.Msg.GetSuccess(), "CreateConversation should succeed for %s", file)
	}

	// Verify summary shows all files
	summaryReq := connect.NewRequest(&api.GetConversationsSummaryRequest{})
	summaryResp, err := srv.GetConversationsSummary(ctx, summaryReq)

	assert.NoError(t, err, "GetConversationsSummary should not return error")
	assert.Equals(t, len(summaryResp.Msg.GetSummaries()), 3, "should have three file summaries")

	// Verify each file has exactly one conversation
	for _, summary := range summaryResp.Msg.GetSummaries() {
		assert.Equals(t, summary.GetTotalCount(), int32(1), "each file should have one conversation")
	}

	// Verify GetConversations returns only conversations for the requested file
	for _, file := range files {
		req := connect.NewRequest(&api.GetConversationsRequest{Paths: []string{file}})
		resp, err := srv.GetConversations(ctx, req)

		assert.NoError(t, err, "GetConversations should not fail for %s", file)
		assert.Equals(t, len(resp.Msg.GetConversations()), 1, "should have one conversation for %s", file)
		assert.Equals(t, resp.Msg.GetConversations()[0].GetFilePath(), file, "conversation should be for correct file")
	}
}

// TestConversationsEmptyFile tests getting conversations for a file with no conversations
func TestConversationsEmptyFile(t *testing.T) {
	messaging := NewTestMessaging()
	srv := server.NewServer(server.Config{
		Messaging: messaging,
	})

	ctx := context.Background()

	// Get conversations for a file with no conversations
	req := connect.NewRequest(&api.GetConversationsRequest{Paths: []string{"nonexistent.go"}})
	resp, err := srv.GetConversations(ctx, req)

	assert.NoError(t, err, "GetConversations should not return error for empty file")
	assert.Equals(t, len(resp.Msg.GetConversations()), 0, "should return empty list")
}

// TestConversationsMultipleMessagesInThread tests a conversation with multiple replies
func TestConversationsMultipleMessagesInThread(t *testing.T) {
	messaging := NewTestMessaging()
	srv := server.NewServer(server.Config{
		Messaging: messaging,
	})

	ctx := context.Background()
	filePath := "src/complex.go"

	// Create initial conversation
	createReq := connect.NewRequest(&api.CreateConversationRequest{
		OldFile: filePath,
		OldLine: 50,
		NewFile: filePath,
		NewLine: 55,
		Comment: "Why is this using a mutex here?",
	})

	createResp, err := srv.CreateConversation(ctx, createReq)
	assert.NoError(t, err, "CreateConversation should not return error")

	// Get the conversation to retrieve its ID
	getReq := connect.NewRequest(&api.GetConversationsRequest{Paths: []string{filePath}})
	getResp, err := srv.GetConversations(ctx, getReq)
	assert.NoError(t, err, "GetConversations should not return error")

	conversationID := getResp.Msg.GetConversations()[0].GetId()
	_ = createResp // silence unused variable

	// Add multiple replies
	replies := []string{
		"Good question. It prevents race conditions.",
		"Makes sense, thanks for explaining!",
		"No problem. Let me know if you have other questions.",
	}

	for _, reply := range replies {
		replyReq := connect.NewRequest(&api.ReplyToConversationRequest{
			ConversationId: conversationID,
			Message:        reply,
		})
		replyResp, err := srv.ReplyToConversation(ctx, replyReq)
		assert.NoError(t, err, "ReplyToConversation should not return error")
		assert.True(t, replyResp.Msg.GetSuccess(), "ReplyToConversation should succeed")
	}

	// Verify conversation has all messages
	finalResp, err := srv.GetConversations(ctx, getReq)
	assert.NoError(t, err, "GetConversations should not return error")

	conv := finalResp.Msg.GetConversations()[0]
	assert.Equals(t, len(conv.GetMessages()), 4, "conversation should have 4 messages (1 initial + 3 replies)")

	// Verify message order
	assert.Equals(t, conv.GetMessages()[0].GetContent(), "Why is this using a mutex here?", "first message should be the initial comment")
	assert.Equals(t, conv.GetMessages()[1].GetContent(), replies[0], "second message should be first reply")
	assert.Equals(t, conv.GetMessages()[2].GetContent(), replies[1], "third message should be second reply")
	assert.Equals(t, conv.GetMessages()[3].GetContent(), replies[2], "fourth message should be third reply")
}

// TestConversationsSummaryEmpty tests getting summary when no conversations exist
func TestConversationsSummaryEmpty(t *testing.T) {
	messaging := NewTestMessaging()
	srv := server.NewServer(server.Config{
		Messaging: messaging,
	})

	ctx := context.Background()

	summaryReq := connect.NewRequest(&api.GetConversationsSummaryRequest{})
	summaryResp, err := srv.GetConversationsSummary(ctx, summaryReq)

	assert.NoError(t, err, "GetConversationsSummary should not return error when empty")
	assert.Equals(t, len(summaryResp.Msg.GetSummaries()), 0, "should return empty summaries")
}

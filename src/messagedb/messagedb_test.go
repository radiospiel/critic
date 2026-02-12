package messagedb

import (
	"os"
	"testing"

	"github.com/radiospiel/critic/simple-go/assert"
	"github.com/radiospiel/critic/src/pkg/critic"
)

func setupTestDB(t *testing.T) (*DB, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "critic-test-*")
	assert.NoError(t, err, "failed to create temp dir")

	db, err := New(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create database: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

func TestCreateConversationWithMessage(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	conv, msg, err := db.createConversationWithMessage(AuthorHuman, "Test comment", "src/main.go", 42, "abc123", "context", ConversationTypeConversation)
	assert.NoError(t, err, "failed to create conversation")

	assert.NotEquals(t, conv.ID, "", "expected ID to be set")
	assert.Equals(t, conv.Status, StatusNew)
	assert.Equals(t, conv.FilePath, "src/main.go")
	assert.Equals(t, conv.Lineno, 42)
	assert.Equals(t, conv.Commit, "abc123")
	assert.Equals(t, conv.Context, "context")

	assert.Equals(t, msg.ID, conv.ID, "first message should share conversation ID")
	assert.Equals(t, msg.ConversationID, conv.ID)
	assert.Equals(t, msg.Author, AuthorHuman)
	assert.Equals(t, msg.ReadStatus, ReadStatusRead, "expected read_status to be read for human message")
	assert.Equals(t, msg.Message, "Test comment")
}

func TestCreateAIConversation(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	_, msg, err := db.createConversationWithMessage(AuthorAI, "AI response", "src/main.go", 42, "abc123", "content", ConversationTypeConversation)
	assert.NoError(t, err, "failed to create AI conversation")
	assert.Equals(t, msg.Author, AuthorAI)
	assert.Equals(t, msg.ReadStatus, ReadStatusUnread, "expected read_status to be unread for AI message")
}

func TestCreateReply(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create parent conversation
	conv, _, err := db.createConversationWithMessage(AuthorHuman, "Parent comment", "src/main.go", 42, "abc123", "content", ConversationTypeConversation)
	assert.NoError(t, err, "failed to create parent conversation")

	// Create reply
	reply, err := db.CreateReply(AuthorAI, "AI reply", conv.ID)
	assert.NoError(t, err, "failed to create reply")

	assert.Equals(t, reply.ConversationID, conv.ID)
}

func TestGetMessage(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	_, created, err := db.createConversationWithMessage(AuthorHuman, "Test comment", "src/main.go", 42, "abc123", "content", ConversationTypeConversation)
	assert.NoError(t, err, "failed to create conversation")

	retrieved, err := db.GetMessage(created.ID)
	assert.NoError(t, err, "failed to get message")
	assert.NotNil(t, retrieved, "expected message to be found")
	assert.Equals(t, retrieved.ID, created.ID)
	assert.Equals(t, retrieved.Message, created.Message)

	// Test non-existent message
	notFound, err := db.GetMessage("non-existent-id")
	assert.NoError(t, err)
	assert.Nil(t, notFound, "expected nil for non-existent message")
}

func TestGetThreadMessages(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create parent conversation
	conv, parentMsg, err := db.createConversationWithMessage(AuthorHuman, "Parent comment", "src/main.go", 42, "abc123", "content", ConversationTypeConversation)
	assert.NoError(t, err, "failed to create parent")

	// Create replies
	reply1, err := db.CreateReply(AuthorAI, "AI reply 1", conv.ID)
	assert.NoError(t, err, "failed to create reply 1")

	reply2, err := db.CreateReply(AuthorHuman, "Human reply 2", conv.ID)
	assert.NoError(t, err, "failed to create reply 2")

	// Get thread
	thread, err := db.GetThreadMessages(conv.ID)
	assert.NoError(t, err, "failed to get thread")
	assert.Equals(t, len(thread), 3, "expected 3 messages in thread")

	// Check order (should be by created_at)
	assert.Equals(t, thread[0].ID, parentMsg.ID, "expected first message to be parent")
	assert.Equals(t, thread[1].ID, reply1.ID, "expected second message to be reply1")
	assert.Equals(t, thread[2].ID, reply2.ID, "expected third message to be reply2")
}

func TestGetUnresolvedConversations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create some conversations
	conv1, _, _ := db.createConversationWithMessage(AuthorHuman, "Comment 1", "src/main.go", 10, "abc123", "content", ConversationTypeConversation)
	conv2, _, _ := db.createConversationWithMessage(AuthorHuman, "Comment 2", "src/main.go", 20, "abc123", "content", ConversationTypeConversation)
	conv3, _, _ := db.createConversationWithMessage(AuthorHuman, "Comment 3", "src/util.go", 5, "abc123", "content", ConversationTypeConversation)

	// Create a reply (should not appear in conversations)
	db.CreateReply(AuthorAI, "Reply to conv1", conv1.ID)

	// Mark conv2 as resolved
	db.MarkConversationAs(conv2.ID, critic.ConversationResolved)

	// Get unresolved
	unresolved, err := db.GetUnresolvedConversations()
	assert.NoError(t, err, "failed to get unresolved")
	assert.Equals(t, len(unresolved), 2, "expected 2 unresolved conversations")

	// Should be conv1 and conv3 (not conv2 which is resolved)
	ids := []string{unresolved[0].ID, unresolved[1].ID}
	assert.Contains(t, ids, conv1.ID, "expected conv1 in unresolved conversations")
	assert.Contains(t, ids, conv3.ID, "expected conv3 in unresolved conversations")
}

func TestGetConversationsByFile(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create conversations in different files
	conv1, _, _ := db.createConversationWithMessage(AuthorHuman, "Comment 1", "src/main.go", 10, "abc123", "content", ConversationTypeConversation)
	conv2, _, _ := db.createConversationWithMessage(AuthorHuman, "Comment 2", "src/main.go", 20, "abc123", "content", ConversationTypeConversation)
	db.createConversationWithMessage(AuthorHuman, "Comment 3", "src/util.go", 5, "abc123", "content", ConversationTypeConversation)

	// Create a reply (should not affect conversation count)
	db.CreateReply(AuthorAI, "Reply", conv1.ID)

	// Get conversations for src/main.go
	conversations, err := db.GetConversationsByFile("src/main.go")
	assert.NoError(t, err, "failed to get conversations by file")
	assert.Equals(t, len(conversations), 2, "expected 2 conversations for src/main.go")

	// Should be ordered by line number
	assert.Equals(t, conversations[0].ID, conv1.ID, "expected conv1 to be first (line 10)")
	assert.Equals(t, conversations[1].ID, conv2.ID, "expected conv2 to be second (line 20)")
}

func TestMarkConversationAsResolved(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create conversation with a reply
	conv, _, _ := db.createConversationWithMessage(AuthorHuman, "Parent", "src/main.go", 10, "abc123", "content", ConversationTypeConversation)
	db.CreateReply(AuthorAI, "Reply", conv.ID)

	// Mark as resolved
	err := db.MarkConversationAs(conv.ID, critic.ConversationResolved)
	assert.NoError(t, err, "failed to mark as resolved")

	// Check conversation record
	convAfter, _ := db.getConversation(conv.ID)
	assert.Equals(t, convAfter.Status, StatusResolved)
}

func TestMarkMessageAsRead(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create AI conversation (message should be unread by default)
	_, msg, _ := db.createConversationWithMessage(AuthorAI, "AI comment", "src/main.go", 10, "abc123", "content", ConversationTypeConversation)
	assert.Equals(t, msg.ReadStatus, ReadStatusUnread, "expected AI message to be unread initially")

	// Mark as read
	err := db.MarkMessageAs(msg.ID, critic.MessageRead)
	assert.NoError(t, err, "failed to mark as read")

	// Check status
	msgAfter, _ := db.GetMessage(msg.ID)
	assert.Equals(t, msgAfter.ReadStatus, ReadStatusRead)
}

func TestGetFilesWithUnreadAIMessages(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create some AI conversations
	db.createConversationWithMessage(AuthorAI, "AI comment 1", "src/main.go", 10, "abc123", "content", ConversationTypeConversation)
	conv2, msg2, _ := db.createConversationWithMessage(AuthorAI, "AI comment 2", "src/util.go", 20, "abc123", "content", ConversationTypeConversation)
	db.createConversationWithMessage(AuthorAI, "AI comment 3", "src/main.go", 30, "abc123", "content", ConversationTypeConversation)

	// Create human conversation (should not affect result)
	db.createConversationWithMessage(AuthorHuman, "Human comment", "src/test.go", 5, "abc123", "content", ConversationTypeConversation)

	// Mark one as read
	_ = conv2 // suppress unused warning
	db.MarkMessageAs(msg2.ID, critic.MessageRead)

	// Get files with unread
	files, err := db.GetFilesWithUnreadAIMessages()
	assert.NoError(t, err, "failed to get files with unread")

	// Should only have src/main.go (src/util.go was marked as read, src/test.go is human)
	assert.Equals(t, len(files), 1, "expected 1 file with unread messages")
	assert.Equals(t, files[0], "src/main.go")
}

func TestUpdateConversationStatus(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	conv, _, _ := db.createConversationWithMessage(AuthorHuman, "Comment", "src/main.go", 10, "abc123", "content", ConversationTypeConversation)

	err := db.UpdateConversationStatus(conv.ID, StatusDelivered)
	assert.NoError(t, err, "failed to update status")

	convAfter, _ := db.getConversation(conv.ID)
	assert.Equals(t, convAfter.Status, StatusDelivered)
}

func TestGetConversationsReturnsOnlyTopLevel(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create three root conversations
	conv1, _, _ := db.createConversationWithMessage(AuthorHuman, "Conversation 1", "src/main.go", 10, "abc123", "content", ConversationTypeConversation)
	conv2, _, _ := db.createConversationWithMessage(AuthorHuman, "Conversation 2", "src/main.go", 20, "abc123", "content", ConversationTypeConversation)
	conv3, _, _ := db.createConversationWithMessage(AuthorHuman, "Conversation 3", "src/util.go", 5, "abc123", "content", ConversationTypeConversation)

	// Create replies to conv1 (should NOT appear in GetConversations)
	db.CreateReply(AuthorAI, "Reply 1 to conv1", conv1.ID)
	db.CreateReply(AuthorHuman, "Reply 2 to conv1", conv1.ID)

	// Create a reply to conv2
	db.CreateReply(AuthorAI, "Reply to conv2", conv2.ID)

	// Get all conversations
	conversations, err := db.GetConversations("", nil)
	assert.NoError(t, err, "failed to get conversations")

	// Should only return 3 top-level conversations, not the 4 replies
	assert.Equals(t, len(conversations), 3, "expected 3 conversations")

	// Extract UUIDs for comparison
	uuids := make([]string, len(conversations))
	for i, conv := range conversations {
		uuids[i] = conv.UUID
	}

	// Verify all returned IDs are root conversations
	assert.Contains(t, uuids, conv1.ID, "expected conv1 in conversations")
	assert.Contains(t, uuids, conv2.ID, "expected conv2 in conversations")
	assert.Contains(t, uuids, conv3.ID, "expected conv3 in conversations")
}

func TestGetConversationsWithStatusFilter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create conversations
	conv1, _, _ := db.createConversationWithMessage(AuthorHuman, "Unresolved 1", "src/main.go", 10, "abc123", "content", ConversationTypeConversation)
	conv2, _, _ := db.createConversationWithMessage(AuthorHuman, "Unresolved 2", "src/main.go", 20, "abc123", "content", ConversationTypeConversation)
	conv3, _, _ := db.createConversationWithMessage(AuthorHuman, "To be resolved", "src/util.go", 5, "abc123", "content", ConversationTypeConversation)

	// Add replies (should not affect conversation count)
	db.CreateReply(AuthorAI, "Reply to conv1", conv1.ID)
	db.CreateReply(AuthorAI, "Reply to conv3", conv3.ID)

	// Mark conv3 as resolved
	db.MarkConversationAs(conv3.ID, critic.ConversationResolved)

	// Get unresolved conversations
	unresolved, err := db.GetConversations("unresolved", nil)
	assert.NoError(t, err, "failed to get unresolved conversations")
	assert.Equals(t, len(unresolved), 2, "expected 2 unresolved conversations")

	unresolvedUUIDs := make([]string, len(unresolved))
	for i, conv := range unresolved {
		unresolvedUUIDs[i] = conv.UUID
	}
	assert.Contains(t, unresolvedUUIDs, conv1.ID, "expected conv1 in unresolved conversations")
	assert.Contains(t, unresolvedUUIDs, conv2.ID, "expected conv2 in unresolved conversations")

	// Get resolved conversations
	resolved, err := db.GetConversations("resolved", nil)
	assert.NoError(t, err, "failed to get resolved conversations")
	assert.Equals(t, len(resolved), 1, "expected 1 resolved conversation")
	assert.Equals(t, resolved[0].UUID, conv3.ID, "expected conv3 in resolved conversations")
}

func TestGetConversationsWithPathFilter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	db.createConversationWithMessage(AuthorHuman, "Main comment", "src/main.go", 10, "abc123", "content", ConversationTypeConversation)
	db.createConversationWithMessage(AuthorHuman, "Util comment", "src/util.go", 5, "abc123", "content", ConversationTypeConversation)
	db.createConversationWithMessage(AuthorHuman, "Test comment", "src/test.go", 1, "abc123", "content", ConversationTypeConversation)

	// Filter to single path
	convs, err := db.GetConversations("", []string{"src/main.go"})
	assert.NoError(t, err, "failed to get conversations for path")
	assert.Equals(t, len(convs), 1, "expected 1 conversation for src/main.go")
	assert.Equals(t, convs[0].FilePath, "src/main.go")

	// Filter to multiple paths
	convs, err = db.GetConversations("", []string{"src/main.go", "src/util.go"})
	assert.NoError(t, err, "failed to get conversations for paths")
	assert.Equals(t, len(convs), 2, "expected 2 conversations for two paths")

	// Combined status + path filter
	db.MarkConversationAs(convs[0].UUID, critic.ConversationResolved)
	unresolved, err := db.GetConversations("unresolved", []string{"src/main.go", "src/util.go"})
	assert.NoError(t, err, "failed to get filtered conversations")
	assert.Equals(t, len(unresolved), 1, "expected 1 unresolved conversation after resolving one")
}

func TestGetFullConversations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create conversations with replies
	conv1, _, _ := db.createConversationWithMessage(AuthorHuman, "Comment 1", "src/main.go", 10, "abc123", "content", ConversationTypeConversation)
	db.CreateReply(AuthorAI, "Reply to 1", conv1.ID)

	conv2, _, _ := db.createConversationWithMessage(AuthorHuman, "Comment 2", "src/util.go", 5, "def456", "content", ConversationTypeConversation)
	db.CreateReply(AuthorAI, "Reply to 2a", conv2.ID)
	db.CreateReply(AuthorHuman, "Reply to 2b", conv2.ID)

	// Batch fetch
	convs, err := db.GetFullConversations([]string{conv1.ID, conv2.ID})
	assert.NoError(t, err, "failed to batch-fetch conversations")
	assert.Equals(t, len(convs), 2, "expected 2 conversations")

	// First conversation should have 2 messages
	assert.Equals(t, convs[0].UUID, conv1.ID)
	assert.Equals(t, len(convs[0].Messages), 2, "expected 2 messages in first conversation")
	assert.Equals(t, convs[0].FilePath, "src/main.go")

	// Second conversation should have 3 messages
	assert.Equals(t, convs[1].UUID, conv2.ID)
	assert.Equals(t, len(convs[1].Messages), 3, "expected 3 messages in second conversation")
	assert.Equals(t, convs[1].FilePath, "src/util.go")

	// Empty input returns nil
	empty, err := db.GetFullConversations([]string{})
	assert.NoError(t, err, "empty input should not error")
	assert.Nil(t, empty, "empty input should return nil")
}

func TestMarkConversationAsReadByAI(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create conversation and replies
	conv, _, err := db.createConversationWithMessage(AuthorHuman, "Parent comment", "src/main.go", 10, "abc123", "content", ConversationTypeConversation)
	assert.NoError(t, err, "failed to create conversation")
	assert.False(t, conv.ReadByAI, "expected ReadByAI to be false initially")

	db.CreateReply(AuthorAI, "AI reply", conv.ID)
	db.CreateReply(AuthorHuman, "Human reply", conv.ID)

	// Mark conversation as read by AI
	err = db.MarkConversationAs(conv.ID, critic.ConversationReadByAI)
	assert.NoError(t, err, "failed to mark as read by AI")

	// Verify conversation record is marked
	convAfter, _ := db.getConversation(conv.ID)
	assert.True(t, convAfter.ReadByAI, "expected conversation to be marked as read by AI")
}

func TestReadByAIFieldDefaultsFalse(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	conv, _, _ := db.createConversationWithMessage(AuthorHuman, "Comment", "src/main.go", 10, "abc123", "content", ConversationTypeConversation)

	// Verify ReadByAI defaults to false
	retrieved, _ := db.getConversation(conv.ID)
	assert.False(t, retrieved.ReadByAI, "expected ReadByAI to default to false")
}

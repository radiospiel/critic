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

func TestCreateMessage(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	msg, err := db.CreateMessage(AuthorHuman, "Test comment", "src/main.go", 42, "abc123", "context")
	assert.NoError(t, err, "failed to create message")

	assert.NotEquals(t, msg.ID, "", "expected ID to be set")
	assert.Equals(t, msg.Author, AuthorHuman)
	assert.Equals(t, msg.Status, StatusNew)
	assert.Equals(t, msg.ReadStatus, ReadStatusRead, "expected read_status to be read for human message")
	assert.Equals(t, msg.Message, "Test comment")
	assert.Equals(t, msg.FilePath, "src/main.go")
	assert.Equals(t, msg.Lineno, 42)
	assert.Equals(t, msg.ConversationID, msg.ID, "expected conversation_id to equal id for root message")
}

func TestCreateAIMessage(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	msg, err := db.CreateMessage(AuthorAI, "AI response", "src/main.go", 42, "abc123", "content")
	assert.NoError(t, err, "failed to create AI message")
	assert.Equals(t, msg.Author, AuthorAI)
	assert.Equals(t, msg.ReadStatus, ReadStatusUnread, "expected read_status to be unread for AI message")
}

func TestCreateReply(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create parent message
	parent, err := db.CreateMessage(AuthorHuman, "Parent comment", "src/main.go", 42, "abc123", "conent")
	assert.NoError(t, err, "failed to create parent message")

	// Create reply
	reply, err := db.CreateReply(AuthorAI, "AI reply", parent.ID)
	assert.NoError(t, err, "failed to create reply")

	assert.Equals(t, reply.ConversationID, parent.ID)
	assert.Equals(t, reply.FilePath, parent.FilePath)
	assert.Equals(t, reply.Lineno, parent.Lineno)
}

func TestGetMessage(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	created, err := db.CreateMessage(AuthorHuman, "Test comment", "src/main.go", 42, "abc123", "content")
	assert.NoError(t, err, "failed to create message")

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

	// Create parent message
	parent, err := db.CreateMessage(AuthorHuman, "Parent comment", "src/main.go", 42, "abc123", "content")
	assert.NoError(t, err, "failed to create parent")

	// Create replies
	reply1, err := db.CreateReply(AuthorAI, "AI reply 1", parent.ID)
	assert.NoError(t, err, "failed to create reply 1")

	reply2, err := db.CreateReply(AuthorHuman, "Human reply 2", parent.ID)
	assert.NoError(t, err, "failed to create reply 2")

	// Get thread
	thread, err := db.GetThreadMessages(parent.ID)
	assert.NoError(t, err, "failed to get thread")
	assert.Equals(t, len(thread), 3, "expected 3 messages in thread")

	// Check order (should be by created_at)
	assert.Equals(t, thread[0].ID, parent.ID, "expected first message to be parent")
	assert.Equals(t, thread[1].ID, reply1.ID, "expected second message to be reply1")
	assert.Equals(t, thread[2].ID, reply2.ID, "expected third message to be reply2")
}

func TestGetUnresolvedRootMessages(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create some messages
	msg1, _ := db.CreateMessage(AuthorHuman, "Comment 1", "src/main.go", 10, "abc123", "content")
	msg2, _ := db.CreateMessage(AuthorHuman, "Comment 2", "src/main.go", 20, "abc123", "content")
	msg3, _ := db.CreateMessage(AuthorHuman, "Comment 3", "src/util.go", 5, "abc123", "content")

	// Create a reply (should not appear in root messages)
	db.CreateReply(AuthorAI, "Reply to msg1", msg1.ID)

	// Mark msg2 as resolved
	db.MarkConversationAs(msg2.ID, critic.ConversationResolved)

	// Get unresolved
	unresolved, err := db.GetUnresolvedRootMessages()
	assert.NoError(t, err, "failed to get unresolved")
	assert.Equals(t, len(unresolved), 2, "expected 2 unresolved messages")

	// Should be msg1 and msg3 (not msg2 which is resolved)
	uuids := []string{unresolved[0].ID, unresolved[1].ID}
	assert.Contains(t, uuids, msg1.ID, "expected msg1 in unresolved messages")
	assert.Contains(t, uuids, msg3.ID, "expected msg3 in unresolved messages")
}

func TestGetMessagesByFile(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create messages in different files
	msg1, _ := db.CreateMessage(AuthorHuman, "Comment 1", "src/main.go", 10, "abc123", "content")
	msg2, _ := db.CreateMessage(AuthorHuman, "Comment 2", "src/main.go", 20, "abc123", "content")
	db.CreateMessage(AuthorHuman, "Comment 3", "src/util.go", 5, "abc123", "content")

	// Create a reply (should not appear)
	db.CreateReply(AuthorAI, "Reply", msg1.ID)

	// Get messages for src/main.go
	messages, err := db.GetMessagesByFile("src/main.go")
	assert.NoError(t, err, "failed to get messages by file")
	assert.Equals(t, len(messages), 2, "expected 2 messages for src/main.go")

	// Should be ordered by line number
	assert.Equals(t, messages[0].ID, msg1.ID, "expected msg1 to be first (line 10)")
	assert.Equals(t, messages[1].ID, msg2.ID, "expected msg2 to be second (line 20)")
}

func TestMarkConversationAsResolved(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create parent and replies
	parent, _ := db.CreateMessage(AuthorHuman, "Parent", "src/main.go", 10, "abc123", "content")
	reply, _ := db.CreateReply(AuthorAI, "Reply", parent.ID)

	// Mark as resolved
	err := db.MarkConversationAs(parent.ID, critic.ConversationResolved)
	assert.NoError(t, err, "failed to mark as resolved")

	// Check parent
	parentAfter, _ := db.GetMessage(parent.ID)
	assert.Equals(t, parentAfter.Status, StatusResolved)

	// Check reply (should also be resolved)
	replyAfter, _ := db.GetMessage(reply.ID)
	assert.Equals(t, replyAfter.Status, StatusResolved)
}

func TestMarkMessageAsRead(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create AI message (should be unread by default)
	msg, _ := db.CreateMessage(AuthorAI, "AI comment", "src/main.go", 10, "abc123", "content")
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

	// Create some AI messages
	db.CreateMessage(AuthorAI, "AI comment 1", "src/main.go", 10, "abc123", "content")
	msg2, _ := db.CreateMessage(AuthorAI, "AI comment 2", "src/util.go", 20, "abc123", "content")
	db.CreateMessage(AuthorAI, "AI comment 3", "src/main.go", 30, "abc123", "content")

	// Create human message (should not affect result)
	db.CreateMessage(AuthorHuman, "Human comment", "src/test.go", 5, "abc123", "content")

	// Mark one as read
	db.MarkMessageAs(msg2.ID, critic.MessageRead)

	// Get files with unread
	files, err := db.GetFilesWithUnreadAIMessages()
	assert.NoError(t, err, "failed to get files with unread")

	// Should only have src/main.go (src/util.go was marked as read, src/test.go is human)
	assert.Equals(t, len(files), 1, "expected 1 file with unread messages")
	assert.Equals(t, files[0], "src/main.go")
}

func TestUpdateMessageStatus(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	msg, _ := db.CreateMessage(AuthorHuman, "Comment", "src/main.go", 10, "abc123", "content")

	err := db.UpdateMessageStatus(msg.ID, StatusDelivered)
	assert.NoError(t, err, "failed to update status")

	msgAfter, _ := db.GetMessage(msg.ID)
	assert.Equals(t, msgAfter.Status, StatusDelivered)
}

func TestGetConversationsReturnsOnlyTopLevel(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create three root conversations
	conv1, _ := db.CreateMessage(AuthorHuman, "Conversation 1", "src/main.go", 10, "abc123", "content")
	conv2, _ := db.CreateMessage(AuthorHuman, "Conversation 2", "src/main.go", 20, "abc123", "content")
	conv3, _ := db.CreateMessage(AuthorHuman, "Conversation 3", "src/util.go", 5, "abc123", "content")

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
	conv1, _ := db.CreateMessage(AuthorHuman, "Unresolved 1", "src/main.go", 10, "abc123", "content")
	conv2, _ := db.CreateMessage(AuthorHuman, "Unresolved 2", "src/main.go", 20, "abc123", "content")
	conv3, _ := db.CreateMessage(AuthorHuman, "To be resolved", "src/util.go", 5, "abc123", "content")

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

	db.CreateMessage(AuthorHuman, "Main comment", "src/main.go", 10, "abc123", "content")
	db.CreateMessage(AuthorHuman, "Util comment", "src/util.go", 5, "abc123", "content")
	db.CreateMessage(AuthorHuman, "Test comment", "src/test.go", 1, "abc123", "content")

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
	msg1, _ := db.CreateMessage(AuthorHuman, "Comment 1", "src/main.go", 10, "abc123", "content")
	db.CreateReply(AuthorAI, "Reply to 1", msg1.ID)

	msg2, _ := db.CreateMessage(AuthorHuman, "Comment 2", "src/util.go", 5, "def456", "content")
	db.CreateReply(AuthorAI, "Reply to 2a", msg2.ID)
	db.CreateReply(AuthorHuman, "Reply to 2b", msg2.ID)

	// Batch fetch
	convs, err := db.GetFullConversations([]string{msg1.ID, msg2.ID})
	assert.NoError(t, err, "failed to batch-fetch conversations")
	assert.Equals(t, len(convs), 2, "expected 2 conversations")

	// First conversation should have 2 messages
	assert.Equals(t, convs[0].UUID, msg1.ID)
	assert.Equals(t, len(convs[0].Messages), 2, "expected 2 messages in first conversation")
	assert.Equals(t, convs[0].FilePath, "src/main.go")

	// Second conversation should have 3 messages
	assert.Equals(t, convs[1].UUID, msg2.ID)
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

	// Create parent message and replies
	parent, err := db.CreateMessage(AuthorHuman, "Parent comment", "src/main.go", 10, "abc123", "content")
	assert.NoError(t, err, "failed to create parent")
	assert.False(t, parent.ReadByAI, "expected ReadByAI to be false initially")

	// Create replies
	reply1, err := db.CreateReply(AuthorAI, "AI reply", parent.ID)
	assert.NoError(t, err, "failed to create reply1")

	reply2, err := db.CreateReply(AuthorHuman, "Human reply", parent.ID)
	assert.NoError(t, err, "failed to create reply2")

	// Mark conversation as read by AI
	err = db.MarkConversationAs(parent.ID, critic.ConversationReadByAI)
	assert.NoError(t, err, "failed to mark as read by AI")

	// Verify all messages in conversation are marked
	parentAfter, _ := db.GetMessage(parent.ID)
	assert.True(t, parentAfter.ReadByAI, "expected parent to be marked as read by AI")

	reply1After, _ := db.GetMessage(reply1.ID)
	assert.True(t, reply1After.ReadByAI, "expected reply1 to be marked as read by AI")

	reply2After, _ := db.GetMessage(reply2.ID)
	assert.True(t, reply2After.ReadByAI, "expected reply2 to be marked as read by AI")
}

func TestReadByAIFieldInGetThreadMessages(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create conversation
	parent, _ := db.CreateMessage(AuthorHuman, "Parent", "src/main.go", 10, "abc123", "content")
	db.CreateReply(AuthorAI, "Reply", parent.ID)

	// Mark as read by AI
	db.MarkConversationAs(parent.ID, critic.ConversationReadByAI)

	// Get thread messages and verify ReadByAI field
	thread, err := db.GetThreadMessages(parent.ID)
	assert.NoError(t, err, "failed to get thread messages")
	assert.Equals(t, len(thread), 2, "expected 2 messages in thread")

	for _, msg := range thread {
		assert.True(t, msg.ReadByAI, "expected message %s to be marked as read by AI", msg.ID)
	}
}

func TestReadByAIFieldDefaultsFalse(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create message
	msg, _ := db.CreateMessage(AuthorHuman, "Comment", "src/main.go", 10, "abc123", "content")

	// Verify ReadByAI defaults to false
	retrieved, _ := db.GetMessage(msg.ID)
	assert.False(t, retrieved.ReadByAI, "expected ReadByAI to default to false")
}

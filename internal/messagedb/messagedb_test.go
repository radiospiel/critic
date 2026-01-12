package messagedb

import (
	"os"
	"testing"

	"git.15b.it/eno/critic/simple-go/assert"
)

func setupTestDB(t *testing.T) (*DB, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "critic-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

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

	msg, err := db.CreateMessage(AuthorHuman, "Test comment", "src/main.go", 42, "abc123")
	if err != nil {
		t.Fatalf("failed to create message: %v", err)
	}

	if msg.ID == "" {
		t.Error("expected ID to be set")
	}
	if msg.Author != AuthorHuman {
		t.Errorf("expected author to be %s, got %s", AuthorHuman, msg.Author)
	}
	if msg.Status != StatusNew {
		t.Errorf("expected status to be %s, got %s", StatusNew, msg.Status)
	}
	if msg.ReadStatus != ReadStatusRead {
		t.Errorf("expected read_status to be %s for human message, got %s", ReadStatusRead, msg.ReadStatus)
	}
	if msg.Message != "Test comment" {
		t.Errorf("expected message to be 'Test comment', got %s", msg.Message)
	}
	if msg.FilePath != "src/main.go" {
		t.Errorf("expected file_path to be 'src/main.go', got %s", msg.FilePath)
	}
	if msg.LineNumber != 42 {
		t.Errorf("expected line_number to be 42, got %d", msg.LineNumber)
	}
	if msg.ConversationID != msg.ID {
		t.Errorf("expected conversation_id to equal id for root message, got %s", msg.ConversationID)
	}
}

func TestCreateAIMessage(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	msg, err := db.CreateMessage(AuthorAI, "AI response", "src/main.go", 42, "abc123")
	if err != nil {
		t.Fatalf("failed to create AI message: %v", err)
	}

	if msg.Author != AuthorAI {
		t.Errorf("expected author to be %s, got %s", AuthorAI, msg.Author)
	}
	if msg.ReadStatus != ReadStatusUnread {
		t.Errorf("expected read_status to be %s for AI message, got %s", ReadStatusUnread, msg.ReadStatus)
	}
}

func TestCreateReply(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create parent message
	parent, err := db.CreateMessage(AuthorHuman, "Parent comment", "src/main.go", 42, "abc123")
	if err != nil {
		t.Fatalf("failed to create parent message: %v", err)
	}

	// Create reply
	reply, err := db.CreateReply(AuthorAI, "AI reply", parent.ID)
	if err != nil {
		t.Fatalf("failed to create reply: %v", err)
	}

	if reply.ConversationID != parent.ID {
		t.Errorf("expected conversation_id to be %s, got %s", parent.ID, reply.ConversationID)
	}
	if reply.FilePath != parent.FilePath {
		t.Errorf("expected file_path to match parent (%s), got %s", parent.FilePath, reply.FilePath)
	}
	if reply.LineNumber != parent.LineNumber {
		t.Errorf("expected line_number to match parent (%d), got %d", parent.LineNumber, reply.LineNumber)
	}
}

func TestGetMessage(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	created, err := db.CreateMessage(AuthorHuman, "Test comment", "src/main.go", 42, "abc123")
	if err != nil {
		t.Fatalf("failed to create message: %v", err)
	}

	retrieved, err := db.GetMessage(created.ID)
	if err != nil {
		t.Fatalf("failed to get message: %v", err)
	}

	if retrieved == nil {
		t.Fatal("expected message to be found")
	}
	if retrieved.ID != created.ID {
		t.Errorf("expected ID to be %s, got %s", created.ID, retrieved.ID)
	}
	if retrieved.Message != created.Message {
		t.Errorf("expected message to be %s, got %s", created.Message, retrieved.Message)
	}

	// Test non-existent message
	notFound, err := db.GetMessage("non-existent-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notFound != nil {
		t.Error("expected nil for non-existent message")
	}
}

func TestGetThreadMessages(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create parent message
	parent, err := db.CreateMessage(AuthorHuman, "Parent comment", "src/main.go", 42, "abc123")
	if err != nil {
		t.Fatalf("failed to create parent: %v", err)
	}

	// Create replies
	reply1, err := db.CreateReply(AuthorAI, "AI reply 1", parent.ID)
	if err != nil {
		t.Fatalf("failed to create reply 1: %v", err)
	}

	reply2, err := db.CreateReply(AuthorHuman, "Human reply 2", parent.ID)
	if err != nil {
		t.Fatalf("failed to create reply 2: %v", err)
	}

	// Get thread
	thread, err := db.GetThreadMessages(parent.ID)
	if err != nil {
		t.Fatalf("failed to get thread: %v", err)
	}

	if len(thread) != 3 {
		t.Fatalf("expected 3 messages in thread, got %d", len(thread))
	}

	// Check order (should be by created_at)
	if thread[0].ID != parent.ID {
		t.Error("expected first message to be parent")
	}
	if thread[1].ID != reply1.ID {
		t.Error("expected second message to be reply1")
	}
	if thread[2].ID != reply2.ID {
		t.Error("expected third message to be reply2")
	}
}

func TestGetUnresolvedRootMessages(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create some messages
	msg1, _ := db.CreateMessage(AuthorHuman, "Comment 1", "src/main.go", 10, "abc123")
	msg2, _ := db.CreateMessage(AuthorHuman, "Comment 2", "src/main.go", 20, "abc123")
	msg3, _ := db.CreateMessage(AuthorHuman, "Comment 3", "src/util.go", 5, "abc123")

	// Create a reply (should not appear in root messages)
	db.CreateReply(AuthorAI, "Reply to msg1", msg1.ID)

	// Mark msg2 as resolved
	db.MarkAsResolved(msg2.ID)

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
	msg1, _ := db.CreateMessage(AuthorHuman, "Comment 1", "src/main.go", 10, "abc123")
	msg2, _ := db.CreateMessage(AuthorHuman, "Comment 2", "src/main.go", 20, "abc123")
	db.CreateMessage(AuthorHuman, "Comment 3", "src/util.go", 5, "abc123")

	// Create a reply (should not appear)
	db.CreateReply(AuthorAI, "Reply", msg1.ID)

	// Get messages for src/main.go
	messages, err := db.GetMessagesByFile("src/main.go")
	if err != nil {
		t.Fatalf("failed to get messages by file: %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("expected 2 messages for src/main.go, got %d", len(messages))
	}

	// Should be ordered by line number
	if messages[0].ID != msg1.ID {
		t.Error("expected msg1 to be first (line 10)")
	}
	if messages[1].ID != msg2.ID {
		t.Error("expected msg2 to be second (line 20)")
	}
}

func TestMarkAsResolved(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create parent and replies
	parent, _ := db.CreateMessage(AuthorHuman, "Parent", "src/main.go", 10, "abc123")
	reply, _ := db.CreateReply(AuthorAI, "Reply", parent.ID)

	// Mark as resolved
	err := db.MarkAsResolved(parent.ID)
	if err != nil {
		t.Fatalf("failed to mark as resolved: %v", err)
	}

	// Check parent
	parentAfter, _ := db.GetMessage(parent.ID)
	if parentAfter.Status != StatusResolved {
		t.Errorf("expected parent status to be %s, got %s", StatusResolved, parentAfter.Status)
	}

	// Check reply (should also be resolved)
	replyAfter, _ := db.GetMessage(reply.ID)
	if replyAfter.Status != StatusResolved {
		t.Errorf("expected reply status to be %s, got %s", StatusResolved, replyAfter.Status)
	}
}

func TestMarkAsRead(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create AI message (should be unread by default)
	msg, _ := db.CreateMessage(AuthorAI, "AI comment", "src/main.go", 10, "abc123")

	if msg.ReadStatus != ReadStatusUnread {
		t.Fatalf("expected AI message to be unread initially, got %s", msg.ReadStatus)
	}

	// Mark as read
	err := db.MarkAsRead(msg.ID)
	if err != nil {
		t.Fatalf("failed to mark as read: %v", err)
	}

	// Check status
	msgAfter, _ := db.GetMessage(msg.ID)
	if msgAfter.ReadStatus != ReadStatusRead {
		t.Errorf("expected read_status to be %s, got %s", ReadStatusRead, msgAfter.ReadStatus)
	}
}

func TestGetFilesWithUnreadAIMessages(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create some AI messages
	db.CreateMessage(AuthorAI, "AI comment 1", "src/main.go", 10, "abc123")
	msg2, _ := db.CreateMessage(AuthorAI, "AI comment 2", "src/util.go", 20, "abc123")
	db.CreateMessage(AuthorAI, "AI comment 3", "src/main.go", 30, "abc123")

	// Create human message (should not affect result)
	db.CreateMessage(AuthorHuman, "Human comment", "src/test.go", 5, "abc123")

	// Mark one as read
	db.MarkAsRead(msg2.ID)

	// Get files with unread
	files, err := db.GetFilesWithUnreadAIMessages()
	if err != nil {
		t.Fatalf("failed to get files with unread: %v", err)
	}

	// Should only have src/main.go (src/util.go was marked as read, src/test.go is human)
	if len(files) != 1 {
		t.Fatalf("expected 1 file with unread messages, got %d: %v", len(files), files)
	}
	if files[0] != "src/main.go" {
		t.Errorf("expected src/main.go, got %s", files[0])
	}
}

func TestUpdateMessageStatus(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	msg, _ := db.CreateMessage(AuthorHuman, "Comment", "src/main.go", 10, "abc123")

	err := db.UpdateMessageStatus(msg.ID, StatusDelivered)
	if err != nil {
		t.Fatalf("failed to update status: %v", err)
	}

	msgAfter, _ := db.GetMessage(msg.ID)
	if msgAfter.Status != StatusDelivered {
		t.Errorf("expected status to be %s, got %s", StatusDelivered, msgAfter.Status)
	}
}

func TestGetConversationsReturnsOnlyTopLevel(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create three root conversations
	conv1, _ := db.CreateMessage(AuthorHuman, "Conversation 1", "src/main.go", 10, "abc123")
	conv2, _ := db.CreateMessage(AuthorHuman, "Conversation 2", "src/main.go", 20, "abc123")
	conv3, _ := db.CreateMessage(AuthorHuman, "Conversation 3", "src/util.go", 5, "abc123")

	// Create replies to conv1 (should NOT appear in GetConversations)
	db.CreateReply(AuthorAI, "Reply 1 to conv1", conv1.ID)
	db.CreateReply(AuthorHuman, "Reply 2 to conv1", conv1.ID)

	// Create a reply to conv2
	db.CreateReply(AuthorAI, "Reply to conv2", conv2.ID)

	// Get all conversations
	conversations, err := db.GetConversations("")
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
	conv1, _ := db.CreateMessage(AuthorHuman, "Unresolved 1", "src/main.go", 10, "abc123")
	conv2, _ := db.CreateMessage(AuthorHuman, "Unresolved 2", "src/main.go", 20, "abc123")
	conv3, _ := db.CreateMessage(AuthorHuman, "To be resolved", "src/util.go", 5, "abc123")

	// Add replies (should not affect conversation count)
	db.CreateReply(AuthorAI, "Reply to conv1", conv1.ID)
	db.CreateReply(AuthorAI, "Reply to conv3", conv3.ID)

	// Mark conv3 as resolved
	db.MarkAsResolved(conv3.ID)

	// Get unresolved conversations
	unresolved, err := db.GetConversations("unresolved")
	assert.NoError(t, err, "failed to get unresolved conversations")
	assert.Equals(t, len(unresolved), 2, "expected 2 unresolved conversations")

	unresolvedUUIDs := make([]string, len(unresolved))
	for i, conv := range unresolved {
		unresolvedUUIDs[i] = conv.UUID
	}
	assert.Contains(t, unresolvedUUIDs, conv1.ID, "expected conv1 in unresolved conversations")
	assert.Contains(t, unresolvedUUIDs, conv2.ID, "expected conv2 in unresolved conversations")

	// Get resolved conversations
	resolved, err := db.GetConversations("resolved")
	assert.NoError(t, err, "failed to get resolved conversations")
	assert.Equals(t, len(resolved), 1, "expected 1 resolved conversation")
	assert.Equals(t, resolved[0].UUID, conv3.ID, "expected conv3 in resolved conversations")
}

package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/radiospiel/critic/simple-go/assert"
	"github.com/radiospiel/critic/src/messagedb"
	"github.com/radiospiel/critic/src/pkg/critic"
	"github.com/spf13/cobra"
)

// setupTestDB creates a temporary messagedb for testing
func setupTestDB(t *testing.T) (*messagedb.DB, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "critic-agent-test-*")
	assert.NoError(t, err, "failed to create temp dir")

	db, err := messagedb.New(tmpDir)
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

// captureCmd creates a cobra command that captures stdout output
func captureCmd() (*cobra.Command, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(buf)
	return cmd, buf
}

func TestAgentConversations_Empty(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	cmd, buf := captureCmd()

	err := runAgentConversations(cmd, db, "", "")
	assert.NoError(t, err, "expected no error for empty list")
	assert.Equals(t, buf.String(), "[]\n")
}

func TestAgentConversations_ListAll(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create conversations
	conv1, err := db.CreateConversation(critic.AuthorHuman, "Comment 1", "src/main.go", 10, "abc123", "", critic.TypeConversation)
	assert.NoError(t, err)
	conv2, err := db.CreateConversation(critic.AuthorHuman, "Comment 2", "src/util.go", 5, "abc123", "", critic.TypeConversation)
	assert.NoError(t, err)

	// Add AI reply to conv2
	_, err = db.ReplyToConversation(conv2.UUID, "Fixed it", critic.AuthorAI)
	assert.NoError(t, err)

	cmd, buf := captureCmd()

	err = runAgentConversations(cmd, db, "", "")
	assert.NoError(t, err)

	var entries []AgentConversationEntry
	err = json.Unmarshal(buf.Bytes(), &entries)
	assert.NoError(t, err, "failed to parse JSON output")
	assert.Equals(t, len(entries), 2, "expected 2 conversations")

	// Find entries by UUID
	entryMap := make(map[string]AgentConversationEntry)
	for _, e := range entries {
		entryMap[e.UUID] = e
	}

	// conv1 last author should be human (only message)
	assert.Equals(t, entryMap[conv1.UUID].Author, "human")
	assert.Equals(t, entryMap[conv1.UUID].Status, "unresolved")

	// conv2 last author should be ai (has AI reply)
	assert.Equals(t, entryMap[conv2.UUID].Author, "ai")
	assert.Equals(t, entryMap[conv2.UUID].Status, "unresolved")
}

func TestAgentConversations_StatusFilter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	conv1, err := db.CreateConversation(critic.AuthorHuman, "Unresolved", "src/main.go", 10, "abc123", "", critic.TypeConversation)
	assert.NoError(t, err)
	_, err = db.CreateConversation(critic.AuthorHuman, "To resolve", "src/util.go", 5, "abc123", "", critic.TypeConversation)
	assert.NoError(t, err)

	// Resolve conv2 (we need to get the UUID from the return)
	conv2, err := db.CreateConversation(critic.AuthorHuman, "Resolved one", "src/test.go", 1, "abc123", "", critic.TypeConversation)
	assert.NoError(t, err)
	err = db.MarkConversationAs(conv2.UUID, critic.ConversationResolved)
	assert.NoError(t, err)

	// Filter for unresolved only
	cmd, buf := captureCmd()
	err = runAgentConversations(cmd, db, "unresolved", "")
	assert.NoError(t, err)

	var entries []AgentConversationEntry
	err = json.Unmarshal(buf.Bytes(), &entries)
	assert.NoError(t, err)
	assert.Equals(t, len(entries), 2, "expected 2 unresolved conversations")

	// All should be unresolved
	for _, e := range entries {
		assert.Equals(t, e.Status, "unresolved")
	}

	// Filter for resolved only
	cmd2, buf2 := captureCmd()
	err = runAgentConversations(cmd2, db, "resolved", "")
	assert.NoError(t, err)

	var resolvedEntries []AgentConversationEntry
	err = json.Unmarshal(buf2.Bytes(), &resolvedEntries)
	assert.NoError(t, err)
	assert.Equals(t, len(resolvedEntries), 1, "expected 1 resolved conversation")
	assert.Equals(t, resolvedEntries[0].UUID, conv2.UUID)

	// Filter for multiple statuses
	cmd3, buf3 := captureCmd()
	err = runAgentConversations(cmd3, db, "unresolved,resolved", "")
	assert.NoError(t, err)

	var allEntries []AgentConversationEntry
	err = json.Unmarshal(buf3.Bytes(), &allEntries)
	assert.NoError(t, err)
	assert.Equals(t, len(allEntries), 3, "expected 3 conversations with unresolved,resolved filter")

	_ = conv1
}

func TestAgentConversations_LastAuthorFilter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// conv1: human only
	_, err := db.CreateConversation(critic.AuthorHuman, "Human only", "src/main.go", 10, "abc123", "", critic.TypeConversation)
	assert.NoError(t, err)

	// conv2: human start, AI reply → last author = ai
	conv2, err := db.CreateConversation(critic.AuthorHuman, "With AI reply", "src/util.go", 5, "abc123", "", critic.TypeConversation)
	assert.NoError(t, err)
	_, err = db.ReplyToConversation(conv2.UUID, "AI response", critic.AuthorAI)
	assert.NoError(t, err)

	// conv3: human start, AI reply, human follow-up → last author = human
	conv3, err := db.CreateConversation(critic.AuthorHuman, "With follow-up", "src/test.go", 1, "abc123", "", critic.TypeConversation)
	assert.NoError(t, err)
	_, err = db.ReplyToConversation(conv3.UUID, "AI response", critic.AuthorAI)
	assert.NoError(t, err)
	_, err = db.ReplyToConversation(conv3.UUID, "Human follow-up", critic.AuthorHuman)
	assert.NoError(t, err)

	// Filter for last-author=human
	cmd, buf := captureCmd()
	err = runAgentConversations(cmd, db, "", "human")
	assert.NoError(t, err)

	var humanEntries []AgentConversationEntry
	err = json.Unmarshal(buf.Bytes(), &humanEntries)
	assert.NoError(t, err)
	assert.Equals(t, len(humanEntries), 2, "expected 2 conversations with last author=human")

	for _, e := range humanEntries {
		assert.Equals(t, e.Author, "human")
	}

	// Filter for last-author=ai
	cmd2, buf2 := captureCmd()
	err = runAgentConversations(cmd2, db, "", "ai")
	assert.NoError(t, err)

	var aiEntries []AgentConversationEntry
	err = json.Unmarshal(buf2.Bytes(), &aiEntries)
	assert.NoError(t, err)
	assert.Equals(t, len(aiEntries), 1, "expected 1 conversation with last author=ai")
	assert.Equals(t, aiEntries[0].Author, "ai")

	// Filter for last-author=human,ai (should return all)
	cmd3, buf3 := captureCmd()
	err = runAgentConversations(cmd3, db, "", "human,ai")
	assert.NoError(t, err)

	var allEntries []AgentConversationEntry
	err = json.Unmarshal(buf3.Bytes(), &allEntries)
	assert.NoError(t, err)
	assert.Equals(t, len(allEntries), 3, "expected 3 conversations with last-author=human,ai")
}

func TestAgentConversations_CombinedFilters(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// conv1: unresolved, last author = human
	_, err := db.CreateConversation(critic.AuthorHuman, "Comment", "src/main.go", 10, "abc123", "", critic.TypeConversation)
	assert.NoError(t, err)

	// conv2: unresolved, last author = ai
	conv2, err := db.CreateConversation(critic.AuthorHuman, "Comment 2", "src/util.go", 5, "abc123", "", critic.TypeConversation)
	assert.NoError(t, err)
	_, err = db.ReplyToConversation(conv2.UUID, "Fixed", critic.AuthorAI)
	assert.NoError(t, err)

	// conv3: resolved, last author = ai
	conv3, err := db.CreateConversation(critic.AuthorHuman, "Comment 3", "src/test.go", 1, "abc123", "", critic.TypeConversation)
	assert.NoError(t, err)
	_, err = db.ReplyToConversation(conv3.UUID, "Done", critic.AuthorAI)
	assert.NoError(t, err)
	err = db.MarkConversationAs(conv3.UUID, critic.ConversationResolved)
	assert.NoError(t, err)

	// Filter: unresolved + last-author=human
	cmd, buf := captureCmd()
	err = runAgentConversations(cmd, db, "unresolved", "human")
	assert.NoError(t, err)

	var entries []AgentConversationEntry
	err = json.Unmarshal(buf.Bytes(), &entries)
	assert.NoError(t, err)
	assert.Equals(t, len(entries), 1, "expected 1 conversation matching unresolved + last-author=human")
	assert.Equals(t, entries[0].Author, "human")
	assert.Equals(t, entries[0].Status, "unresolved")
}

func TestAgentConversations_ActionableFilter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// conv1: unresolved, last author = human → actionable
	_, err := db.CreateConversation(critic.AuthorHuman, "Please fix", "src/main.go", 10, "abc123", "", critic.TypeConversation)
	assert.NoError(t, err)

	// conv2: unresolved, last author = ai → NOT actionable
	conv2, err := db.CreateConversation(critic.AuthorHuman, "Comment", "src/util.go", 5, "abc123", "", critic.TypeConversation)
	assert.NoError(t, err)
	_, err = db.ReplyToConversation(conv2.UUID, "Fixed", critic.AuthorAI)
	assert.NoError(t, err)

	// conv3: resolved, last author = human → NOT actionable
	conv3, err := db.CreateConversation(critic.AuthorHuman, "Old issue", "src/test.go", 1, "abc123", "", critic.TypeConversation)
	assert.NoError(t, err)
	err = db.MarkConversationAs(conv3.UUID, critic.ConversationResolved)
	assert.NoError(t, err)

	cmd, buf := captureCmd()
	err = runAgentConversations(cmd, db, "actionable", "")
	assert.NoError(t, err)

	var entries []AgentConversationEntry
	err = json.Unmarshal(buf.Bytes(), &entries)
	assert.NoError(t, err)
	assert.Equals(t, len(entries), 1, "expected 1 actionable conversation")
	assert.Equals(t, entries[0].Author, "human")
	assert.Equals(t, entries[0].Status, "unresolved")
}

func TestAgentConversation_Show(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	conv, err := db.CreateConversation(critic.AuthorHuman, "Original comment", "src/main.go", 42, "abc123", "some context", critic.TypeConversation)
	assert.NoError(t, err)
	_, err = db.ReplyToConversation(conv.UUID, "AI response", critic.AuthorAI)
	assert.NoError(t, err)

	cmd, buf := captureCmd()
	err = runAgentConversation(cmd, db, conv.UUID)
	assert.NoError(t, err)

	var response ConversationResponse
	err = json.Unmarshal(buf.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equals(t, response.UUID, conv.UUID)
	assert.Equals(t, response.Status, "unresolved")
	assert.Equals(t, response.FilePath, "src/main.go")
	assert.Equals(t, response.LineNumber, 42)
	assert.Equals(t, len(response.Messages), 2, "expected 2 messages")
	assert.Equals(t, response.Messages[0].Author, "human")
	assert.Equals(t, response.Messages[0].Message, "Original comment")
	assert.Equals(t, response.Messages[1].Author, "ai")
	assert.Equals(t, response.Messages[1].Message, "AI response")
}

func TestAgentConversation_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	cmd, _ := captureCmd()
	err := runAgentConversation(cmd, db, "nonexistent-uuid")
	assert.NotNil(t, err, "expected error for nonexistent conversation")
	assert.Contains(t, err.Error(), "conversation not found")
}

func TestAgentReply(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	conv, err := db.CreateConversation(critic.AuthorHuman, "Please fix this", "src/main.go", 10, "abc123", "", critic.TypeConversation)
	assert.NoError(t, err)

	cmd, buf := captureCmd()
	err = runAgentReply(cmd, db, conv.UUID, "I've fixed the issue")
	assert.NoError(t, err)

	var response ReplyResponse
	err = json.Unmarshal(buf.Bytes(), &response)
	assert.NoError(t, err)

	assert.NotEquals(t, response.UUID, "")
	assert.Equals(t, response.Author, "ai")
	assert.Equals(t, response.Message, "I've fixed the issue")

	// Verify the reply was actually created
	fullConv, err := db.GetFullConversation(conv.UUID)
	assert.NoError(t, err)
	assert.Equals(t, len(fullConv.Messages), 2, "expected 2 messages after reply")
	assert.Equals(t, string(fullConv.Messages[1].Author), "ai")
	assert.Equals(t, fullConv.Messages[1].Message, "I've fixed the issue")
}

func TestAgentAnnounce(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	cmd, buf := captureCmd()
	err := runAgentAnnounce(cmd, db, "Build is broken, do not merge")
	assert.NoError(t, err)

	var response ReplyResponse
	err = json.Unmarshal(buf.Bytes(), &response)
	assert.NoError(t, err)

	assert.NotEquals(t, response.UUID, "")
	assert.Equals(t, response.Author, "ai")
	assert.Equals(t, response.Message, "Build is broken, do not merge")

	// Verify the root conversation is unresolved
	rootConv, err := db.LoadRootConversation()
	assert.NoError(t, err)
	assert.Equals(t, string(rootConv.Status), "unresolved")

	// Verify the announcement message exists in root conversation
	assert.True(t, len(rootConv.Messages) >= 2, "expected at least 2 messages in root conversation (sentinel + announcement)")
}

func TestAgentExplain(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	cmd, buf := captureCmd()
	err := runAgentExplain(cmd, db, "src/main.go", 42, "This handles the auth flow", "abc123")
	assert.NoError(t, err)

	var response ReplyResponse
	err = json.Unmarshal(buf.Bytes(), &response)
	assert.NoError(t, err)

	assert.NotEquals(t, response.UUID, "")
	assert.Equals(t, response.Author, "ai")
	assert.Equals(t, response.Message, "This handles the auth flow")

	// Verify the explanation was created with correct type
	conv, err := db.GetFullConversation(response.UUID)
	assert.NoError(t, err)
	assert.Equals(t, conv.FilePath, "src/main.go")
	assert.Equals(t, conv.LineNumber, 42)
	assert.Equals(t, string(conv.ConversationType), "explanation")
	assert.Equals(t, string(conv.Status), "informal")
}

func TestParseCommaSeparated(t *testing.T) {
	// Empty string returns nil
	result := parseCommaSeparated("")
	assert.Nil(t, result)

	// Single value
	result = parseCommaSeparated("unresolved")
	assert.Equals(t, len(result), 1)
	assert.True(t, result["unresolved"])

	// Multiple values
	result = parseCommaSeparated("unresolved,resolved")
	assert.Equals(t, len(result), 2)
	assert.True(t, result["unresolved"])
	assert.True(t, result["resolved"])

	// With whitespace
	result = parseCommaSeparated(" human , ai ")
	assert.Equals(t, len(result), 2)
	assert.True(t, result["human"])
	assert.True(t, result["ai"])

	// Empty entries are ignored
	result = parseCommaSeparated("human,,ai,")
	assert.Equals(t, len(result), 2)
	assert.True(t, result["human"])
	assert.True(t, result["ai"])
}

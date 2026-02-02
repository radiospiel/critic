package session

import (
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/radiospiel/critic/simple-go/assert"
	"github.com/radiospiel/critic/src/pkg/critic"
	"github.com/radiospiel/critic/src/pkg/types"
)

// createTestSession creates a Session for testing with a temp directory.
// If messaging is nil, it defaults to DummyMessaging.
func createTestSession(t *testing.T, messaging critic.Messaging) *Session {
	if messaging == nil {
		messaging = &critic.DummyMessaging{}
	}
	tempDir := t.TempDir()
	session, err := NewSession(tempDir, messaging, DiffArgs{})
	assert.NoError(t, err, "should create session")
	return session
}

func TestNewSession(t *testing.T) {
	tempDir := t.TempDir()
	session, err := NewSession(tempDir, &critic.DummyMessaging{}, DiffArgs{})
	assert.NoError(t, err, "should create session")
	assert.NotNil(t, session, "session should not be nil")
}

func TestDiff(t *testing.T) {
	session := createTestSession(t, nil)

	// Initially nil
	assert.Nil(t, session.GetDiff(), "initial fileDiffs should be nil")
	assert.Equals(t, session.GetFileCount(), 0, "initial file count should be 0")

	// Set fileDiffs
	diff := []*types.FileDiff{
		{NewPath: "file1.go", OldPath: "file1.go"},
		{NewPath: "file2.go", OldPath: "file2.go"},
	}
	session.SetDiff(diff)

	assert.NotNil(t, session.GetDiff(), "fileDiffs should be set")
	assert.Equals(t, session.GetFileCount(), 2, "file count should be 2")
}

func TestConversations(t *testing.T) {
	messaging := critic.NewDummyMessaging()
	messaging.Conversations["file1.go"] = []*critic.Conversation{
		{UUID: "conv-1", FilePath: "file1.go", LineNumber: 10},
		{UUID: "conv-2", FilePath: "file1.go", LineNumber: 20},
	}
	messaging.Summaries["file1.go"] = &critic.FileConversationSummary{
		FilePath:              "file1.go",
		HasUnresolvedComments: true,
	}

	session := createTestSession(t, messaging)

	// Get conversations from messaging
	convs, err := session.GetConversationsForFile("file1.go")
	assert.NoError(t, err, "should get conversations")
	assert.Equals(t, len(convs), 2, "should have 2 conversations")

	// Get summary from messaging
	summary, err := session.GetConversationSummary("file1.go")
	assert.NoError(t, err, "should get summary")
	assert.NotNil(t, summary, "summary should not be nil")
	assert.True(t, summary.HasUnresolvedComments, "should have unresolved comments")
}

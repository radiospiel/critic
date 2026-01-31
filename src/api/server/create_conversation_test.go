package server

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/pkg/critic"
)

func TestCreateConversation(t *testing.T) {
	// Create a server with a dummy messaging implementation
	messaging := critic.NewDummyMessaging()
	s := &Server{
		config: Config{
			Messaging: messaging,
		},
		session: &Session{}, // Empty session - HeadCommit will return ""
	}

	req := connect.NewRequest(&api.CreateConversationRequest{
		OldFile: "test.go",
		OldLine: 10,
		NewFile: "test.go",
		NewLine: 15,
		Comment: "This is a test comment with **markdown**",
	})

	resp, err := s.CreateConversation(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateConversation failed: %v", err)
	}

	if !resp.Msg.GetSuccess() {
		t.Error("Expected success to be true")
	}

	if resp.Msg.GetError() != nil {
		t.Errorf("Expected no error, got: %v", resp.Msg.GetError())
	}
}

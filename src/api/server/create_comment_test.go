package server

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/radiospiel/critic/src/api"
	"github.com/radiospiel/critic/src/pkg/critic"
)

func TestCreateComment(t *testing.T) {
	// Create a server with a dummy messaging implementation
	messaging := critic.NewDummyMessaging()
	s := &Server{
		config: Config{
			Messaging: messaging,
		},
		session: &Session{}, // Empty session - HeadCommit will return ""
	}

	req := connect.NewRequest(&api.CreateCommentRequest{
		OldFile: "test.go",
		OldLine: 10,
		NewFile: "test.go",
		NewLine: 15,
		Comment: "This is a test comment with **markdown**",
	})

	resp, err := s.CreateComment(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateComment failed: %v", err)
	}

	if !resp.Msg.GetSuccess() {
		t.Error("Expected success to be true")
	}

	if resp.Msg.GetError() != nil {
		t.Errorf("Expected no error, got: %v", resp.Msg.GetError())
	}
}

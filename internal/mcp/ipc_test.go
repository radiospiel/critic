package mcp

import (
	"encoding/json"
	"net"
	"testing"
	"time"
)

func TestReviewerIPCStartStop(t *testing.T) {
	ipc := NewReviewerIPC("/tmp/test-ipc-" + time.Now().Format("20060102150405.000") + ".sock")

	// Start should succeed
	err := ipc.Start()
	if err != nil {
		t.Fatalf("Failed to start IPC: %v", err)
	}

	// Starting again should fail
	err = ipc.Start()
	if err == nil {
		t.Error("Expected error when starting twice")
	}

	// Stop should succeed
	err = ipc.Stop()
	if err != nil {
		t.Fatalf("Failed to stop IPC: %v", err)
	}
}

func TestReviewerIPCFeedback(t *testing.T) {
	socketPath := "/tmp/test-ipc-feedback-" + time.Now().Format("20060102150405.000") + ".sock"
	ipc := NewReviewerIPC(socketPath)

	err := ipc.Start()
	if err != nil {
		t.Fatalf("Failed to start IPC: %v", err)
	}
	defer ipc.Stop()

	// Send feedback in a goroutine
	go func() {
		time.Sleep(50 * time.Millisecond) // Give time for WaitForFeedback to start

		conn, err := net.Dial("unix", socketPath)
		if err != nil {
			t.Logf("Failed to connect: %v", err)
			return
		}
		defer conn.Close()

		msg := ReviewerMessage{
			Type:     "feedback",
			Feedback: "Test feedback message",
		}
		data, _ := json.Marshal(msg)
		conn.Write(append(data, '\n'))
	}()

	// Wait for feedback
	msg, err := ipc.WaitForFeedback(1 * time.Second)
	if err != nil {
		t.Fatalf("Failed to get feedback: %v", err)
	}

	if msg.Type != "feedback" {
		t.Errorf("Expected type 'feedback', got '%s'", msg.Type)
	}

	if msg.Feedback != "Test feedback message" {
		t.Errorf("Expected feedback 'Test feedback message', got '%s'", msg.Feedback)
	}
}

func TestReviewerIPCPlainText(t *testing.T) {
	socketPath := "/tmp/test-ipc-plain-" + time.Now().Format("20060102150405.000") + ".sock"
	ipc := NewReviewerIPC(socketPath)

	err := ipc.Start()
	if err != nil {
		t.Fatalf("Failed to start IPC: %v", err)
	}
	defer ipc.Stop()

	// Send plain text feedback
	go func() {
		time.Sleep(50 * time.Millisecond)

		conn, err := net.Dial("unix", socketPath)
		if err != nil {
			t.Logf("Failed to connect: %v", err)
			return
		}
		defer conn.Close()

		conn.Write([]byte("Plain text feedback\n"))
	}()

	msg, err := ipc.WaitForFeedback(1 * time.Second)
	if err != nil {
		t.Fatalf("Failed to get feedback: %v", err)
	}

	if msg.Type != "feedback" {
		t.Errorf("Expected type 'feedback', got '%s'", msg.Type)
	}

	if msg.Feedback != "Plain text feedback" {
		t.Errorf("Expected feedback 'Plain text feedback', got '%s'", msg.Feedback)
	}
}

func TestReviewerIPCTimeout(t *testing.T) {
	socketPath := "/tmp/test-ipc-timeout-" + time.Now().Format("20060102150405.000") + ".sock"
	ipc := NewReviewerIPC(socketPath)

	err := ipc.Start()
	if err != nil {
		t.Fatalf("Failed to start IPC: %v", err)
	}
	defer ipc.Stop()

	// Don't send any feedback - should timeout
	start := time.Now()
	_, err = ipc.WaitForFeedback(100 * time.Millisecond)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Expected timeout error")
	}

	if elapsed < 100*time.Millisecond {
		t.Errorf("Timeout happened too quickly: %v", elapsed)
	}

	if elapsed > 200*time.Millisecond {
		t.Errorf("Timeout took too long: %v", elapsed)
	}
}

func TestReviewerIPCNotify(t *testing.T) {
	socketPath := "/tmp/test-ipc-notify-" + time.Now().Format("20060102150405.000") + ".sock"
	ipc := NewReviewerIPC(socketPath)

	err := ipc.Start()
	if err != nil {
		t.Fatalf("Failed to start IPC: %v", err)
	}
	defer ipc.Stop()

	// Connect a reviewer
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Give time for connection to be accepted
	time.Sleep(50 * time.Millisecond)

	// Send notification
	err = ipc.NotifyReviewer(NotificationMessage{
		Type:    "waiting",
		Summary: "Test summary",
	})
	if err != nil {
		t.Errorf("Failed to notify: %v", err)
	}

	// Read notification from connection
	buf := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read notification: %v", err)
	}

	var notification NotificationMessage
	if err := json.Unmarshal(buf[:n-1], &notification); err != nil { // -1 for newline
		t.Fatalf("Failed to unmarshal notification: %v", err)
	}

	if notification.Type != "waiting" {
		t.Errorf("Expected type 'waiting', got '%s'", notification.Type)
	}

	if notification.Summary != "Test summary" {
		t.Errorf("Expected summary 'Test summary', got '%s'", notification.Summary)
	}
}

func TestReviewerIPCApprovalRejection(t *testing.T) {
	tests := []struct {
		msgType  string
		feedback string
	}{
		{"approved", "LGTM"},
		{"rejected", "Needs work"},
	}

	for _, tt := range tests {
		t.Run(tt.msgType, func(t *testing.T) {
			socketPath := "/tmp/test-ipc-" + tt.msgType + "-" + time.Now().Format("20060102150405.000") + ".sock"
			ipc := NewReviewerIPC(socketPath)

			err := ipc.Start()
			if err != nil {
				t.Fatalf("Failed to start IPC: %v", err)
			}
			defer ipc.Stop()

			go func() {
				time.Sleep(50 * time.Millisecond)

				conn, err := net.Dial("unix", socketPath)
				if err != nil {
					return
				}
				defer conn.Close()

				msg := ReviewerMessage{
					Type:     tt.msgType,
					Feedback: tt.feedback,
				}
				data, _ := json.Marshal(msg)
				conn.Write(append(data, '\n'))
			}()

			msg, err := ipc.WaitForFeedback(1 * time.Second)
			if err != nil {
				t.Fatalf("Failed to get feedback: %v", err)
			}

			if msg.Type != tt.msgType {
				t.Errorf("Expected type '%s', got '%s'", tt.msgType, msg.Type)
			}

			if msg.Feedback != tt.feedback {
				t.Errorf("Expected feedback '%s', got '%s'", tt.feedback, msg.Feedback)
			}
		})
	}
}

func TestGetSocketPath(t *testing.T) {
	customPath := "/custom/path.sock"
	ipc := NewReviewerIPC(customPath)

	if ipc.GetSocketPath() != customPath {
		t.Errorf("Expected socket path '%s', got '%s'", customPath, ipc.GetSocketPath())
	}

	defaultIPC := NewReviewerIPC("")
	if defaultIPC.GetSocketPath() != DefaultSocketPath {
		t.Errorf("Expected default socket path '%s', got '%s'", DefaultSocketPath, defaultIPC.GetSocketPath())
	}
}

package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

const (
	// DefaultSocketPath is the default Unix socket path for HITL communication
	DefaultSocketPath = "/tmp/critic-hitl.sock"
)

// ReviewerMessage represents a message from the reviewer
type ReviewerMessage struct {
	Type     string `json:"type"`               // "feedback", "approved", "rejected"
	Feedback string `json:"feedback,omitempty"` // The feedback text
}

// NotificationMessage represents a notification to the reviewer
type NotificationMessage struct {
	Type    string `json:"type"`    // "waiting", "notification"
	Summary string `json:"summary"` // Brief description of what Claude has done
}

// ReviewerIPC handles communication with the external reviewer process
type ReviewerIPC struct {
	socketPath     string
	listener       net.Listener
	mu             sync.Mutex
	pendingFeedback []ReviewerMessage
	feedbackChan   chan ReviewerMessage
	connections    []net.Conn
	running        bool
	stopChan       chan struct{}
}

// NewReviewerIPC creates a new reviewer IPC handler
func NewReviewerIPC(socketPath string) *ReviewerIPC {
	if socketPath == "" {
		socketPath = DefaultSocketPath
	}
	return &ReviewerIPC{
		socketPath:     socketPath,
		feedbackChan:   make(chan ReviewerMessage, 10),
		stopChan:       make(chan struct{}),
	}
}

// Start starts the IPC server
func (r *ReviewerIPC) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		return fmt.Errorf("IPC server already running")
	}

	// Remove existing socket file if present
	if err := os.Remove(r.socketPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing socket: %w", err)
	}

	// Create Unix socket listener
	listener, err := net.Listen("unix", r.socketPath)
	if err != nil {
		return fmt.Errorf("failed to create socket: %w", err)
	}

	r.listener = listener
	r.running = true

	// Start accepting connections
	go r.acceptLoop()

	return nil
}

// Stop stops the IPC server
func (r *ReviewerIPC) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.running {
		return nil
	}

	r.running = false
	close(r.stopChan)

	// Close all connections
	for _, conn := range r.connections {
		conn.Close()
	}
	r.connections = nil

	// Close listener
	if r.listener != nil {
		r.listener.Close()
	}

	// Remove socket file
	os.Remove(r.socketPath)

	return nil
}

// acceptLoop accepts incoming connections
func (r *ReviewerIPC) acceptLoop() {
	for {
		conn, err := r.listener.Accept()
		if err != nil {
			select {
			case <-r.stopChan:
				return
			default:
				// Log error but continue accepting
				fmt.Fprintf(os.Stderr, "[HITL] Accept error: %v\n", err)
				continue
			}
		}

		r.mu.Lock()
		r.connections = append(r.connections, conn)
		r.mu.Unlock()

		go r.handleConnection(conn)
	}
}

// handleConnection handles a single reviewer connection
func (r *ReviewerIPC) handleConnection(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Try to parse as JSON
		var msg ReviewerMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			// Treat as plain text feedback
			msg = ReviewerMessage{
				Type:     "feedback",
				Feedback: line,
			}
		}

		// Send to feedback channel
		select {
		case r.feedbackChan <- msg:
		case <-r.stopChan:
			return
		default:
			// Channel full, add to pending
			r.mu.Lock()
			r.pendingFeedback = append(r.pendingFeedback, msg)
			r.mu.Unlock()
		}
	}

	// Remove connection from list
	r.mu.Lock()
	for i, c := range r.connections {
		if c == conn {
			r.connections = append(r.connections[:i], r.connections[i+1:]...)
			break
		}
	}
	r.mu.Unlock()
}

// WaitForFeedback waits for feedback from the reviewer with a timeout
func (r *ReviewerIPC) WaitForFeedback(timeout time.Duration) (*ReviewerMessage, error) {
	// First check if there's pending feedback
	r.mu.Lock()
	if len(r.pendingFeedback) > 0 {
		msg := r.pendingFeedback[0]
		r.pendingFeedback = r.pendingFeedback[1:]
		r.mu.Unlock()
		return &msg, nil
	}
	r.mu.Unlock()

	// Wait for feedback with timeout
	select {
	case msg := <-r.feedbackChan:
		return &msg, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for reviewer feedback")
	case <-r.stopChan:
		return nil, fmt.Errorf("IPC server stopped")
	}
}

// NotifyReviewer sends a notification to connected reviewers
func (r *ReviewerIPC) NotifyReviewer(notification NotificationMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	// Send to all connected reviewers
	for _, conn := range r.connections {
		conn.Write(append(data, '\n'))
	}

	return nil
}

// GetSocketPath returns the socket path
func (r *ReviewerIPC) GetSocketPath() string {
	return r.socketPath
}

package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"
)

// testServer creates a server with custom io for testing
type testServer struct {
	*Server
	input  *bytes.Buffer
	output *bytes.Buffer
}

func newTestServer() *testServer {
	input := &bytes.Buffer{}
	output := &bytes.Buffer{}

	s := &Server{
		reader:          bufio.NewReader(input),
		writer:          output,
		reviewerIPC:     NewReviewerIPC("/tmp/test-hitl-" + time.Now().Format("20060102150405") + ".sock"),
		feedbackTimeout: 100 * time.Millisecond,
	}

	return &testServer{
		Server: s,
		input:  input,
		output: output,
	}
}

func (ts *testServer) sendRequest(method string, params interface{}, id interface{}) error {
	req := Request{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	ts.input.Write(append(data, '\n'))
	return nil
}

func (ts *testServer) readResponse() (*Response, error) {
	line, err := bufio.NewReader(ts.output).ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	var resp Response
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func TestServerInitialize(t *testing.T) {
	ts := newTestServer()

	// Send initialize request
	err := ts.sendRequest("initialize", InitializeParams{
		ProtocolVersion: "2024-11-05",
		ClientInfo:      ClientInfo{Name: "test", Version: "1.0"},
	}, 1)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Process the request
	line, _ := ts.reader.ReadBytes('\n')
	err = ts.handleMessage(line)
	if err != nil {
		t.Fatalf("Failed to handle message: %v", err)
	}

	// Read response
	resp, err := ts.readResponse()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("Unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", resp.Result)
	}

	if result["protocolVersion"] != ProtocolVersion {
		t.Errorf("Expected protocol version %s, got %v", ProtocolVersion, result["protocolVersion"])
	}
}

func TestServerToolsList(t *testing.T) {
	ts := newTestServer()

	err := ts.sendRequest("tools/list", nil, 1)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	line, _ := ts.reader.ReadBytes('\n')
	err = ts.handleMessage(line)
	if err != nil {
		t.Fatalf("Failed to handle message: %v", err)
	}

	resp, err := ts.readResponse()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("Unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", resp.Result)
	}

	tools, ok := result["tools"].([]interface{})
	if !ok {
		t.Fatalf("Expected tools array, got %T", result["tools"])
	}

	if len(tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(tools))
	}

	// Check tool names
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolMap := tool.(map[string]interface{})
		toolNames[toolMap["name"].(string)] = true
	}

	if !toolNames["get_review_feedback"] {
		t.Error("Missing get_review_feedback tool")
	}
	if !toolNames["notify_reviewer"] {
		t.Error("Missing notify_reviewer tool")
	}
}

func TestServerNotifyReviewer(t *testing.T) {
	ts := newTestServer()

	// Start IPC server
	if err := ts.reviewerIPC.Start(); err != nil {
		t.Fatalf("Failed to start IPC: %v", err)
	}
	defer ts.reviewerIPC.Stop()

	params := CallToolParams{
		Name: "notify_reviewer",
		Arguments: map[string]interface{}{
			"message": "Test notification",
		},
	}

	err := ts.sendRequest("tools/call", params, 1)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	line, _ := ts.reader.ReadBytes('\n')
	err = ts.handleMessage(line)
	if err != nil {
		t.Fatalf("Failed to handle message: %v", err)
	}

	resp, err := ts.readResponse()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("Unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", resp.Result)
	}

	content, ok := result["content"].([]interface{})
	if !ok || len(content) == 0 {
		t.Fatalf("Expected content array, got %v", result["content"])
	}

	firstContent := content[0].(map[string]interface{})
	if !strings.Contains(firstContent["text"].(string), "notified") {
		t.Errorf("Expected 'notified' in response, got %s", firstContent["text"])
	}
}

func TestServerUnknownMethod(t *testing.T) {
	ts := newTestServer()

	err := ts.sendRequest("unknown/method", nil, 1)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	line, _ := ts.reader.ReadBytes('\n')
	err = ts.handleMessage(line)
	if err != nil {
		t.Fatalf("Failed to handle message: %v", err)
	}

	resp, err := ts.readResponse()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if resp.Error == nil {
		t.Error("Expected error for unknown method")
	}

	if resp.Error.Code != MethodNotFound {
		t.Errorf("Expected MethodNotFound error code, got %d", resp.Error.Code)
	}
}

func TestServerPing(t *testing.T) {
	ts := newTestServer()

	err := ts.sendRequest("ping", nil, 1)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	line, _ := ts.reader.ReadBytes('\n')
	err = ts.handleMessage(line)
	if err != nil {
		t.Fatalf("Failed to handle message: %v", err)
	}

	resp, err := ts.readResponse()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("Unexpected error: %v", resp.Error)
	}
}

func TestInvalidJSON(t *testing.T) {
	ts := newTestServer()

	ts.input.WriteString("not valid json\n")

	line, _ := ts.reader.ReadBytes('\n')
	err := ts.handleMessage(line)
	if err != nil {
		// Error is sent as response, not returned
		t.Logf("handleMessage returned error: %v", err)
	}

	resp, err := ts.readResponse()
	if err != nil && err != io.EOF {
		t.Fatalf("Failed to read response: %v", err)
	}

	if resp != nil && resp.Error == nil {
		t.Error("Expected error for invalid JSON")
	}
}

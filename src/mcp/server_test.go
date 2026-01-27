package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"testing"

	"github.org/radiospiel/critic/simple-go/assert"
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
		reader: bufio.NewReader(input),
		writer: output,
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
	assert.NoError(t, err, "Failed to send request")

	// Process the request
	line, _ := ts.reader.ReadBytes('\n')
	err = ts.handleMessage(line)
	assert.NoError(t, err, "Failed to handle message")

	// Read response
	resp, err := ts.readResponse()
	assert.NoError(t, err, "Failed to read response")
	assert.Nil(t, resp.Error, "Unexpected error: %v", resp.Error)

	result, ok := resp.Result.(map[string]interface{})
	assert.True(t, ok, "Expected map result, got %T", resp.Result)
	assert.Equals(t, result["protocolVersion"], ProtocolVersion)
}

func TestServerToolsList(t *testing.T) {
	ts := newTestServer()

	err := ts.sendRequest("tools/list", nil, 1)
	assert.NoError(t, err, "Failed to send request")

	line, _ := ts.reader.ReadBytes('\n')
	err = ts.handleMessage(line)
	assert.NoError(t, err, "Failed to handle message")

	resp, err := ts.readResponse()
	assert.NoError(t, err, "Failed to read response")
	assert.Nil(t, resp.Error, "Unexpected error: %v", resp.Error)

	result, ok := resp.Result.(map[string]interface{})
	assert.True(t, ok, "Expected map result, got %T", resp.Result)

	tools, ok := result["tools"].([]interface{})
	assert.True(t, ok, "Expected tools array, got %T", result["tools"])
	assert.Equals(t, len(tools), 3, "Expected 3 tools")

	// Check tool names
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolMap := tool.(map[string]interface{})
		toolNames[toolMap["name"].(string)] = true
	}

	expectedTools := []string{
		"get_critic_conversations",
		"get_full_critic_conversation",
		"reply_to_critic_conversation",
	}

	for _, name := range expectedTools {
		assert.True(t, toolNames[name], "Missing %s tool", name)
	}
}

func TestServerUnknownMethod(t *testing.T) {
	ts := newTestServer()

	err := ts.sendRequest("unknown/method", nil, 1)
	assert.NoError(t, err, "Failed to send request")

	line, _ := ts.reader.ReadBytes('\n')
	err = ts.handleMessage(line)
	assert.NoError(t, err, "Failed to handle message")

	resp, err := ts.readResponse()
	assert.NoError(t, err, "Failed to read response")
	assert.NotNil(t, resp.Error, "Expected error for unknown method")
	assert.Equals(t, resp.Error.Code, MethodNotFound, "Expected MethodNotFound error code")
}

func TestServerPing(t *testing.T) {
	ts := newTestServer()

	err := ts.sendRequest("ping", nil, 1)
	assert.NoError(t, err, "Failed to send request")

	line, _ := ts.reader.ReadBytes('\n')
	err = ts.handleMessage(line)
	assert.NoError(t, err, "Failed to handle message")

	resp, err := ts.readResponse()
	assert.NoError(t, err, "Failed to read response")
	assert.Nil(t, resp.Error, "Unexpected error: %v", resp.Error)
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

	if resp != nil {
		assert.NotNil(t, resp.Error, "Expected error for invalid JSON")
	}
}

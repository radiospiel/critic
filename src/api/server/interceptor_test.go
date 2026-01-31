package server

import (
	"testing"

	"github.com/radiospiel/critic/simple-go/assert"
	"github.com/radiospiel/critic/src/api"
)

func TestToJSON(t *testing.T) {
	// Test with nil
	result := toJSON(nil)
	assert.Equals(t, result, "null", "nil should be 'null'")

	// Test with simple struct
	data := struct {
		Name string `json:"name"`
	}{Name: "test"}
	result = toJSON(data)
	assert.Contains(t, result, `"name":"test"`, "should contain JSON field")

	// Test with map
	m := map[string]any{"key": "value"}
	result = toJSON(m)
	assert.Contains(t, result, `"key":"value"`, "should contain JSON field")

	// Test with protobuf message (uses protojson)
	req := &api.GetDiffRequest{Path: "src/main.go"}
	result = toJSON(req)
	assert.Contains(t, result, `"path":`, "protobuf message should contain path field")
	assert.Contains(t, result, `src/main.go`, "protobuf message should contain path value")
}

func TestToJSON_Truncation(t *testing.T) {
	// Create a large protobuf response that exceeds maxLogLength (200)
	files := make([]*api.FileSummary, 20)
	for i := range files {
		files[i] = &api.FileSummary{
			OldPath: "very/long/path/to/file/number" + string(rune('A'+i)) + ".go",
			NewPath: "very/long/path/to/file/number" + string(rune('A'+i)) + ".go",
			Status:  api.FileStatus_FILE_STATUS_MODIFIED,
		}
	}
	resp := &api.GetDiffSummaryResponse{
		State: "clean",
		Diff:  &api.DiffSummary{Files: files},
	}

	result := toJSON(resp)
	assert.True(t, len(result) <= maxLogLength+3, "should truncate to maxLogLength+3 (for '...')")
	assert.Contains(t, result, "...", "truncated output should end with ...")
}

func TestTruncate(t *testing.T) {
	// Short string - no truncation
	short := "hello"
	assert.Equals(t, truncate(short, 10), "hello", "short strings should not be truncated")

	// Exact length - no truncation
	exact := "1234567890"
	assert.Equals(t, truncate(exact, 10), "1234567890", "exact length should not be truncated")

	// Long string - truncated
	long := "12345678901234567890"
	result := truncate(long, 10)
	assert.Equals(t, result, "1234567890...", "long strings should be truncated with ...")
	assert.Equals(t, len(result), 13, "truncated length should be maxLen + 3")
}

func TestValidateRequest_GetDiff_MissingPath(t *testing.T) {
	procedure := "/critic.v1.CriticService/GetDiff"
	req := &api.GetDiffRequest{}

	err := validateRequest(procedure, req)
	assert.NotNil(t, err, "should return error for missing path")
	assert.Contains(t, err.Error(), "path", "should mention path field")
}

func TestValidateRequest_GetDiff_EmptyPath(t *testing.T) {
	procedure := "/critic.v1.CriticService/GetDiff"
	req := &api.GetDiffRequest{Path: ""}

	err := validateRequest(procedure, req)
	assert.NotNil(t, err, "should return error for empty path")
}

func TestValidateRequest_GetDiff_ValidPath(t *testing.T) {
	procedure := "/critic.v1.CriticService/GetDiff"
	req := &api.GetDiffRequest{Path: "src/main.go"}

	err := validateRequest(procedure, req)
	assert.Nil(t, err, "should not return error for valid request")
}

func TestValidateRequest_GetLastChange(t *testing.T) {
	procedure := "/critic.v1.CriticService/GetLastChange"
	req := &api.GetLastChangeRequest{}

	err := validateRequest(procedure, req)
	assert.Nil(t, err, "should not return error for GetLastChange")
}

func TestValidateRequest_GetDiffSummary(t *testing.T) {
	procedure := "/critic.v1.CriticService/GetDiffSummary"
	req := &api.GetDiffSummaryRequest{}

	err := validateRequest(procedure, req)
	assert.Nil(t, err, "should not return error for GetDiffSummary")
}

func TestValidateRequest_UnknownProcedure(t *testing.T) {
	procedure := "/unknown/procedure"
	req := struct{ Foo string }{Foo: "bar"}

	err := validateRequest(procedure, req)
	assert.Nil(t, err, "should not return error for unknown procedure")
}

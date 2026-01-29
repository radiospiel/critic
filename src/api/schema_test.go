package api

import (
	"testing"

	"github.com/radiospiel/critic/simple-go/assert"
)

func TestValidateRequest_GetLastChange(t *testing.T) {
	procedure := "/critic.v1.CriticService/GetLastChange"

	// Empty request should be valid (no required fields)
	err := ValidateRequest(procedure, map[string]any{})
	assert.NoError(t, err, "empty GetLastChange request should be valid")
}

func TestValidateRequest_GetDiffSummary(t *testing.T) {
	procedure := "/critic.v1.CriticService/GetDiffSummary"

	// Empty request should be valid (no required fields)
	err := ValidateRequest(procedure, map[string]any{})
	assert.NoError(t, err, "empty GetDiffSummary request should be valid")
}

func TestValidateRequest_GetDiff(t *testing.T) {
	procedure := "/critic.v1.CriticService/GetDiff"

	// Missing required field
	err := ValidateRequest(procedure, map[string]any{})
	assert.Error(t, err, "path", "missing path should produce an error")
	assert.Contains(t, FormatValidationError(err), "path", "error should mention path field")

	// Empty path (violates minLength)
	err = ValidateRequest(procedure, map[string]any{"path": ""})
	assert.Error(t, err, "minLength", "empty path should produce an error")
	assert.Contains(t, FormatValidationError(err), "path", "error should mention path field")

	// Valid request
	err = ValidateRequest(procedure, map[string]any{"path": "src/main.go"})
	assert.NoError(t, err, "valid GetDiff request should have no errors")
}

func TestValidateRequest_UnknownProcedure(t *testing.T) {
	// Unknown procedure should pass (no schema defined)
	err := ValidateRequest("/unknown/procedure", map[string]any{"foo": "bar"})
	assert.NoError(t, err, "unknown procedure should pass validation")
}

func TestProtoToJSON(t *testing.T) {
	// Test with a simple struct
	msg := struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}{Name: "test", Value: 42}

	json, err := ProtoToJSON(msg)
	assert.NoError(t, err)
	assert.Contains(t, json, `"name":"test"`, "JSON should contain name field")
	assert.Contains(t, json, `"value":42`, "JSON should contain value field")
}

func TestProtoToMap(t *testing.T) {
	// Test with a simple struct
	msg := struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}{Name: "test", Value: 42}

	m, err := ProtoToMap(msg)
	assert.NoError(t, err)
	assert.Equals(t, m["name"], "test", "map should contain name field")
	assert.Equals(t, m["value"], float64(42), "map should contain value field as float64")
}

func TestFormatValidationError(t *testing.T) {
	// nil error should return empty string
	assert.Equals(t, FormatValidationError(nil), "", "nil error should return empty string")

	// Actual validation error
	err := ValidateRequest("/critic.v1.CriticService/GetDiff", map[string]any{})
	formatted := FormatValidationError(err)
	assert.Contains(t, formatted, "path", "formatted error should mention the field")
}

func TestNewRpcError(t *testing.T) {
	err := NewRpcError(ErrorCode_ERROR_CODE_INVALID_ARGUMENT, "test message")

	assert.NotNil(t, err, "error should not be nil")
	assert.Equals(t, err.Code, ErrorCode_ERROR_CODE_INVALID_ARGUMENT, "code should match")
	assert.Equals(t, err.Message, "test message", "message should match")
}

func TestInvalidArgument(t *testing.T) {
	err := InvalidArgument("bad input")

	assert.Equals(t, err.Code, ErrorCode_ERROR_CODE_INVALID_ARGUMENT, "code should be INVALID_ARGUMENT")
	assert.Equals(t, err.Message, "bad input", "message should match")
}

func TestNotFound(t *testing.T) {
	err := NotFound("resource not found")

	assert.Equals(t, err.Code, ErrorCode_ERROR_CODE_NOT_FOUND, "code should be NOT_FOUND")
	assert.Equals(t, err.Message, "resource not found", "message should match")
}

func TestInternalError(t *testing.T) {
	err := InternalError("unexpected error")

	assert.Equals(t, err.Code, ErrorCode_ERROR_CODE_INTERNAL, "code should be INTERNAL")
	assert.Equals(t, err.Message, "unexpected error", "message should match")
}

func TestUnavailable(t *testing.T) {
	err := Unavailable("service unavailable")

	assert.Equals(t, err.Code, ErrorCode_ERROR_CODE_UNAVAILABLE, "code should be UNAVAILABLE")
	assert.Equals(t, err.Message, "service unavailable", "message should match")
}

func TestRpcErrorMessage(t *testing.T) {
	// Without details
	err := NewRpcError(ErrorCode_ERROR_CODE_INTERNAL, "server error")
	assert.Contains(t, RpcErrorMessage(err), "server error", "message should contain error text")

	// With details
	err = NewRpcError(ErrorCode_ERROR_CODE_INVALID_ARGUMENT, "validation failed")
	err.Details = "field 'name' is required"
	msg := RpcErrorMessage(err)
	assert.Contains(t, msg, "validation failed", "message should contain error text")
	assert.Contains(t, msg, "field 'name' is required", "message should contain details")
}

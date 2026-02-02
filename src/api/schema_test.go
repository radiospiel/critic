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
	procedure := "/critic.v1.CriticService/GetFileDiffs"

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
	assert.NoError(t, err, "valid GetFileDiffs request should have no errors")
}

func TestValidateRequest_UnknownProcedure(t *testing.T) {
	// Unknown procedure should pass (no schema defined)
	err := ValidateRequest("/unknown/procedure", map[string]any{"foo": "bar"})
	assert.NoError(t, err, "unknown procedure should pass validation")
}

func TestFormatValidationError(t *testing.T) {
	// nil error should return empty string
	assert.Equals(t, FormatValidationError(nil), "", "nil error should return empty string")

	// Actual validation error
	err := ValidateRequest("/critic.v1.CriticService/GetFileDiffs", map[string]any{})
	formatted := FormatValidationError(err)
	assert.Contains(t, formatted, "path", "formatted error should mention the field")
}

func TestInvalidArgumentError(t *testing.T) {
	err := InvalidArgumentError("bad input")
	rpcErr := err.(*RpcErr).RpcError()

	assert.Equals(t, rpcErr.Code, ErrorCode_ERROR_CODE_INVALID_ARGUMENT, "code should be INVALID_ARGUMENT")
	assert.Equals(t, rpcErr.Message, "bad input", "message should match")
}

func TestNotFoundError(t *testing.T) {
	err := NotFoundError("resource not found")
	rpcErr := err.(*RpcErr).RpcError()

	assert.Equals(t, rpcErr.Code, ErrorCode_ERROR_CODE_NOT_FOUND, "code should be NOT_FOUND")
	assert.Equals(t, rpcErr.Message, "resource not found", "message should match")
}

func TestInternalServerError(t *testing.T) {
	err := InternalServerError("unexpected error")
	rpcErr := err.(*RpcErr).RpcError()

	assert.Equals(t, rpcErr.Code, ErrorCode_ERROR_CODE_INTERNAL, "code should be INTERNAL")
	assert.Equals(t, rpcErr.Message, "unexpected error", "message should match")
}

func TestUnavailableError(t *testing.T) {
	err := UnavailableError("service unavailable")
	rpcErr := err.(*RpcErr).RpcError()

	assert.Equals(t, rpcErr.Code, ErrorCode_ERROR_CODE_UNAVAILABLE, "code should be UNAVAILABLE")
	assert.Equals(t, rpcErr.Message, "service unavailable", "message should match")
}

func TestRpcErrorMessage(t *testing.T) {
	// Without details
	err := InternalServerError("server error")
	rpcErr := err.(*RpcErr).RpcError()
	assert.Contains(t, RpcErrorMessage(rpcErr), "server error", "message should contain error text")

	// With details
	err = InvalidArgumentError("validation failed", "field 'name' is required")
	rpcErr = err.(*RpcErr).RpcError()
	msg := RpcErrorMessage(rpcErr)
	assert.Contains(t, msg, "validation failed", "message should contain error text")
	assert.Contains(t, msg, "field 'name' is required", "message should contain details")
}

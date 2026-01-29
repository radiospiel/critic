package api

import (
	"testing"

	"github.com/radiospiel/critic/simple-go/assert"
)

func TestErrorCode_String(t *testing.T) {
	assert.Equals(t, ErrorCodeUnspecified.String(), "UNSPECIFIED", "UNSPECIFIED code")
	assert.Equals(t, ErrorCodeInvalidArgument.String(), "INVALID_ARGUMENT", "INVALID_ARGUMENT code")
	assert.Equals(t, ErrorCodeNotFound.String(), "NOT_FOUND", "NOT_FOUND code")
	assert.Equals(t, ErrorCodeInternal.String(), "INTERNAL", "INTERNAL code")
	assert.Equals(t, ErrorCodeUnavailable.String(), "UNAVAILABLE", "UNAVAILABLE code")

	// Unknown code
	unknownCode := ErrorCode(999)
	assert.Equals(t, unknownCode.String(), "UNKNOWN", "unknown code should return UNKNOWN")
}

func TestNewRpcError(t *testing.T) {
	err := NewRpcError(ErrorCodeInvalidArgument, "test message")

	assert.NotNil(t, err, "error should not be nil")
	assert.Equals(t, err.Code, ErrorCodeInvalidArgument, "code should match")
	assert.Equals(t, err.Message, "test message", "message should match")
	assert.Equals(t, err.Details, "", "details should be empty")
}

func TestRpcError_WithDetails(t *testing.T) {
	err := NewRpcError(ErrorCodeNotFound, "not found").WithDetails("resource xyz")

	assert.Equals(t, err.Details, "resource xyz", "details should be set")
}

func TestRpcError_Error(t *testing.T) {
	// Without details
	err := NewRpcError(ErrorCodeInternal, "server error")
	assert.Equals(t, err.Error(), "INTERNAL: server error", "Error() without details")

	// With details
	err = NewRpcError(ErrorCodeInvalidArgument, "validation failed").WithDetails("field 'name' is required")
	assert.Equals(t, err.Error(), "INVALID_ARGUMENT: validation failed - field 'name' is required", "Error() with details")
}

func TestInvalidArgument(t *testing.T) {
	err := InvalidArgument("bad input")

	assert.Equals(t, err.Code, ErrorCodeInvalidArgument, "code should be INVALID_ARGUMENT")
	assert.Equals(t, err.Message, "bad input", "message should match")
}

func TestNotFound(t *testing.T) {
	err := NotFound("resource not found")

	assert.Equals(t, err.Code, ErrorCodeNotFound, "code should be NOT_FOUND")
	assert.Equals(t, err.Message, "resource not found", "message should match")
}

func TestInternalError(t *testing.T) {
	err := InternalError("unexpected error")

	assert.Equals(t, err.Code, ErrorCodeInternal, "code should be INTERNAL")
	assert.Equals(t, err.Message, "unexpected error", "message should match")
}

func TestUnavailable(t *testing.T) {
	err := Unavailable("service unavailable")

	assert.Equals(t, err.Code, ErrorCodeUnavailable, "code should be UNAVAILABLE")
	assert.Equals(t, err.Message, "service unavailable", "message should match")
}

func TestRpcError_ImplementsErrorInterface(t *testing.T) {
	var _ error = (*RpcError)(nil)
}

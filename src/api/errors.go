package api

import "fmt"

// RpcErr is a Go error that wraps an RpcError.
// It can be returned from impl functions and will be converted to an RpcError
// by rpcErrorFromGoError in the server package.
type RpcErr struct {
	rpcError *RpcError
}

// Error implements the error interface.
func (e *RpcErr) Error() string {
	return RpcErrorMessage(e.rpcError)
}

// RpcError returns the underlying RpcError.
func (e *RpcErr) RpcError() *RpcError {
	return e.rpcError
}

// newRpcErr creates a new RpcErr with the given code, message, and details.
func newRpcErr(code ErrorCode, message string, details ...string) *RpcErr {
	rpcError := &RpcError{
		Code:    code,
		Message: message,
	}
	if len(details) > 0 {
		rpcError.Details = details[0]
	}
	return &RpcErr{rpcError: rpcError}
}

// InvalidArgumentError creates an INVALID_ARGUMENT error.
func InvalidArgumentError(message string, details ...string) error {
	return newRpcErr(ErrorCode_ERROR_CODE_INVALID_ARGUMENT, message, details...)
}

// NotFoundError creates a NOT_FOUND error.
func NotFoundError(message string, details ...string) error {
	return newRpcErr(ErrorCode_ERROR_CODE_NOT_FOUND, message, details...)
}

// InternalServerError creates an INTERNAL error.
func InternalServerError(message string, details ...string) error {
	return newRpcErr(ErrorCode_ERROR_CODE_INTERNAL, message, details...)
}

// UnavailableError creates an UNAVAILABLE error.
func UnavailableError(message string, details ...string) error {
	return newRpcErr(ErrorCode_ERROR_CODE_UNAVAILABLE, message, details...)
}

// WrapError wraps a Go error as an INTERNAL error.
func WrapError(err error, message string) error {
	return newRpcErr(ErrorCode_ERROR_CODE_INTERNAL, message, err.Error())
}

// WrapErrorf wraps a Go error as an INTERNAL error with a formatted message.
func WrapErrorf(err error, format string, args ...any) error {
	return newRpcErr(ErrorCode_ERROR_CODE_INTERNAL, fmt.Sprintf(format, args...), err.Error())
}

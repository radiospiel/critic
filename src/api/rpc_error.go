package api

// ErrorCode represents the type of RPC error.
type ErrorCode int32

const (
	ErrorCodeUnspecified     ErrorCode = 0
	ErrorCodeInvalidArgument ErrorCode = 1
	ErrorCodeNotFound        ErrorCode = 2
	ErrorCodeInternal        ErrorCode = 3
	ErrorCodeUnavailable     ErrorCode = 4
)

var errorCodeNames = map[ErrorCode]string{
	ErrorCodeUnspecified:     "UNSPECIFIED",
	ErrorCodeInvalidArgument: "INVALID_ARGUMENT",
	ErrorCodeNotFound:        "NOT_FOUND",
	ErrorCodeInternal:        "INTERNAL",
	ErrorCodeUnavailable:     "UNAVAILABLE",
}

func (c ErrorCode) String() string {
	if name, ok := errorCodeNames[c]; ok {
		return name
	}
	return "UNKNOWN"
}

// RpcError represents a structured error response.
type RpcError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Details string    `json:"details,omitempty"`
}

// NewRpcError creates a new RpcError with the given code and message.
func NewRpcError(code ErrorCode, message string) *RpcError {
	return &RpcError{
		Code:    code,
		Message: message,
	}
}

// WithDetails adds details to the error and returns it.
func (e *RpcError) WithDetails(details string) *RpcError {
	e.Details = details
	return e
}

// Error implements the error interface.
func (e *RpcError) Error() string {
	if e.Details != "" {
		return e.Code.String() + ": " + e.Message + " - " + e.Details
	}
	return e.Code.String() + ": " + e.Message
}

// InvalidArgument creates an INVALID_ARGUMENT error.
func InvalidArgument(message string) *RpcError {
	return NewRpcError(ErrorCodeInvalidArgument, message)
}

// NotFound creates a NOT_FOUND error.
func NotFound(message string) *RpcError {
	return NewRpcError(ErrorCodeNotFound, message)
}

// InternalError creates an INTERNAL error.
func InternalError(message string) *RpcError {
	return NewRpcError(ErrorCodeInternal, message)
}

// Unavailable creates an UNAVAILABLE error.
func Unavailable(message string) *RpcError {
	return NewRpcError(ErrorCodeUnavailable, message)
}

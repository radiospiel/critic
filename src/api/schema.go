package api

import (
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// requestSchemaStrings defines JSON schemas as JSON string literals.
// This makes schemas easier to read and maintain.
var requestSchemaStrings = map[string]string{
	"/critic.v1.CriticService/GetLastChange": `{
		"type": "object",
		"properties": {}
	}`,
	"/critic.v1.CriticService/GetDiffSummary": `{
		"type": "object",
		"properties": {}
	}`,
	"/critic.v1.CriticService/GetDiff": `{
		"type": "object",
		"properties": {
			"path": {"type": "string", "minLength": 1}
		},
		"required": ["path"]
	}`,
	"/critic.v1.CriticService/GetFile": `{
		"type": "object",
		"properties": {
			"path": {"type": "string", "minLength": 1}
		},
		"required": ["path"]
	}`,
}

// RequestSchemas maps procedure names to their compiled JSON schemas.
var RequestSchemas map[string]*jsonschema.Schema

func init() {
	RequestSchemas = make(map[string]*jsonschema.Schema)
	for procedure, schemaStr := range requestSchemaStrings {
		schema, err := jsonschema.CompileString(procedure, schemaStr)
		if err != nil {
			panic(fmt.Sprintf("failed to compile schema for %s: %v", procedure, err))
		}
		RequestSchemas[procedure] = schema
	}
}

// ValidateRequest validates a request against its JSON schema.
// Returns nil if valid, or an error describing validation failures.
func ValidateRequest(procedure string, data map[string]any) error {
	schema, ok := RequestSchemas[procedure]
	if !ok {
		// No schema defined, skip validation
		return nil
	}

	return schema.Validate(data)
}

// FormatValidationError converts a jsonschema validation error to a user-friendly string.
func FormatValidationError(err error) string {
	if err == nil {
		return ""
	}

	validationErr, ok := err.(*jsonschema.ValidationError)
	if !ok {
		return err.Error()
	}

	var messages []string
	collectErrors(validationErr, &messages)
	return strings.Join(messages, "; ")
}

func collectErrors(err *jsonschema.ValidationError, messages *[]string) {
	if len(err.Causes) == 0 {
		// Leaf error - format the message
		path := err.InstanceLocation
		if path == "" {
			path = "/"
		}
		*messages = append(*messages, fmt.Sprintf("%s: %s", path, err.Message))
	}
	for _, cause := range err.Causes {
		collectErrors(cause, messages)
	}
}

// NewRpcError creates a new RpcError with the given code and message.
func NewRpcError(code ErrorCode, message string) *RpcError {
	return &RpcError{
		Code:    code,
		Message: message,
	}
}

// InvalidArgument creates an INVALID_ARGUMENT error.
func InvalidArgument(message string) *RpcError {
	return NewRpcError(ErrorCode_ERROR_CODE_INVALID_ARGUMENT, message)
}

// NotFound creates a NOT_FOUND error.
func NotFound(message string) *RpcError {
	return NewRpcError(ErrorCode_ERROR_CODE_NOT_FOUND, message)
}

// InternalError creates an INTERNAL error.
func InternalError(message string) *RpcError {
	return NewRpcError(ErrorCode_ERROR_CODE_INTERNAL, message)
}

// Unavailable creates an UNAVAILABLE error.
func Unavailable(message string) *RpcError {
	return NewRpcError(ErrorCode_ERROR_CODE_UNAVAILABLE, message)
}

// RpcErrorMessage returns a formatted error message for the RpcError.
func RpcErrorMessage(e *RpcError) string {
	if e.Details != "" {
		return e.Code.String() + ": " + e.Message + " - " + e.Details
	}
	return e.Code.String() + ": " + e.Message
}

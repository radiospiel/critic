package api

import (
	"encoding/json"
	"fmt"
)

// JSONSchema represents a JSON Schema definition.
type JSONSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]*JSONSchema `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
	Enum       []any                  `json:"enum,omitempty"`
	MinLength  *int                   `json:"minLength,omitempty"`
	MaxLength  *int                   `json:"maxLength,omitempty"`
}

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
}

// RequestSchemas maps procedure names to their parsed JSON schemas.
var RequestSchemas map[string]*JSONSchema

func init() {
	RequestSchemas = make(map[string]*JSONSchema)
	for procedure, schemaStr := range requestSchemaStrings {
		var schema JSONSchema
		if err := json.Unmarshal([]byte(schemaStr), &schema); err != nil {
			panic(fmt.Sprintf("failed to parse schema for %s: %v", procedure, err))
		}
		RequestSchemas[procedure] = &schema
	}
}

// ValidationError represents a schema validation error.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidateRequest validates a request against its JSON schema.
func ValidateRequest(procedure string, data map[string]any) []*ValidationError {
	schema, ok := RequestSchemas[procedure]
	if !ok {
		// No schema defined, skip validation
		return nil
	}

	return validateObject(schema, data, "")
}

func validateObject(schema *JSONSchema, data map[string]any, path string) []*ValidationError {
	var errors []*ValidationError

	// Check required fields
	for _, required := range schema.Required {
		if _, ok := data[required]; !ok {
			fieldPath := required
			if path != "" {
				fieldPath = path + "." + required
			}
			errors = append(errors, &ValidationError{
				Field:   fieldPath,
				Message: "required field is missing",
			})
		}
	}

	// Validate properties
	for name, propSchema := range schema.Properties {
		value, ok := data[name]
		if !ok {
			continue
		}

		fieldPath := name
		if path != "" {
			fieldPath = path + "." + name
		}

		propErrors := validateValue(propSchema, value, fieldPath)
		errors = append(errors, propErrors...)
	}

	return errors
}

func validateValue(schema *JSONSchema, value any, path string) []*ValidationError {
	var errors []*ValidationError

	switch schema.Type {
	case "string":
		str, ok := value.(string)
		if !ok {
			errors = append(errors, &ValidationError{
				Field:   path,
				Message: fmt.Sprintf("expected string, got %T", value),
			})
			return errors
		}

		if schema.MinLength != nil && len(str) < *schema.MinLength {
			errors = append(errors, &ValidationError{
				Field:   path,
				Message: fmt.Sprintf("string length %d is less than minimum %d", len(str), *schema.MinLength),
			})
		}

		if schema.MaxLength != nil && len(str) > *schema.MaxLength {
			errors = append(errors, &ValidationError{
				Field:   path,
				Message: fmt.Sprintf("string length %d exceeds maximum %d", len(str), *schema.MaxLength),
			})
		}

		if schema.Enum != nil {
			found := false
			for _, e := range schema.Enum {
				if e == str {
					found = true
					break
				}
			}
			if !found {
				errors = append(errors, &ValidationError{
					Field:   path,
					Message: fmt.Sprintf("value %q is not in allowed enum values", str),
				})
			}
		}

	case "object":
		obj, ok := value.(map[string]any)
		if !ok {
			errors = append(errors, &ValidationError{
				Field:   path,
				Message: fmt.Sprintf("expected object, got %T", value),
			})
			return errors
		}
		errors = append(errors, validateObject(schema, obj, path)...)
	}

	return errors
}

// ProtoToJSON converts a protobuf message to its JSON representation.
func ProtoToJSON(msg any) (string, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ProtoToMap converts a protobuf message to a map for validation.
func ProtoToMap(msg any) (map[string]any, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
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

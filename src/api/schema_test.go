package api

import (
	"testing"

	"github.com/radiospiel/critic/simple-go/assert"
)

func TestValidateRequest_GetLastChange(t *testing.T) {
	procedure := "/critic.v1.CriticService/GetLastChange"

	// Empty request should be valid (no required fields)
	errors := ValidateRequest(procedure, map[string]any{})
	assert.Equals(t, len(errors), 0, "empty GetLastChange request should be valid")
}

func TestValidateRequest_GetDiffSummary(t *testing.T) {
	procedure := "/critic.v1.CriticService/GetDiffSummary"

	// Empty request should be valid (no required fields)
	errors := ValidateRequest(procedure, map[string]any{})
	assert.Equals(t, len(errors), 0, "empty GetDiffSummary request should be valid")
}

func TestValidateRequest_GetDiff(t *testing.T) {
	procedure := "/critic.v1.CriticService/GetDiff"

	// Missing required field
	errors := ValidateRequest(procedure, map[string]any{})
	assert.Equals(t, len(errors), 1, "missing path should produce one error")
	assert.Equals(t, errors[0].Field, "path", "error should be for path field")
	assert.Contains(t, errors[0].Message, "required", "error message should mention required")

	// Empty path (violates minLength)
	errors = ValidateRequest(procedure, map[string]any{"path": ""})
	assert.Equals(t, len(errors), 1, "empty path should produce one error")
	assert.Equals(t, errors[0].Field, "path", "error should be for path field")

	// Valid request
	errors = ValidateRequest(procedure, map[string]any{"path": "src/main.go"})
	assert.Equals(t, len(errors), 0, "valid GetDiff request should have no errors")
}

func TestValidateRequest_UnknownProcedure(t *testing.T) {
	// Unknown procedure should pass (no schema defined)
	errors := ValidateRequest("/unknown/procedure", map[string]any{"foo": "bar"})
	assert.Equals(t, len(errors), 0, "unknown procedure should pass validation")
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

func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Field:   "path",
		Message: "required field is missing",
	}

	assert.Equals(t, err.Error(), "path: required field is missing", "Error() should format correctly")
}

func TestValidateStringType(t *testing.T) {
	schema := &JSONSchema{
		Type: "object",
		Properties: map[string]*JSONSchema{
			"name": {Type: "string"},
		},
	}

	// Valid string
	errors := validateObject(schema, map[string]any{"name": "test"}, "")
	assert.Equals(t, len(errors), 0, "valid string should pass")

	// Wrong type
	errors = validateObject(schema, map[string]any{"name": 123}, "")
	assert.Equals(t, len(errors), 1, "wrong type should fail")
	assert.Contains(t, errors[0].Message, "expected string", "error should mention expected type")
}

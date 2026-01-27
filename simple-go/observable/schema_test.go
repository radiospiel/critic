package observable

import (
	"testing"

	"github.org/radiospiel/critic/simple-go/assert"
)

// Tests for JSON schema validation support

func TestWithSchemaReturnsObservable(t *testing.T) {
	schema := `{"type": "string"}`
	obs := New().WithSchema("name", schema)
	assert.NotNil(t, obs, "WithSchema should return non-nil Observable")
}

func TestWithSchemaChaining(t *testing.T) {
	obs := New().
		WithSchema("name", `{"type": "string"}`).
		WithSchema("age", `{"type": "integer", "minimum": 0}`)
	assert.NotNil(t, obs, "chained WithSchema should work")
}

func TestSchemaValidationPassesForValidString(t *testing.T) {
	obs := New().WithSchema("name", `{"type": "string"}`)

	// Should not panic
	obs.SetValueAtKey("name", "Alice")
	assert.Equals(t, obs.GetValue("name"), "Alice", "should set valid string")
}

func TestSchemaValidationPassesForValidNumber(t *testing.T) {
	obs := New().WithSchema("age", `{"type": "integer", "minimum": 0, "maximum": 150}`)

	obs.SetValueAtKey("age", 30)
	assert.Equals(t, obs.GetValue("age"), 30, "should set valid integer")
}

func TestSchemaValidationPassesForValidObject(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"}
		},
		"required": ["name"]
	}`
	obs := New().WithSchema("user", schema)

	obs.SetValueAtKey("user", map[string]any{
		"name": "Alice",
		"age":  30,
	})
	assert.Equals(t, obs.GetValue("user.name"), "Alice", "should set valid object")
}

func TestSchemaValidationFailsForInvalidType(t *testing.T) {
	obs := New().WithSchema("name", `{"type": "string"}`)

	err := obs.SetValueAtKey("name", 123) // number instead of string
	assert.NotNil(t, err, "should return error when setting invalid type")
}

func TestSchemaValidationFailsForInvalidNumber(t *testing.T) {
	obs := New().WithSchema("age", `{"type": "integer", "minimum": 0}`)

	err := obs.SetValueAtKey("age", -5) // negative number
	assert.NotNil(t, err, "should return error when setting negative age")
}

func TestSchemaValidationFailsForMissingRequired(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"}
		},
		"required": ["name"]
	}`
	obs := New().WithSchema("user", schema)

	err := obs.SetValueAtKey("user", map[string]any{
		"age": 30, // missing required "name"
	})
	assert.NotNil(t, err, "should return error when required field is missing")
}

func TestSchemaValidationForArray(t *testing.T) {
	schema := `{
		"type": "array",
		"items": {"type": "string"}
	}`
	obs := New().WithSchema("tags", schema)

	// Valid array of strings
	obs.SetValueAtKey("tags", []any{"go", "json", "schema"})
	tags := obs.GetValue("tags").([]any)
	assert.Equals(t, len(tags), 3, "should have 3 tags")
}

func TestSchemaValidationFailsForInvalidArrayItem(t *testing.T) {
	schema := `{
		"type": "array",
		"items": {"type": "string"}
	}`
	obs := New().WithSchema("tags", schema)

	err := obs.SetValueAtKey("tags", []any{"go", 123, "schema"}) // 123 is not a string
	assert.NotNil(t, err, "should return error when array contains non-string")
}

func TestSchemaDoesNotAffectUnrelatedKeys(t *testing.T) {
	obs := New().WithSchema("name", `{"type": "string"}`)

	// Setting a different key should not be validated
	err := obs.SetValueAtKey("count", 42)
	assert.Nil(t, err, "should not return error for unrelated key")
	assert.Equals(t, obs.GetValue("count"), 42, "unrelated key should be set without validation")
}

func TestSchemaValidationForNestedKey(t *testing.T) {
	obs := New().WithSchema("config.port", `{"type": "integer", "minimum": 1, "maximum": 65535}`)

	err := obs.SetValueAtKey("config.port", 8080)
	assert.Nil(t, err, "should not return error for valid port")
	assert.Equals(t, obs.GetValue("config.port"), 8080, "should set valid port")
}

func TestSchemaValidationFailsForInvalidNestedKey(t *testing.T) {
	obs := New().WithSchema("config.port", `{"type": "integer", "minimum": 1, "maximum": 65535}`)

	err := obs.SetValueAtKey("config.port", 99999) // out of range
	assert.NotNil(t, err, "should return error when port is out of range")
}

func TestSchemaOnParentValidatesChildKeys(t *testing.T) {
	// When a schema is set on "config", setting "config.name" should validate
	// the entire "config" object after the change
	schema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"port": {"type": "integer"}
		}
	}`
	obs := New().WithSchema("config", schema)

	// First set a valid config
	err := obs.SetValueAtKey("config", map[string]any{"name": "server", "port": 8080})
	assert.Nil(t, err, "should not return error for valid config")

	// Now try to set an invalid child value
	err = obs.SetValueAtKey("config.port", "not-a-number") // string instead of integer
	assert.NotNil(t, err, "should return error when child value violates parent schema")
}

func TestSchemaValidationWithMapSchema(t *testing.T) {
	// Test that schema can be provided as map[string]any
	schema := map[string]any{
		"type":    "string",
		"pattern": "^[a-z]+$",
	}
	obs := New().WithSchema("id", schema)

	err := obs.SetValueAtKey("id", "abc")
	assert.Nil(t, err, "should not return error for valid lowercase string")
	assert.Equals(t, obs.GetValue("id"), "abc", "should set valid lowercase string")
}

func TestSchemaValidationWithMapSchemaFails(t *testing.T) {
	schema := map[string]any{
		"type":    "string",
		"pattern": "^[a-z]+$",
	}
	obs := New().WithSchema("id", schema)

	err := obs.SetValueAtKey("id", "ABC123") // doesn't match pattern
	assert.NotNil(t, err, "should return error when pattern doesn't match")
}

func TestSchemaValidationAllowsNilForOptionalKey(t *testing.T) {
	obs := New().WithSchema("name", `{"type": "string"}`)

	// Setting to nil (deletion) should be allowed
	err := obs.SetValueAtKey("name", "Alice")
	assert.Nil(t, err, "should not return error for valid string")
	err = obs.SetValueAtKey("name", nil)
	assert.Nil(t, err, "should not return error for nil deletion")
	assert.Nil(t, obs.GetValue("name"), "should allow deletion via nil")
}

func TestSchemaValidationWithBoolean(t *testing.T) {
	obs := New().WithSchema("enabled", `{"type": "boolean"}`)

	err := obs.SetValueAtKey("enabled", true)
	assert.Nil(t, err, "should not return error for valid boolean")
	assert.True(t, obs.GetValue("enabled").(bool), "should set boolean")
}

func TestSchemaValidationFailsForBooleanWithWrongType(t *testing.T) {
	obs := New().WithSchema("enabled", `{"type": "boolean"}`)

	err := obs.SetValueAtKey("enabled", "yes") // string instead of boolean
	assert.NotNil(t, err, "should return error when setting non-boolean")
}

func TestWithSchemaOnNewWithData(t *testing.T) {
	data := map[string]any{"count": 10}
	obs := NewWithData(data).WithSchema("count", `{"type": "integer", "minimum": 0}`)

	// Should be able to update with valid value
	err := obs.SetValueAtKey("count", 20)
	assert.Nil(t, err, "should not return error for valid integer")
	assert.Equals(t, obs.GetValue("count"), 20, "should update to valid value")
}

func TestMultipleSchemasOnDifferentKeys(t *testing.T) {
	obs := New().
		WithSchema("name", `{"type": "string"}`).
		WithSchema("age", `{"type": "integer", "minimum": 0}`).
		WithSchema("email", `{"type": "string", "format": "email"}`)

	err := obs.SetValueAtKey("name", "Alice")
	assert.Nil(t, err, "should not return error for valid name")
	err = obs.SetValueAtKey("age", 30)
	assert.Nil(t, err, "should not return error for valid age")
	// Note: email format validation may be lenient in some implementations
	err = obs.SetValueAtKey("email", "alice@example.com")
	assert.Nil(t, err, "should not return error for valid email")

	assert.Equals(t, obs.GetValue("name"), "Alice", "name should be set")
	assert.Equals(t, obs.GetValue("age"), 30, "age should be set")
	assert.Equals(t, obs.GetValue("email"), "alice@example.com", "email should be set")
}

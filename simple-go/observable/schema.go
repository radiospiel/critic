package observable

import (
	"encoding/json"
	"strings"

	"git.15b.it/eno/critic/simple-go/preconditions"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

// schemaEntry holds a compiled schema and its source for a key.
type schemaEntry struct {
	schema *jsonschema.Schema
	key    string
}

// WithSchema configures a JSON schema for the specified key.
// The schema can be a JSON string or a map[string]any.
// When SetValueAtKey is called on this key (or a child key), the value
// will be validated against the schema.
// Panics if the schema is invalid.
// Returns the Observable for chaining.
func (o *Observable) WithSchema(key string, schema any) *Observable {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Initialize schemas map if needed
	if o.schemas == nil {
		o.schemas = make(map[string]*schemaEntry)
	}

	// Convert schema to JSON bytes
	var schemaBytes []byte
	var err error

	switch s := schema.(type) {
	case string:
		schemaBytes = []byte(s)
	case map[string]any:
		schemaBytes, err = json.Marshal(s)
		preconditions.Check(err == nil, "failed to marshal schema for key %q: %v", key, err)
	default:
		preconditions.Check(false, "schema must be a string or map[string]any, got %T", schema)
	}

	// Compile the schema
	compiler := jsonschema.NewCompiler()
	schemaID := "schema://" + key
	err = compiler.AddResource(schemaID, strings.NewReader(string(schemaBytes)))
	preconditions.Check(err == nil, "invalid schema for key %q: %v", key, err)

	compiled, err := compiler.Compile(schemaID)
	preconditions.Check(err == nil, "failed to compile schema for key %q: %v", key, err)

	o.schemas[key] = &schemaEntry{
		schema: compiled,
		key:    key,
	}

	return o
}

// validateAgainstSchema checks if the value at the given key is valid according
// to any applicable schema. Returns an error message if validation fails, or empty string if valid.
// Must be called with lock held.
func (o *Observable) validateAgainstSchema(key string, value any) string {
	if o.schemas == nil || len(o.schemas) == 0 {
		return ""
	}

	// Setting to nil (deletion) is always allowed
	if value == nil {
		return ""
	}

	// Check for exact match schema
	if entry, ok := o.schemas[key]; ok {
		if err := entry.schema.Validate(value); err != nil {
			return formatValidationError(key, err)
		}
		return ""
	}

	// Check if there's a parent schema that applies
	// If we're setting "config.port", check if there's a schema for "config"
	parts := strings.Split(key, ".")
	for i := len(parts) - 1; i > 0; i-- {
		parentKey := strings.Join(parts[:i], ".")
		if entry, ok := o.schemas[parentKey]; ok {
			// We need to validate the entire parent object after the change
			// First, get the current parent value
			parentValue := o.getValueInternal(parentKey)

			// Simulate what the parent will look like after this change
			childKey := strings.Join(parts[i:], ".")
			simulatedParent := simulateNestedSet(parentValue, childKey, value)

			if err := entry.schema.Validate(simulatedParent); err != nil {
				return formatValidationError(parentKey, err)
			}
			return ""
		}
	}

	return ""
}

// simulateNestedSet creates a copy of the parent with the nested value set.
func simulateNestedSet(parent any, childKey string, value any) any {
	if parent == nil {
		parent = make(map[string]any)
	}

	// Make a shallow copy of the parent
	var result any
	switch p := parent.(type) {
	case map[string]any:
		result = copyMap(p)
	case []any:
		result = copySlice(p)
	default:
		// If parent is a primitive, we can't set a child - this will fail validation
		result = parent
	}

	// Now set the child value in the copy
	parts := strings.Split(childKey, ".")
	setNestedValue(result, parts, value)

	return result
}

// setNestedValue sets a value in a nested structure, modifying the container in place.
func setNestedValue(container any, parts []string, value any) {
	if len(parts) == 0 {
		return
	}

	part := parts[0]
	isLast := len(parts) == 1

	if m, ok := container.(map[string]any); ok {
		if isLast {
			m[part] = value
		} else {
			if m[part] == nil {
				// Determine if next part is numeric
				if _, isNum := parseIndex(parts[1]); isNum {
					m[part] = make([]any, 0)
				} else {
					m[part] = make(map[string]any)
				}
			}
			setNestedValue(m[part], parts[1:], value)
		}
	} else if s, ok := container.([]any); ok {
		if idx, isNum := parseIndex(part); isNum && idx < len(s) {
			if isLast {
				s[idx] = value
			} else {
				if s[idx] == nil {
					if _, nextIsNum := parseIndex(parts[1]); nextIsNum {
						s[idx] = make([]any, 0)
					} else {
						s[idx] = make(map[string]any)
					}
				}
				setNestedValue(s[idx], parts[1:], value)
			}
		}
	}
}

// formatValidationError creates a human-readable error message from a validation error.
func formatValidationError(key string, err error) string {
	if validationErr, ok := err.(*jsonschema.ValidationError); ok {
		// Get the first detailed error
		causes := validationErr.BasicOutput().Errors
		if len(causes) > 0 {
			for _, cause := range causes {
				if cause.Error != "" {
					return "schema validation failed for key \"" + key + "\": " + cause.Error
				}
			}
		}
		return "schema validation failed for key \"" + key + "\": " + validationErr.Error()
	}
	return "schema validation failed for key \"" + key + "\": " + err.Error()
}

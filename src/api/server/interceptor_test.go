package server

import (
	"testing"

	"github.com/radiospiel/critic/simple-go/assert"
	"github.com/radiospiel/critic/src/api"
)

func TestToJSON(t *testing.T) {
	// Test with nil
	result := toJSON(nil)
	assert.Equals(t, result, "null", "nil should be 'null'")

	// Test with simple struct
	data := struct {
		Name string `json:"name"`
	}{Name: "test"}
	result = toJSON(data)
	assert.Contains(t, result, `"name":"test"`, "should contain JSON field")

	// Test with map
	m := map[string]any{"key": "value"}
	result = toJSON(m)
	assert.Contains(t, result, `"key":"value"`, "should contain JSON field")
}

func TestValidateRequest_GetDiff_MissingPath(t *testing.T) {
	procedure := "/critic.v1.CriticService/GetDiff"
	req := &api.GetDiffRequest{}

	err := validateRequest(procedure, req)
	assert.NotNil(t, err, "should return error for missing path")
	assert.Contains(t, err.Error(), "path", "should mention path field")
}

func TestValidateRequest_GetDiff_EmptyPath(t *testing.T) {
	procedure := "/critic.v1.CriticService/GetDiff"
	req := &api.GetDiffRequest{Path: ""}

	err := validateRequest(procedure, req)
	assert.NotNil(t, err, "should return error for empty path")
}

func TestValidateRequest_GetDiff_ValidPath(t *testing.T) {
	procedure := "/critic.v1.CriticService/GetDiff"
	req := &api.GetDiffRequest{Path: "src/main.go"}

	err := validateRequest(procedure, req)
	assert.Nil(t, err, "should not return error for valid request")
}

func TestValidateRequest_GetLastChange(t *testing.T) {
	procedure := "/critic.v1.CriticService/GetLastChange"
	req := &api.GetLastChangeRequest{}

	err := validateRequest(procedure, req)
	assert.Nil(t, err, "should not return error for GetLastChange")
}

func TestValidateRequest_GetDiffSummary(t *testing.T) {
	procedure := "/critic.v1.CriticService/GetDiffSummary"
	req := &api.GetDiffSummaryRequest{}

	err := validateRequest(procedure, req)
	assert.Nil(t, err, "should not return error for GetDiffSummary")
}

func TestValidateRequest_UnknownProcedure(t *testing.T) {
	procedure := "/unknown/procedure"
	req := struct{ Foo string }{Foo: "bar"}

	err := validateRequest(procedure, req)
	assert.Nil(t, err, "should not return error for unknown procedure")
}

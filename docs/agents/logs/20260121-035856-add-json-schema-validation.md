# Task: Add JSON Schema Validation to Observable

**Started:** 2026-01-21 03:58:56
**Ended:** 2026-01-21 04:02:00
**Strategy:** Feature (TDD)
**Status:** Completed
**Complexity:** Medium
**Used Models:** Opus

## Objective
Add JSON schema validation support to the Observable package. Allow configuring schemas on specific keys so that when SetValueAtKey is called, the value is validated against the schema if one is configured for that key (or a parent path).

## Progress
- [x] Explore codebase and understand Observable implementation
- [x] Research JSON schema validation libraries
- [x] Design the schema validation API
- [x] Write failing tests for schema configuration
- [x] Implement schema storage in Observable
- [x] Write failing tests for schema validation on SetValueAtKey
- [x] Implement schema validation in SetValueAtKey
- [x] Write tests for nested key schema matching
- [x] Implement nested key schema matching
- [x] Run all tests and verify (84 tests pass)

## Design Notes
- Using `santhosh-tekuri/jsonschema/v5` library for JSON schema validation
- API: `obs.WithSchema(key string, schema string) *Observable` for configuration
- Validation occurs in SetValueAtKey before changes are applied
- If validation fails, panic with preconditions.Check (consistent with existing error handling)
- Schema matching: exact key match or prefix match (e.g., schema on "config" applies to "config.name")

## Obstacles
None yet.

## Outcome
Successfully implemented JSON schema validation for the Observable package:

**Files changed:**
- `simple-go/observable/observable.go` - Added schemas field to struct and validation call in SetValueAtKey
- `simple-go/observable/schema.go` - New file with WithSchema method and validation logic
- `simple-go/observable/schema_test.go` - 21 comprehensive tests for schema validation
- `go.mod` / `go.sum` - Added `github.com/santhosh-tekuri/jsonschema/v5` dependency

**Features:**
- `WithSchema(key, schema)` method for configuring schemas (chainable)
- Supports JSON string or `map[string]any` schema format
- Validates on exact key match
- Validates child keys against parent schemas (simulates post-change state)
- Setting `nil` (deletion) always allowed
- Panics with descriptive error on validation failure (consistent with existing error handling)

## Learnings
- The preconditions package requires constant format strings, so dynamic error messages need to use `%s` placeholder
- JSON Schema validation libraries in Go are mature; `santhosh-tekuri/jsonschema/v5` supports draft 2020-12
- When validating child key changes against parent schemas, need to simulate the post-change state before validation

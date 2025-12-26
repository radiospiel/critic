package testutils

import (
	"reflect"
	"testing"
)

// AssertEquals checks if actual equals expected and fails the test if not.
// Works with any comparable type (strings, ints, bools, etc.)
func AssertEquals[T comparable](t *testing.T, actual, expected T) {
	t.Helper()
	if actual != expected {
		t.Errorf("AssertEquals failed:\n  Expected: %v\n  Actual:   %v", expected, actual)
	}
}

// AssertNotEquals checks if actual does not equal expected and fails the test if they are equal.
func AssertNotEquals[T comparable](t *testing.T, actual, expected T) {
	t.Helper()
	if actual == expected {
		t.Errorf("AssertNotEquals failed:\n  Expected not to equal: %v\n  Actual:                %v", expected, actual)
	}
}

// AssertTrue checks if the condition is true and fails if not.
func AssertTrue(t *testing.T, condition bool, message string) {
	t.Helper()
	if !condition {
		t.Errorf("AssertTrue failed: %s", message)
	}
}

// AssertFalse checks if the condition is false and fails if not.
func AssertFalse(t *testing.T, condition bool, message string) {
	t.Helper()
	if condition {
		t.Errorf("AssertFalse failed: %s", message)
	}
}

// AssertNil checks if the value is nil and fails if not.
func AssertNil(t *testing.T, value interface{}, message string) {
	t.Helper()
	if !isNil(value) {
		t.Errorf("AssertNil failed: %s\n  Expected: nil\n  Actual:   %v", message, value)
	}
}

// AssertNotNil checks if the value is not nil and fails if it is nil.
func AssertNotNil(t *testing.T, value interface{}, message string) {
	t.Helper()
	if isNil(value) {
		t.Errorf("AssertNotNil failed: %s\n  Expected: not nil\n  Actual:   nil", message)
	}
}

// Helper function to check if a value is nil (handles typed nil pointers)
func isNil(v interface{}) bool {
	if v == nil {
		return true
	}
	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func, reflect.Interface:
		return val.IsNil()
	default:
		return false
	}
}

// AssertError checks if err is not nil and fails if it is nil.
func AssertError(t *testing.T, err error, message string) {
	t.Helper()
	if err == nil {
		t.Errorf("AssertError failed: %s\n  Expected an error but got nil", message)
	}
}

// AssertNoError checks if err is nil and fails if it's not.
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("AssertNoError failed:\n  Expected no error\n  Got:      %v", err)
	}
}

// AssertStringContains checks if str contains substring and fails if not.
func AssertStringContains(t *testing.T, str, substring string) {
	t.Helper()
	if !containsString(str, substring) {
		t.Errorf("AssertStringContains failed:\n  String:    %q\n  Should contain: %q", str, substring)
	}
}

// AssertLen checks if the length of the slice/map/string equals expected length.
func AssertLen(t *testing.T, actual interface{}, expectedLen int) {
	t.Helper()
	actualLen := getLength(actual)
	if actualLen != expectedLen {
		t.Errorf("AssertLen failed:\n  Expected length: %d\n  Actual length:   %d\n  Value: %v", expectedLen, actualLen, actual)
	}
}

// Helper function to check if string contains substring
func containsString(str, substring string) bool {
	for i := 0; i <= len(str)-len(substring); i++ {
		if str[i:i+len(substring)] == substring {
			return true
		}
	}
	return false
}

// Helper function to get length of various types
func getLength(v interface{}) int {
	if v == nil {
		return 0
	}

	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Slice, reflect.Map, reflect.String, reflect.Array, reflect.Chan:
		return val.Len()
	default:
		return 0
	}
}

package assert

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// testingT is an interface wrapper around *testing.T
type testingT interface {
	Helper()
	Errorf(format string, args ...interface{})
}

// Equals checks if actual equals expected and fails the test if not.
// Works with any comparable type (strings, ints, bools, etc.)
// Optional msgAndArgs can provide a custom error message: first arg is format string, rest are values.
func Equals(t testingT, actual any, expected any, msgAndArgs ...interface{}) {
	t.Helper()

	if !isEqual(actual, expected) {
		msg := fmt.Sprintf("Equals failed:\n  Expected: %v\n  Actual:   %v", expected, actual)
		if len(msgAndArgs) > 0 {
			customMsg := formatMessage(msgAndArgs...)
			msg = customMsg + "\n" + msg
		}
		t.Errorf("%s", msg)
	}
}

// NotEquals checks if actual does not equal expected and fails the test if they are equal.
func NotEquals[T comparable](t *testing.T, actual, expected T) {
	t.Helper()
	if isEqual(actual, expected) {
		t.Errorf("NotEquals failed:\n  Expected not to equal: %v\n  Actual:                %v", expected, actual)
	}
}

func isEqual(actual any, expected any) bool {
	if compareAsBytes(actual) && compareAsBytes(expected) {
		return bytes.Equal(bytesFrom(actual), bytesFrom(expected))
	}

	return reflect.DeepEqual(actual, expected)
}

// formatMessage formats a custom error message from variadic arguments.
// First argument should be a format string, remaining arguments are values for formatting.
func formatMessage(msgAndArgs ...interface{}) string {
	if len(msgAndArgs) == 0 {
		return ""
	}
	if len(msgAndArgs) == 1 {
		msg := msgAndArgs[0]
		if msgStr, ok := msg.(string); ok {
			return msgStr
		}
		return fmt.Sprintf("%+v", msg)
	}
	if format, ok := msgAndArgs[0].(string); ok {
		return fmt.Sprintf(format, msgAndArgs[1:]...)
	}
	return fmt.Sprintf("%+v", msgAndArgs[0])
}

func compareAsBytes(content any) bool {
	switch content.(type) {
	case string:
		return true
	case []byte:
		return true
	default:
		return false
	}
}

func bytesFrom(content any) []byte {
	switch v := content.(type) {
	case string:
		return []byte(v)
	case []byte:
		return v
	default:
		panic(fmt.Sprintf("Cannot convert unsupported type %T to []byte, expected string or []byte", content))
	}
}

// True checks if the condition is true and fails if not.
func True(t *testing.T, condition bool, message string) {
	t.Helper()
	if !condition {
		t.Errorf("True failed: %s", message)
	}
}

// False checks if the condition is false and fails if not.
func False(t *testing.T, condition bool, message string) {
	t.Helper()
	if condition {
		t.Errorf("False failed: %s", message)
	}
}

// Nil checks if the value is nil and fails if not.
func Nil(t *testing.T, value interface{}, message string) {
	t.Helper()
	if !isNil(value) {
		t.Errorf("Nil failed: %s\n  Expected: nil\n  Actual:   %v", message, value)
	}
}

// NotNil checks if the value is not nil and fails if it is nil.
func NotNil(t *testing.T, value interface{}, message string) {
	t.Helper()
	if isNil(value) {
		t.Errorf("NotNil failed: %s\n  Expected: not nil\n  Actual:   nil", message)
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

// Error checks if err is not nil and fails if it is nil.
func Error(t *testing.T, err error, message string) {
	t.Helper()
	if err == nil {
		t.Errorf("Error failed: %s\n  Expected an error but got nil", message)
	}
}

// NoError checks if err is nil and fails if it's not.
func NoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("NoError failed:\n  Expected no error\n  Got:      %v", err)
	}
}

// Contains checks if str contains substring and fails if not.
func Contains(t *testing.T, str, substr string) {
	t.Helper()
	if !strings.Contains(str, substr) {
		t.Errorf("Contains failed:\n  String:    %q\n  Should contain: %q", str, substr)
	}
}

// Length checks if the length of the slice/map/string equals expected length.
func Length(t *testing.T, actual interface{}, expectedLen int) {
	t.Helper()
	actualLen := getLength(actual)
	if actualLen != expectedLen {
		t.Errorf("Length failed:\n  Expected length: %d\n  Actual length:   %d\n  Value: %v", expectedLen, actualLen, actual)
	}
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
		panic(fmt.Sprintf("getLength called on unsupported type %T", v))
	}
}

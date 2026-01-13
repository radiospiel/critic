package assert

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// mockTestingT is a minimal mock of testing.T for testing assert functions
type mockTestingT struct {
	failed     bool
	fatalCalled bool
	errorMsg   string
}

func (m *mockTestingT) Helper() {}

func (m *mockTestingT) Error(args ...interface{}) {
	m.failed = true
	m.errorMsg = fmt.Sprint(args...)
}

func (m *mockTestingT) Fatal(args ...interface{}) {
	m.failed = true
	m.fatalCalled = true
	m.errorMsg = fmt.Sprint(args...)
}

func (m *mockTestingT) expectMockSuccessful(t *testing.T) {
	t.Helper()
	if m.failed {
		t.Errorf("Expected mock to succeed, but it failed with: %s", m.errorMsg)
	}
}

func (m *mockTestingT) expectMockFailure(t *testing.T, expectedSubstring string) {
	t.Helper()
	if !m.failed {
		t.Error("Expected mock to fail, but it succeeded")
	}
	if expectedSubstring != "" && !strings.Contains(m.errorMsg, expectedSubstring) {
		t.Errorf("Expected error message to contain %q, got: %s", expectedSubstring, m.errorMsg)
	}
}

func (m *mockTestingT) expectMockFatal(t *testing.T, expectedSubstring string) {
	t.Helper()
	if !m.fatalCalled {
		t.Error("Expected Fatal to be called, but it wasn't")
	}
	if expectedSubstring != "" && !strings.Contains(m.errorMsg, expectedSubstring) {
		t.Errorf("Expected error message to contain %q, got: %s", expectedSubstring, m.errorMsg)
	}
}

func (m *mockTestingT) resetMock() {
	m.failed = false
	m.fatalCalled = false
	m.errorMsg = ""
}

func TestAssertEquals(t *testing.T) {
	m := &mockTestingT{}

	// These should pass
	Equals(m, 42, 42)
	m.expectMockSuccessful(t)
	m.resetMock()

	Equals(m, "hello", "hello")
	m.expectMockSuccessful(t)
	m.resetMock()

	Equals(m, true, true)
	m.expectMockSuccessful(t)
}

func TestAssertNotEquals(t *testing.T) {
	m := &mockTestingT{}

	// These should pass
	NotEquals(m, 42, 43)
	m.expectMockSuccessful(t)
	m.resetMock()

	NotEquals(m, "hello", "world")
	m.expectMockSuccessful(t)
	m.resetMock()

	NotEquals(m, true, false)
	m.expectMockSuccessful(t)
}

func TestAssertTrue(t *testing.T) {
	m := &mockTestingT{}

	True(m, true, "Should be true")
	m.expectMockSuccessful(t)
	m.resetMock()

	True(m, 1 == 1, "1 should equal 1")
	m.expectMockSuccessful(t)
}

func TestAssertFalse(t *testing.T) {
	m := &mockTestingT{}

	False(m, false, "Should be false")
	m.expectMockSuccessful(t)
	m.resetMock()

	False(m, 1 == 2, "1 should not equal 2")
	m.expectMockSuccessful(t)
}

func TestAssertNil(t *testing.T) {
	m := &mockTestingT{}
	var nilPtr *int

	Nil(m, nil, "nil should be nil")
	m.expectMockSuccessful(t)
	m.resetMock()

	Nil(m, nilPtr, "nil pointer should be nil")
	m.expectMockSuccessful(t)
}

func TestAssertNotNil(t *testing.T) {
	m := &mockTestingT{}
	value := 42

	NotNil(m, &value, "pointer should not be nil")
	m.expectMockSuccessful(t)
	m.resetMock()

	NotNil(m, "string", "string should not be nil")
	m.expectMockSuccessful(t)
}

func TestAssertError(t *testing.T) {
	m := &mockTestingT{}

	// Should pass - error contains expected string
	err := errors.New("test error")
	Error(m, err, "test error")
	m.expectMockSuccessful(t)
	m.resetMock()

	// Should pass - error contains substring
	Error(m, err, "test")
	m.expectMockSuccessful(t)
	m.resetMock()

	// Should fail - error is nil
	Error(m, nil, "test error")
	m.expectMockFailure(t, "Expected an error but got nil")
	m.resetMock()

	// Should fail - error doesn't contain expected string
	Error(m, err, "different error")
	m.expectMockFailure(t, "Expected error to contain")
}

func TestAssertNoError(t *testing.T) {
	m := &mockTestingT{}

	NoError(m, nil)
	m.expectMockSuccessful(t)
}

func TestAssertContainsString(t *testing.T) {
	m := &mockTestingT{}

	Contains(m, "hello world", "world")
	m.expectMockSuccessful(t)
	m.resetMock()

	Contains(m, "package main", "main")
	m.expectMockSuccessful(t)
	m.resetMock()

	// Test failure case
	Contains(m, "hello world", "foo")
	m.expectMockFailure(t, "assert.Contains")
}

func TestAssertEqualsWithCustomMessage(t *testing.T) {
	m := &mockTestingT{}

	// This should fail and include the custom message
	Equals(m, 42, 100, "Expected value to be %d but got %d", 100, 42)
	m.expectMockFailure(t, "Expected value to be 100 but got 42")
}

func TestAssertContains(t *testing.T) {
	m := &mockTestingT{}

	// Test with string slice
	Contains(m, []string{"a", "b", "c"}, "b")
	m.expectMockSuccessful(t)
	m.resetMock()

	// Test with int slice
	Contains(m, []int{1, 2, 3}, 2)
	m.expectMockSuccessful(t)
	m.resetMock()

	// Test failure case
	Contains(m, []string{"a", "b", "c"}, "d")
	m.expectMockFailure(t, "assert.Contains")
}

func TestAssertNotContains(t *testing.T) {
	m := &mockTestingT{}

	// Test with string slice
	NotContains(m, []string{"a", "b", "c"}, "d")
	m.expectMockSuccessful(t)
	m.resetMock()

	// Test failure case
	NotContains(m, []string{"a", "b", "c"}, "b")
	m.expectMockFailure(t, "assert.NotContains")
}

func TestAssertLen(t *testing.T) {
	m := &mockTestingT{}

	// Test with slice
	Len(m, []int{1, 2, 3}, 3)
	m.expectMockSuccessful(t)
	m.resetMock()

	// Test with empty slice
	Len(m, []string{}, 0)
	m.expectMockSuccessful(t)
	m.resetMock()

	// Test with string
	Len(m, "hello", 5)
	m.expectMockSuccessful(t)
	m.resetMock()

	// Test with map
	Len(m, map[string]int{"a": 1, "b": 2}, 2)
	m.expectMockSuccessful(t)
	m.resetMock()

	// Test failure case - wrong length
	Len(m, []int{1, 2, 3}, 5)
	m.expectMockFailure(t, "assert.Len")
	m.resetMock()

	// Test failure case - unsupported type
	Len(m, 42, 1)
	m.expectMockFailure(t, "Unsupported type")
}

// ============================================================================
// Fatal variant tests
// ============================================================================

func TestAssertEqualsF(t *testing.T) {
	m := &mockTestingT{}

	// Should pass
	EqualsF(m, 42, 42)
	m.expectMockSuccessful(t)
	m.resetMock()

	// Should fail with Fatal
	EqualsF(m, 42, 100)
	m.expectMockFatal(t, "assert.EqualsF")
}

func TestAssertTrueF(t *testing.T) {
	m := &mockTestingT{}

	// Should pass
	TrueF(m, true)
	m.expectMockSuccessful(t)
	m.resetMock()

	// Should fail with Fatal
	TrueF(m, false)
	m.expectMockFatal(t, "assert.TrueF")
}

func TestAssertFalseF(t *testing.T) {
	m := &mockTestingT{}

	// Should pass
	FalseF(m, false)
	m.expectMockSuccessful(t)
	m.resetMock()

	// Should fail with Fatal
	FalseF(m, true)
	m.expectMockFatal(t, "assert.FalseF")
}

func TestAssertNilF(t *testing.T) {
	m := &mockTestingT{}

	// Should pass
	NilF(m, nil)
	m.expectMockSuccessful(t)
	m.resetMock()

	// Should fail with Fatal
	NilF(m, "not nil")
	m.expectMockFatal(t, "assert.NilF")
}

func TestAssertNotNilF(t *testing.T) {
	m := &mockTestingT{}

	// Should pass
	NotNilF(m, "not nil")
	m.expectMockSuccessful(t)
	m.resetMock()

	// Should fail with Fatal
	NotNilF(m, nil)
	m.expectMockFatal(t, "assert.NotNilF")
}

func TestAssertNoErrorF(t *testing.T) {
	m := &mockTestingT{}

	// Should pass
	NoErrorF(m, nil)
	m.expectMockSuccessful(t)
	m.resetMock()

	// Should fail with Fatal
	NoErrorF(m, errors.New("some error"))
	m.expectMockFatal(t, "assert.NoErrorF")
}

func TestAssertLenF(t *testing.T) {
	m := &mockTestingT{}

	// Should pass
	LenF(m, []int{1, 2, 3}, 3)
	m.expectMockSuccessful(t)
	m.resetMock()

	// Should fail with Fatal
	LenF(m, []int{1, 2, 3}, 5)
	m.expectMockFatal(t, "assert.LenF")
}

func TestAssertContainsF(t *testing.T) {
	m := &mockTestingT{}

	// Should pass with string
	ContainsF(m, "hello world", "world")
	m.expectMockSuccessful(t)
	m.resetMock()

	// Should pass with slice
	ContainsF(m, []string{"a", "b", "c"}, "b")
	m.expectMockSuccessful(t)
	m.resetMock()

	// Should fail with Fatal
	ContainsF(m, "hello", "world")
	m.expectMockFatal(t, "assert.ContainsF")
}

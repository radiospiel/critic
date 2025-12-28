package assert

import (
	"errors"
	"testing"
)

func TestAssertEquals(t *testing.T) {
	// This should pass
	Equals(t, 42, 42)
	Equals(t, "hello", "hello")
	Equals(t, true, true)
}

func TestAssertNotEquals(t *testing.T) {
	// This should pass
	NotEquals(t, 42, 43)
	NotEquals(t, "hello", "world")
	NotEquals(t, true, false)
}

func TestAssertTrue(t *testing.T) {
	True(t, true, "Should be true")
	True(t, 1 == 1, "1 should equal 1")
}

func TestAssertFalse(t *testing.T) {
	False(t, false, "Should be false")
	False(t, 1 == 2, "1 should not equal 2")
}

func TestAssertNil(t *testing.T) {
	var nilPtr *int
	Nil(t, nil, "nil should be nil")
	Nil(t, nilPtr, "nil pointer should be nil")
}

func TestAssertNotNil(t *testing.T) {
	value := 42
	NotNil(t, &value, "pointer should not be nil")
	NotNil(t, "string", "string should not be nil")
}

func TestAssertError(t *testing.T) {
	err := errors.New("test error")
	Error(t, err, "Should have an error")
}

func TestAssertNoError(t *testing.T) {
	NoError(t, nil)
}

func TestAssertStringContains(t *testing.T) {
	Contains(t, "hello world", "world")
	Contains(t, "package main", "main")
}

func TestAssertLen(t *testing.T) {
	Length(t, "hello", 5)
	Length(t, []int{1, 2, 3}, 3)
}

package testutils

import (
	"errors"
	"testing"
)

func TestAssertEquals(t *testing.T) {
	// This should pass
	AssertEquals(t, 42, 42)
	AssertEquals(t, "hello", "hello")
	AssertEquals(t, true, true)
}

func TestAssertNotEquals(t *testing.T) {
	// This should pass
	AssertNotEquals(t, 42, 43)
	AssertNotEquals(t, "hello", "world")
	AssertNotEquals(t, true, false)
}

func TestAssertTrue(t *testing.T) {
	AssertTrue(t, true, "Should be true")
	AssertTrue(t, 1 == 1, "1 should equal 1")
}

func TestAssertFalse(t *testing.T) {
	AssertFalse(t, false, "Should be false")
	AssertFalse(t, 1 == 2, "1 should not equal 2")
}

func TestAssertNil(t *testing.T) {
	var nilPtr *int
	AssertNil(t, nil, "nil should be nil")
	AssertNil(t, nilPtr, "nil pointer should be nil")
}

func TestAssertNotNil(t *testing.T) {
	value := 42
	AssertNotNil(t, &value, "pointer should not be nil")
	AssertNotNil(t, "string", "string should not be nil")
}

func TestAssertError(t *testing.T) {
	err := errors.New("test error")
	AssertError(t, err, "Should have an error")
}

func TestAssertNoError(t *testing.T) {
	AssertNoError(t, nil)
}

func TestAssertStringContains(t *testing.T) {
	AssertStringContains(t, "hello world", "world")
	AssertStringContains(t, "package main", "main")
}

func TestAssertLen(t *testing.T) {
	AssertLen(t, "hello", 5)
	AssertLen(t, []int{1, 2, 3}, 3)
}

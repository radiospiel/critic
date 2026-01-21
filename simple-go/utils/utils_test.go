package utils

import (
	"testing"

	"git.15b.it/eno/critic/simple-go/assert"
)

func TestReverseEmpty(t *testing.T) {
	s := []int{}
	Reverse(s)
	assert.Equals(t, len(s), 0, "empty slice should remain empty")
}

func TestReverseSingleElement(t *testing.T) {
	s := []int{42}
	Reverse(s)
	assert.Equals(t, s[0], 42, "single element should remain unchanged")
}

func TestReverseTwoElements(t *testing.T) {
	s := []int{1, 2}
	Reverse(s)
	assert.Equals(t, s[0], 2, "first element should be 2")
	assert.Equals(t, s[1], 1, "second element should be 1")
}

func TestReverseOddLength(t *testing.T) {
	s := []int{1, 2, 3, 4, 5}
	Reverse(s)
	expected := []int{5, 4, 3, 2, 1}
	for i, v := range expected {
		assert.Equals(t, s[i], v, "element %d should be %d", i, v)
	}
}

func TestReverseEvenLength(t *testing.T) {
	s := []int{1, 2, 3, 4}
	Reverse(s)
	expected := []int{4, 3, 2, 1}
	for i, v := range expected {
		assert.Equals(t, s[i], v, "element %d should be %d", i, v)
	}
}

func TestReverseStrings(t *testing.T) {
	s := []string{"a", "b", "c"}
	Reverse(s)
	assert.Equals(t, s[0], "c", "first element should be 'c'")
	assert.Equals(t, s[1], "b", "second element should be 'b'")
	assert.Equals(t, s[2], "a", "third element should be 'a'")
}

func TestReverseInPlace(t *testing.T) {
	original := []int{1, 2, 3}
	s := original
	Reverse(s)
	// Verify it modified the original slice (same backing array)
	assert.Equals(t, original[0], 3, "original slice should be modified in place")
}

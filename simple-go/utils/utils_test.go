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

func TestMemoize1ReturnsCorrectValue(t *testing.T) {
	double := func(x int) int { return x * 2 }
	memoized := Memoize1(double)

	assert.Equals(t, memoized(5), 10, "memoized(5) should return 10")
	assert.Equals(t, memoized(3), 6, "memoized(3) should return 6")
}

func TestMemoize1CachesResults(t *testing.T) {
	callCount := 0
	expensive := func(x int) int {
		callCount++
		return x * x
	}
	memoized := Memoize1(expensive)

	// First call should invoke the function
	result1 := memoized(4)
	assert.Equals(t, result1, 16, "first call should return 16")
	assert.Equals(t, callCount, 1, "function should be called once")

	// Second call with same argument should use cache
	result2 := memoized(4)
	assert.Equals(t, result2, 16, "cached call should return 16")
	assert.Equals(t, callCount, 1, "function should still be called only once")

	// Call with different argument should invoke function again
	result3 := memoized(5)
	assert.Equals(t, result3, 25, "new argument should return 25")
	assert.Equals(t, callCount, 2, "function should be called twice total")
}

func TestMemoize1WithStrings(t *testing.T) {
	toUpper := func(s string) string { return s + "!" }
	memoized := Memoize1(toUpper)

	assert.Equals(t, memoized("hello"), "hello!", "should append exclamation")
	assert.Equals(t, memoized("world"), "world!", "should append exclamation")
}

func TestMemoize1LRUEviction(t *testing.T) {
	callCount := 0
	fn := func(x int) int {
		callCount++
		return x
	}
	memoized := Memoize1(fn)

	// Fill cache beyond limit (256)
	for i := 0; i < 260; i++ {
		memoized(i)
	}
	assert.Equals(t, callCount, 260, "should have called function 260 times")

	// Access early entries - they should have been evicted
	memoized(0)
	assert.Equals(t, callCount, 261, "entry 0 should have been evicted and recomputed")

	// Access recent entry - should still be cached
	memoized(259)
	assert.Equals(t, callCount, 261, "entry 259 should still be cached")
}

func TestMemoize2ReturnsCorrectValue(t *testing.T) {
	add := func(a, b int) int { return a + b }
	memoized := Memoize2(add)

	assert.Equals(t, memoized(2, 3), 5, "memoized(2, 3) should return 5")
	assert.Equals(t, memoized(10, 20), 30, "memoized(10, 20) should return 30")
}

func TestMemoize2CachesResults(t *testing.T) {
	callCount := 0
	multiply := func(a, b int) int {
		callCount++
		return a * b
	}
	memoized := Memoize2(multiply)

	// First call
	result1 := memoized(3, 4)
	assert.Equals(t, result1, 12, "first call should return 12")
	assert.Equals(t, callCount, 1, "function should be called once")

	// Same arguments - should use cache
	result2 := memoized(3, 4)
	assert.Equals(t, result2, 12, "cached call should return 12")
	assert.Equals(t, callCount, 1, "function should still be called only once")

	// Different arguments
	result3 := memoized(4, 3)
	assert.Equals(t, result3, 12, "different args should return 12")
	assert.Equals(t, callCount, 2, "function should be called twice total")

	// Original args still cached
	memoized(3, 4)
	assert.Equals(t, callCount, 2, "original args should still be cached")
}

func TestMemoize2WithStrings(t *testing.T) {
	concat := func(a, b string) string { return a + b }
	memoized := Memoize2(concat)

	assert.Equals(t, memoized("hello", "world"), "helloworld", "should concatenate")
	assert.Equals(t, memoized("foo", "bar"), "foobar", "should concatenate")
}

func TestMemoize2LRUEviction(t *testing.T) {
	callCount := 0
	fn := func(a, b int) int {
		callCount++
		return a + b
	}
	memoized := Memoize2(fn)

	// Fill cache beyond limit (256)
	for i := 0; i < 260; i++ {
		memoized(i, 0)
	}
	assert.Equals(t, callCount, 260, "should have called function 260 times")

	// Access early entry - should have been evicted
	memoized(0, 0)
	assert.Equals(t, callCount, 261, "entry (0,0) should have been evicted and recomputed")

	// Access recent entry - should still be cached
	memoized(259, 0)
	assert.Equals(t, callCount, 261, "entry (259,0) should still be cached")
}

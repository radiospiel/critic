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

func TestLRUCacheGetSet(t *testing.T) {
	cache := NewLRUCache[string, int](10)

	// Set and get a value
	cache.Set("a", 1)
	val, ok := cache.Get("a")
	assert.Equals(t, ok, true, "key 'a' should exist")
	assert.Equals(t, val, 1, "value should be 1")

	// Get non-existent key
	_, ok = cache.Get("b")
	assert.Equals(t, ok, false, "key 'b' should not exist")
}

func TestLRUCacheUpdate(t *testing.T) {
	cache := NewLRUCache[string, int](10)

	cache.Set("a", 1)
	cache.Set("a", 2) // Update existing key

	val, ok := cache.Get("a")
	assert.Equals(t, ok, true, "key 'a' should exist")
	assert.Equals(t, val, 2, "value should be updated to 2")
}

func TestLRUCacheEviction(t *testing.T) {
	cache := NewLRUCache[int, int](3)

	// Fill cache
	cache.Set(1, 10)
	cache.Set(2, 20)
	cache.Set(3, 30)

	// Add one more, should evict oldest (1)
	cache.Set(4, 40)

	_, ok := cache.Get(1)
	assert.Equals(t, ok, false, "key 1 should have been evicted")

	val, ok := cache.Get(2)
	assert.Equals(t, ok, true, "key 2 should still exist")
	assert.Equals(t, val, 20, "value should be 20")
}

func TestLRUCacheLRUOrder(t *testing.T) {
	cache := NewLRUCache[int, int](3)

	cache.Set(1, 10)
	cache.Set(2, 20)
	cache.Set(3, 30)

	// Access key 1, making it most recently used
	cache.Get(1)

	// Add new key, should evict key 2 (now oldest)
	cache.Set(4, 40)

	_, ok := cache.Get(2)
	assert.Equals(t, ok, false, "key 2 should have been evicted")

	_, ok = cache.Get(1)
	assert.Equals(t, ok, true, "key 1 should still exist (was accessed)")
}

func TestLRUCacheWithStructKey(t *testing.T) {
	type key struct {
		a string
		b int
	}
	cache := NewLRUCache[key, string](10)

	k1 := key{"hello", 1}
	k2 := key{"world", 2}

	cache.Set(k1, "value1")
	cache.Set(k2, "value2")

	val, ok := cache.Get(k1)
	assert.Equals(t, ok, true, "key k1 should exist")
	assert.Equals(t, val, "value1", "value should be 'value1'")

	val, ok = cache.Get(k2)
	assert.Equals(t, ok, true, "key k2 should exist")
	assert.Equals(t, val, "value2", "value should be 'value2'")
}

func TestLRUCacheDefaultLimit(t *testing.T) {
	assert.Equals(t, LRUCacheDefaultLimit, 256, "default limit should be 256")
}

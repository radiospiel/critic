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

func TestLRUCacheCreatesValue(t *testing.T) {
	cache := NewLRUCache(10, func(key string) int {
		return len(key)
	})

	assert.Equals(t, cache.Get("a"), 1, "should create value for 'a'")
	assert.Equals(t, cache.Get("hello"), 5, "should create value for 'hello'")
}

func TestLRUCacheCachesResults(t *testing.T) {
	callCount := 0
	cache := NewLRUCache(10, func(key int) int {
		callCount++
		return key * 2
	})

	// First call creates value
	assert.Equals(t, cache.Get(5), 10, "should return 10")
	assert.Equals(t, callCount, 1, "creator should be called once")

	// Second call uses cache
	assert.Equals(t, cache.Get(5), 10, "should return cached 10")
	assert.Equals(t, callCount, 1, "creator should still be called only once")

	// Different key creates new value
	assert.Equals(t, cache.Get(3), 6, "should return 6")
	assert.Equals(t, callCount, 2, "creator should be called twice")
}

func TestLRUCacheEviction(t *testing.T) {
	callCount := 0
	cache := NewLRUCache(3, func(key int) int {
		callCount++
		return key * 10
	})

	// Fill cache
	cache.Get(1)
	cache.Get(2)
	cache.Get(3)
	assert.Equals(t, callCount, 3, "should have called creator 3 times")

	// Add one more, should evict oldest (1)
	cache.Get(4)
	assert.Equals(t, callCount, 4, "should have called creator 4 times")

	// Access evicted key - should recreate
	cache.Get(1)
	assert.Equals(t, callCount, 5, "key 1 should have been evicted and recreated")

	// Access cached key - should not call creator
	cache.Get(4)
	assert.Equals(t, callCount, 5, "key 4 should still be cached")
}

func TestLRUCacheLRUOrder(t *testing.T) {
	callCount := 0
	cache := NewLRUCache(3, func(key int) int {
		callCount++
		return key
	})

	cache.Get(1)
	cache.Get(2)
	cache.Get(3)

	// Access key 1, making it most recently used
	cache.Get(1)
	assert.Equals(t, callCount, 3, "no new creation for cached key")

	// Add new key, should evict key 2 (now oldest)
	cache.Get(4)
	assert.Equals(t, callCount, 4, "should create key 4")

	// Key 2 should have been evicted
	cache.Get(2)
	assert.Equals(t, callCount, 5, "key 2 should have been evicted and recreated")

	// Key 1 should still be cached
	cache.Get(1)
	assert.Equals(t, callCount, 5, "key 1 should still be cached")
}

func TestLRUCacheWithStructKey(t *testing.T) {
	type key struct {
		a string
		b int
	}
	cache := NewLRUCache(10, func(k key) string {
		return k.a + "!"
	})

	k1 := key{"hello", 1}
	k2 := key{"world", 2}

	assert.Equals(t, cache.Get(k1), "hello!", "should create value for k1")
	assert.Equals(t, cache.Get(k2), "world!", "should create value for k2")
}

func TestLRUCacheDefaultLimit(t *testing.T) {
	assert.Equals(t, LRUCacheDefaultLimit, 256, "default limit should be 256")
}

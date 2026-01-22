package utils

import (
	"cmp"
	"slices"

	"git.15b.it/eno/critic/simple-go/preconditions"
)

// Reverse reverses a slice in place
func Reverse[S ~[]E, E any](s S) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

// SortBy returns a sorted copy of the slice, sorted by the key returned by the iteratee.
func SortBy[T any, K cmp.Ordered](collection []T, iteratee func(T) K) []T {
	result := make([]T, len(collection))
	copy(result, collection)
	slices.SortFunc(result, func(a, b T) int {
		return cmp.Compare(iteratee(a), iteratee(b))
	})
	return result
}

// Partition splits a slice into two slices based on a predicate.
// The first slice contains elements for which the predicate returns true,
// the second contains elements for which it returns false.
func Partition[T any](collection []T, predicate func(T) bool) ([]T, []T) {
	var matched, unmatched []T
	for _, item := range collection {
		if predicate(item) {
			matched = append(matched, item)
		} else {
			unmatched = append(unmatched, item)
		}
	}
	return matched, unmatched
}

// IfElse returns ifTrue if condition is true, otherwise returns ifFalse.
func IfElse[T any](condition bool, ifTrue T, ifFalse T) T {
	if condition {
		return ifTrue
	}
	return ifFalse
}

// Clamp constrains a value to be within the range [minVal, maxVal].
func Clamp[T cmp.Ordered](value, minVal, maxVal T) T {
	preconditions.Check(minVal <= maxVal, "Clamp: minVal (%v) must be <= maxVal (%v)", minVal, maxVal)
	if value < minVal {
		return minVal
	}
	if value > maxVal {
		return maxVal
	}
	return value
}

const lruCacheDefaultLimit = 256

// lruCache is a simple LRU cache using a map and slice.
type lruCache[K comparable, V any] struct {
	data  map[K]V
	order []K
	limit int
}

// newLRUCache creates a new LRU cache with the specified limit.
func newLRUCache[K comparable, V any](limit int) *lruCache[K, V] {
	return &lruCache[K, V]{
		data:  make(map[K]V),
		order: make([]K, 0, limit),
		limit: limit,
	}
}

// Get retrieves a value from the cache, moving it to most recently used.
func (c *lruCache[K, V]) Get(key K) (V, bool) {
	value, ok := c.data[key]
	if ok {
		// Move to end (most recently used)
		for i, k := range c.order {
			if k == key {
				c.order = append(c.order[:i], c.order[i+1:]...)
				c.order = append(c.order, key)
				break
			}
		}
	}
	return value, ok
}

// Set adds or updates a value in the cache, evicting the oldest if at capacity.
func (c *lruCache[K, V]) Set(key K, value V) {
	if _, exists := c.data[key]; exists {
		c.data[key] = value
		return
	}

	// Evict oldest if at capacity
	if len(c.order) >= c.limit {
		oldest := c.order[0]
		c.order = c.order[1:]
		delete(c.data, oldest)
	}

	c.data[key] = value
	c.order = append(c.order, key)
}

// Memoize1 returns a memoized version of a single-argument function.
// The returned function caches results based on the argument value.
// Uses LRU eviction with a cache limit of 256 entries.
func Memoize1[A comparable, R any](fn func(A) R) func(A) R {
	cache := newLRUCache[A, R](lruCacheDefaultLimit)

	return func(arg A) R {
		if result, ok := cache.Get(arg); ok {
			return result
		}
		result := fn(arg)
		cache.Set(arg, result)
		return result
	}
}

// Memoize2 returns a memoized version of a two-argument function.
// The returned function caches results based on both argument values.
// Uses LRU eviction with a cache limit of 256 entries.
func Memoize2[A, B comparable, R any](fn func(A, B) R) func(A, B) R {
	type key struct {
		a A
		b B
	}
	cache := newLRUCache[key, R](lruCacheDefaultLimit)

	return func(a A, b B) R {
		k := key{a, b}
		if result, ok := cache.Get(k); ok {
			return result
		}
		result := fn(a, b)
		cache.Set(k, result)
		return result
	}
}

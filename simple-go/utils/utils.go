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

const memoizeCacheLimit = 256

// Memoize1 returns a memoized version of a single-argument function.
// The returned function caches results based on the argument value.
// Uses LRU eviction with a cache limit of 256 entries.
func Memoize1[A comparable, R any](fn func(A) R) func(A) R {
	cache := make(map[A]R)
	order := make([]A, 0, memoizeCacheLimit)

	return func(arg A) R {
		if result, ok := cache[arg]; ok {
			// Move to end (most recently used)
			for i, k := range order {
				if k == arg {
					order = append(order[:i], order[i+1:]...)
					order = append(order, arg)
					break
				}
			}
			return result
		}

		// Evict oldest if at capacity
		if len(order) >= memoizeCacheLimit {
			oldest := order[0]
			order = order[1:]
			delete(cache, oldest)
		}

		result := fn(arg)
		cache[arg] = result
		order = append(order, arg)
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
	cache := make(map[key]R)
	order := make([]key, 0, memoizeCacheLimit)

	return func(a A, b B) R {
		k := key{a, b}
		if result, ok := cache[k]; ok {
			// Move to end (most recently used)
			for i, stored := range order {
				if stored == k {
					order = append(order[:i], order[i+1:]...)
					order = append(order, k)
					break
				}
			}
			return result
		}

		// Evict oldest if at capacity
		if len(order) >= memoizeCacheLimit {
			oldest := order[0]
			order = order[1:]
			delete(cache, oldest)
		}

		result := fn(a, b)
		cache[k] = result
		order = append(order, k)
		return result
	}
}

// Package observable provides a data wrapper for maps and lists with path-based
// access and change subscriptions.
package observable

import (
	"path"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"git.15b.it/eno/critic/simple-go/preconditions"
)

// maxArrayIndex is the maximum allowed array index to prevent accidental huge allocations.
const maxArrayIndex = 99999

// Subscription represents a registered observer subscription.
type Subscription int

// ChangeCallback is the function signature for change notifications.
// It receives the full key path, the old value, and the new value.
type ChangeCallback func(key string, oldValue, newValue any)

type subscription struct {
	patterns []string
	callback ChangeCallback
}

// Observable wraps maps and lists with path-based access and change subscriptions.
type Observable struct {
	data          any
	subscriptions map[Subscription]*subscription
	nextSubID     Subscription
	mu            sync.RWMutex
}

// New creates a new Observable with an empty map as the root.
func New() *Observable {
	return &Observable{
		data:          make(map[string]any),
		subscriptions: make(map[Subscription]*subscription),
	}
}

// NewWithData creates a new Observable with the provided data as the root.
// The data must be nil, a map[string]any, or []any.
func NewWithData(data any) *Observable {
	if data != nil {
		preconditions.Check(
			isMap(data) || isSlice(data),
			"root data must be nil, map[string]any, or []any, got %T", data,
		)
	}
	return &Observable{
		data:          data,
		subscriptions: make(map[Subscription]*subscription),
	}
}

// GetValue returns the value at the given key path, or nil if not found.
// Key is a dot-separated path like "x.1.a".
func (o *Observable) GetValue(key string) any {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return o.getValueInternal(key)
}

// GetValueAs returns the value at the given key path, cast to type T.
// Returns the zero value and false if not found or if the type assertion fails.
// Usage: val, ok := observable.GetValueAs[string](obs, "config.name")
func GetValueAs[T any](o *Observable, key string) (T, bool) {
	val := o.GetValue(key)
	if val == nil {
		var zero T
		return zero, false
	}
	typed, ok := val.(T)
	return typed, ok
}

// MustGetValueAs returns the value at the given key path, cast to type T.
// Panics if not found or if the type assertion fails.
// Usage: val := observable.MustGetValueAs[string](obs, "config.name")
func MustGetValueAs[T any](o *Observable, key string) T {
	val, ok := GetValueAs[T](o, key)
	preconditions.Check(ok, "value at key %q is not of type %T or does not exist", key, val)
	return val
}

// GetMap returns the value at the given key path as a map[string]any.
// Returns nil and false if not found or if the value is not a map.
func (o *Observable) GetMap(key string) (map[string]any, bool) {
	return GetValueAs[map[string]any](o, key)
}

// GetSlice returns the value at the given key path as a []any.
// Returns nil and false if not found or if the value is not a slice.
func (o *Observable) GetSlice(key string) ([]any, bool) {
	return GetValueAs[[]any](o, key)
}

// GetString returns the value at the given key path as a string.
// Returns empty string and false if not found or if the value is not a string.
func (o *Observable) GetString(key string) (string, bool) {
	return GetValueAs[string](o, key)
}

// GetInt returns the value at the given key path as an int.
// Returns 0 and false if not found or if the value is not an int.
func (o *Observable) GetInt(key string) (int, bool) {
	return GetValueAs[int](o, key)
}

// GetFloat64 returns the value at the given key path as a float64.
// Returns 0 and false if not found or if the value is not a float64.
func (o *Observable) GetFloat64(key string) (float64, bool) {
	return GetValueAs[float64](o, key)
}

// GetBool returns the value at the given key path as a bool.
// Returns false and false if not found or if the value is not a bool.
func (o *Observable) GetBool(key string) (bool, bool) {
	return GetValueAs[bool](o, key)
}

// getValueInternal returns the value without locking (caller must hold lock).
func (o *Observable) getValueInternal(key string) any {
	if key == "" {
		return o.data
	}

	parts := strings.Split(key, ".")
	current := o.data

	for _, part := range parts {
		if current == nil {
			return nil
		}

		if idx, isNum := parseIndex(part); isNum {
			// Numeric index - expect slice
			slice, ok := current.([]any)
			if !ok {
				return nil
			}
			if idx < 0 || idx >= len(slice) {
				return nil
			}
			current = slice[idx]
		} else {
			// String key - expect map
			m, ok := current.(map[string]any)
			if !ok {
				return nil
			}
			current = m[part]
		}
	}

	return current
}

// SetValueAtKey sets the value at the given key path, creating intermediate
// structures as needed.
//
// Key is a dot-separated path like "x.1.a".
// Numeric path segments indicate array indices, string segments indicate map keys.
//
// Panics if:
//   - The existing value at a path segment has an incompatible type
//   - An array index is negative or >= 100000
func (o *Observable) SetValueAtKey(key string, value any) {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Collect all changes for notification
	changes := make(map[string]struct{ old, new any })

	// Get old value at the target key
	oldValue := o.getValueInternal(key)

	// Perform the set
	o.setValueInternal(key, value)

	// Record the direct change
	newValue := o.getValueInternal(key)
	if !reflect.DeepEqual(oldValue, newValue) {
		changes[key] = struct{ old, new any }{oldValue, newValue}
	}

	// If setting a nested value (map or slice), also record deeper changes
	o.collectNestedChanges(key, oldValue, newValue, changes)

	// Notify subscribers
	o.notifySubscribers(changes)
}

// setValueInternal sets the value without locking (caller must hold lock).
func (o *Observable) setValueInternal(key string, value any) {
	if key == "" {
		// Setting the root
		if value != nil {
			preconditions.Check(
				isMap(value) || isSlice(value),
				"root value must be nil, map, or slice, got %T", value,
			)
		}
		o.data = value
		return
	}

	parts := strings.Split(key, ".")

	// Ensure root is a map (since first part is a string key in a path like "x.1.a")
	if o.data == nil {
		o.data = make(map[string]any)
	}
	preconditions.Check(isMap(o.data), "root must be a map when setting path %q, got %T", key, o.data)

	// Use the recursive setter that returns the modified value
	o.data = o.setAtPath(o.data, parts, value)
}

// setAtPath recursively sets a value at the given path parts and returns the modified container.
func (o *Observable) setAtPath(current any, parts []string, value any) any {
	if len(parts) == 0 {
		return value
	}

	part := parts[0]
	isLastPart := len(parts) == 1

	if idx, isNum := parseIndex(part); isNum {
		// Current part is a numeric index - current must be a slice
		preconditions.Check(isSlice(current), "expected slice at index %d, got %T", idx, current)
		preconditions.Check(idx >= 0 && idx <= maxArrayIndex, "array index must be 0-%d, got %d", maxArrayIndex, idx)

		slice := current.([]any)

		// Extend slice if needed
		for len(slice) <= idx {
			slice = append(slice, nil)
		}

		if isLastPart {
			slice[idx] = value
		} else {
			// Determine what type the next level should be
			nextPart := parts[1]
			_, nextIsNum := parseIndex(nextPart)

			if slice[idx] == nil {
				if nextIsNum {
					slice[idx] = make([]any, 0)
				} else {
					slice[idx] = make(map[string]any)
				}
			} else {
				// Validate existing type
				if nextIsNum {
					preconditions.Check(isSlice(slice[idx]), "expected slice at %q, got %T", part, slice[idx])
				} else {
					preconditions.Check(isMap(slice[idx]), "expected map at %q, got %T", part, slice[idx])
				}
			}

			slice[idx] = o.setAtPath(slice[idx], parts[1:], value)
		}

		return slice
	}

	// Current part is a string key - current must be a map
	preconditions.Check(isMap(current), "expected map at key %q, got %T", part, current)

	m := current.(map[string]any)

	if isLastPart {
		if value == nil {
			delete(m, part)
		} else {
			m[part] = value
		}
	} else {
		// Determine what type the next level should be
		nextPart := parts[1]
		_, nextIsNum := parseIndex(nextPart)

		if m[part] == nil {
			if nextIsNum {
				m[part] = make([]any, 0)
			} else {
				m[part] = make(map[string]any)
			}
		} else {
			// Validate existing type
			if nextIsNum {
				preconditions.Check(isSlice(m[part]), "expected slice at %q, got %T", part, m[part])
			} else {
				preconditions.Check(isMap(m[part]), "expected map at %q, got %T", part, m[part])
			}
		}

		m[part] = o.setAtPath(m[part], parts[1:], value)
	}

	return m
}

// DeleteValueAtKey removes the value at the given key path.
// This is equivalent to SetValueAtKey(key, nil).
func (o *Observable) DeleteValueAtKey(key string) {
	o.SetValueAtKey(key, nil)
}

// OnKeyChange registers a callback to be notified when values at matching paths change.
// Patterns use fnmatch-style matching (using path.Match).
// Returns the subscriptions created (one per pattern) for later cleanup.
func (o *Observable) OnKeyChange(patterns []string, callback ChangeCallback) []Subscription {
	o.mu.Lock()
	defer o.mu.Unlock()

	preconditions.Check(len(patterns) > 0, "at least one pattern required")
	preconditions.Check(callback != nil, "callback must not be nil")

	id := o.nextSubID
	o.nextSubID++

	o.subscriptions[id] = &subscription{
		patterns: patterns,
		callback: callback,
	}

	return []Subscription{id}
}

// ClearSubscriptions removes the specified subscriptions.
func (o *Observable) ClearSubscriptions(subs ...Subscription) {
	o.mu.Lock()
	defer o.mu.Unlock()

	for _, sub := range subs {
		delete(o.subscriptions, sub)
	}
}

// notifySubscribers notifies all matching subscribers of the changes.
// Each subscription is triggered at most once per SetValueAtKey call.
func (o *Observable) notifySubscribers(changes map[string]struct{ old, new any }) {
	if len(changes) == 0 {
		return
	}

	// For each subscription, check if any of its patterns match any changed key
	for _, sub := range o.subscriptions {
		var matchedKey string
		var matchedChange struct{ old, new any }
		matched := false

		for key, change := range changes {
			for _, pattern := range sub.patterns {
				if matchPattern(pattern, key) {
					if !matched {
						matchedKey = key
						matchedChange = change
						matched = true
					}
					break
				}
			}
			if matched {
				break
			}
		}

		if matched {
			sub.callback(matchedKey, matchedChange.old, matchedChange.new)
		}
	}
}

// collectNestedChanges adds changes for nested keys when setting a map or slice.
func (o *Observable) collectNestedChanges(prefix string, oldVal, newVal any, changes map[string]struct{ old, new any }) {
	// Get all keys from both old and new values
	oldKeys := collectKeys(oldVal, prefix)
	newKeys := collectKeys(newVal, prefix)

	// Union of all keys
	allKeys := make(map[string]bool)
	for k := range oldKeys {
		allKeys[k] = true
	}
	for k := range newKeys {
		allKeys[k] = true
	}

	// Check each nested key for changes
	for key := range allKeys {
		if key == prefix {
			continue // Already handled
		}
		oldNested := getNestedValue(oldVal, strings.TrimPrefix(key, prefix+"."))
		newNested := getNestedValue(newVal, strings.TrimPrefix(key, prefix+"."))
		if !reflect.DeepEqual(oldNested, newNested) {
			changes[key] = struct{ old, new any }{oldNested, newNested}
		}
	}
}

// collectKeys returns all leaf keys from a value with the given prefix.
func collectKeys(val any, prefix string) map[string]bool {
	keys := make(map[string]bool)

	if val == nil {
		return keys
	}

	switch v := val.(type) {
	case map[string]any:
		for k, child := range v {
			childPrefix := prefix + "." + k
			keys[childPrefix] = true
			for nested := range collectKeys(child, childPrefix) {
				keys[nested] = true
			}
		}
	case []any:
		for i, child := range v {
			childPrefix := prefix + "." + strconv.Itoa(i)
			keys[childPrefix] = true
			for nested := range collectKeys(child, childPrefix) {
				keys[nested] = true
			}
		}
	}

	return keys
}

// getNestedValue gets a value from a nested structure by path.
func getNestedValue(val any, keyPath string) any {
	if keyPath == "" {
		return val
	}

	parts := strings.Split(keyPath, ".")
	current := val

	for _, part := range parts {
		if current == nil {
			return nil
		}

		if idx, isNum := parseIndex(part); isNum {
			slice, ok := current.([]any)
			if !ok || idx < 0 || idx >= len(slice) {
				return nil
			}
			current = slice[idx]
		} else {
			m, ok := current.(map[string]any)
			if !ok {
				return nil
			}
			current = m[part]
		}
	}

	return current
}

// matchPattern checks if a key matches a pattern using fnmatch-style matching.
func matchPattern(pattern, key string) bool {
	matched, err := path.Match(pattern, key)
	if err != nil {
		return false
	}
	return matched
}

// parseIndex tries to parse a string as a non-negative integer.
// Returns the index and true if successful, or 0 and false otherwise.
func parseIndex(s string) (int, bool) {
	idx, err := strconv.Atoi(s)
	if err != nil || idx < 0 {
		return 0, false
	}
	return idx, true
}

// isMap checks if a value is a map[string]any.
func isMap(v any) bool {
	_, ok := v.(map[string]any)
	return ok
}

// isSlice checks if a value is a []any.
func isSlice(v any) bool {
	_, ok := v.([]any)
	return ok
}

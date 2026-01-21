// Package observable provides a data wrapper for maps and lists with path-based
// access and change subscriptions.
package observable

import (
	"encoding/json"
	"path"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"git.15b.it/eno/critic/simple-go/preconditions"
	"git.15b.it/eno/critic/simple-go/utils"
)

// maxArrayIndex is the maximum allowed array index to prevent accidental huge allocations.
const maxArrayIndex = 99999

// Subscription represents a registered observer subscription.
type Subscription int

// ChangeCallback is the function signature for change notifications.
// It receives the observable and the full key path that changed.
// The callback can read from the observable to get the current value.
type ChangeCallback func(obs *Observable, key string)

type subscription struct {
	pattern  string
	callback ChangeCallback
}

// Observable wraps maps and lists with path-based access and change subscriptions.
type Observable struct {
	data          any
	subscriptions map[Subscription]*subscription
	nextSubID     Subscription
	schemas       map[string]*schemaEntry
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

// GetValueAs returns the value at the given key path, converted to type T.
// Returns the zero value if the key does not exist.
// For primitive types, performs a direct type assertion.
// For structs, uses JSON marshaling to convert from map[string]any.
// The returned value is always a copy.
// Panics if the conversion fails.
// Usage: val := observable.GetValueAs[MyStruct](obs, "config")
func GetValueAs[T any](o *Observable, key string) T {
	val := o.GetValue(key)
	if val == nil {
		var zero T
		return zero
	}

	// Try direct type assertion first (works for primitives and exact type matches)
	if typed, ok := val.(T); ok {
		return typed
	}

	// Use JSON round-trip for struct conversion (provides deep copy)
	bytes, err := json.Marshal(val)
	preconditions.Check(err == nil, "failed to marshal value at key %q: %v", key, err)

	var result T
	err = json.Unmarshal(bytes, &result)
	preconditions.Check(err == nil, "failed to unmarshal value at key %q to %T: %v", key, result, err)

	return result
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
	o.Transaction(func(tx *Txn) {
		tx.SetValueAtKey(key, value)
	})
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
	o.Transaction(func(tx *Txn) {
		tx.DeleteValueAtKey(key)
	})
}

// OnKeyChange registers a callback to be notified when values at the matching path change.
// Pattern uses fnmatch-style matching (using path.Match).
// Returns the subscription ID for later cleanup.
func (o *Observable) OnKeyChange(pattern string, callback ChangeCallback) Subscription {
	o.mu.Lock()
	defer o.mu.Unlock()

	preconditions.Check(callback != nil, "callback must not be nil")

	id := o.nextSubID
	o.nextSubID++

	o.subscriptions[id] = &subscription{
		pattern:  pattern,
		callback: callback,
	}

	return id
}

// ClearSubscriptions removes the specified subscriptions.
func (o *Observable) ClearSubscriptions(subs ...Subscription) {
	o.mu.Lock()
	defer o.mu.Unlock()

	for _, sub := range subs {
		delete(o.subscriptions, sub)
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

// change represents a single key-value change in a transaction.
type change struct {
	key   string
	value any
}

// Txn represents a transaction that batches changes before applying them.
// Transactions are created via Observable.Transaction() and are
// automatically committed when the callback returns, unless Abort() is called.
type Txn struct {
	obs     *Observable
	changes []change
	aborted bool
}

// SetValueAtKey records a change to be applied when the transaction commits.
// If the transaction has been aborted, the call is ignored.
func (tx *Txn) SetValueAtKey(key string, value any) {
	if tx.aborted {
		return
	}
	tx.changes = append(tx.changes, change{key: key, value: value})
}

// DeleteValueAtKey records a deletion to be applied when the transaction commits.
func (tx *Txn) DeleteValueAtKey(key string) {
	tx.SetValueAtKey(key, nil)
}

// Abort cancels the transaction. All recorded changes will be discarded
// and no notifications will be sent. Subsequent SetValueAtKey calls are ignored.
func (tx *Txn) Abort() {
	tx.aborted = true
	tx.changes = nil
}

// deduplicateChanges removes changes that are overridden by later changes.
// A change is overridden if a later change sets a parent key (or the same key).
func deduplicateChanges(changes []change) []change {
	// Work backwards: for each change, check if any later change overrides it
	result := make([]change, 0, len(changes))

	for i := len(changes) - 1; i >= 0; i-- {
		current := changes[i]
		overridden := false

		// Check if any change already in result (which are later changes) overrides this one
		for _, later := range result {
			if keyOverrides(later.key, current.key) {
				overridden = true
				break
			}
		}

		if !overridden {
			result = append(result, current)
		}
	}

	// Reverse to restore original order (we built it backwards)
	utils.Reverse(result)

	return result
}

// keyOverrides returns true if setting 'parent' would override a change to 'child'.
// This is true if parent is a prefix of child (or equal).
// Examples:
//   - keyOverrides("a", "a.1.b") = true  (setting "a" overwrites "a.1.b")
//   - keyOverrides("a", "a") = true      (same key)
//   - keyOverrides("a.1", "a") = false   (setting "a.1" doesn't overwrite "a")
//   - keyOverrides("", "a") = true       (setting root overwrites everything)
func keyOverrides(parent, child string) bool {
	if parent == child {
		return true
	}
	if parent == "" {
		return true // Root overrides everything
	}
	// parent must be a prefix followed by "."
	return strings.HasPrefix(child, parent+".")
}

// Txn executes a transaction. The callback receives a Txn object to record changes.
// Changes are automatically committed when the callback returns, unless Abort() is called.
// Example:
//
//	obs.Transaction(func(tx *Transaction) {
//	    tx.SetValueAtKey("foo", "bar")
//	    tx.SetValueAtKey("baz", 123)
//	})
func (o *Observable) Transaction(fn func(*Txn)) {
	tx := &Txn{
		obs:     o,
		changes: make([]change, 0),
	}
	fn(tx)

	if tx.aborted {
		return
	}

	// Remove changes that are overridden by later change on the same or a parent key
	// For example: ["a.1.b", "a", "a.2", "c", "a.1", "a"] becomes ["c", "a"]
	deduplicatedChanges := deduplicateChanges(tx.changes)

	tx.obs.setValuesAtKeys(deduplicatedChanges)
}

// setValuesAtKeys applies multiple changes atomically and notifies subscribers.
func (o *Observable) setValuesAtKeys(changes []change) {
	if len(changes) == 0 {
		return
	}

	o.mu.Lock()

	// Collect old values for all keys that will be affected
	oldValues := make(map[string]any)
	for _, c := range changes {
		// Get old value at the key being set
		oldValues[c.key] = o.getValueInternal(c.key)

		// Also collect old values for nested keys that will be overwritten
		oldValue := oldValues[c.key]
		for nestedKey := range collectKeys(oldValue, c.key) {
			if _, exists := oldValues[nestedKey]; !exists {
				oldValues[nestedKey] = getNestedValue(oldValue, strings.TrimPrefix(nestedKey, c.key+"."))
			}
		}
	}

	// Apply all changes
	for _, c := range changes {
		o.setValueInternal(c.key, c.value)
	}

	// Collect new values and determine what actually changed
	changesMap := make(map[string]struct{ old, new any })
	for key, oldValue := range oldValues {
		newValue := o.getValueInternal(key)
		if !reflect.DeepEqual(oldValue, newValue) {
			changesMap[key] = struct{ old, new any }{oldValue, newValue}
		}
	}

	// Also check for new nested keys created by the changes
	for _, c := range changes {
		newValue := o.getValueInternal(c.key)
		for nestedKey := range collectKeys(newValue, c.key) {
			if _, exists := changesMap[nestedKey]; !exists {
				oldValue := oldValues[nestedKey]
				newNestedValue := o.getValueInternal(nestedKey)
				if !reflect.DeepEqual(oldValue, newNestedValue) {
					changesMap[nestedKey] = struct{ old, new any }{oldValue, newNestedValue}
				}
			}
		}
	}

	// Copy subscriptions for notification
	subs := make([]*subscription, 0, len(o.subscriptions))
	for _, sub := range o.subscriptions {
		subs = append(subs, sub)
	}
	o.mu.Unlock()

	// Notify each subscription once per matching key
	for key := range changesMap {
		for _, sub := range subs {
			if matchPattern(sub.pattern, key) {
				sub.callback(o, key)
			}
		}
	}
}

// copyMap creates a shallow copy of a map.
func copyMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// copySlice creates a shallow copy of a slice.
func copySlice(s []any) []any {
	if s == nil {
		return nil
	}
	result := make([]any, len(s))
	copy(result, s)
	return result
}

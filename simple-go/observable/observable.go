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

// GetMap returns a copy of the map at the given key path.
// Returns nil if the key does not exist.
// Panics if the key exists but the value is not a map.
func (o *Observable) GetMap(key string) map[string]any {
	val := o.GetValue(key)
	if val == nil {
		return nil
	}
	m, ok := val.(map[string]any)
	preconditions.Check(ok, "value at key %q is not a map, got %T", key, val)
	return copyMap(m)
}

// GetSlice returns a copy of the slice at the given key path.
// Returns nil if the key does not exist.
// Panics if the key exists but the value is not a slice.
func (o *Observable) GetSlice(key string) []any {
	val := o.GetValue(key)
	if val == nil {
		return nil
	}
	s, ok := val.([]any)
	preconditions.Check(ok, "value at key %q is not a slice, got %T", key, val)
	return copySlice(s)
}

// GetString returns the string at the given key path.
// Returns empty string if the key does not exist.
// Panics if the key exists but the value is not a string.
func (o *Observable) GetString(key string) string {
	return GetValueAs[string](o, key)
}

// GetInt returns the int at the given key path.
// Returns 0 if the key does not exist.
// Panics if the key exists but the value is not an int.
func (o *Observable) GetInt(key string) int {
	return GetValueAs[int](o, key)
}

// GetFloat64 returns the float64 at the given key path.
// Returns 0 if the key does not exist.
// Panics if the key exists but the value is not a float64.
func (o *Observable) GetFloat64(key string) float64 {
	return GetValueAs[float64](o, key)
}

// GetBool returns the bool at the given key path.
// Returns false if the key does not exist.
// Panics if the key exists but the value is not a bool.
func (o *Observable) GetBool(key string) bool {
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
	// Collect all changes for notification
	changes := make(map[string]struct{ old, new any })

	// Hold lock for state changes
	o.mu.Lock()

	// Validate against schema before making changes
	if errMsg := o.validateAgainstSchema(key, value); errMsg != "" {
		o.mu.Unlock()
		preconditions.Check(false, "%s", errMsg)
	}

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

	// Copy subscriptions for notification outside lock
	subs := make([]*subscription, 0, len(o.subscriptions))
	for _, sub := range o.subscriptions {
		subs = append(subs, sub)
	}

	o.mu.Unlock()

	// Notify subscribers outside lock to allow callbacks to access observable state
	o.notifySubscribersOutsideLock(subs, changes)
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

// notifySubscribersOutsideLock notifies all matching subscribers of the changes.
// Each subscription is triggered at most once per SetValueAtKey call.
// This must be called without holding the lock to allow callbacks to access observable state.
func (o *Observable) notifySubscribersOutsideLock(subs []*subscription, changes map[string]struct{ old, new any }) {
	if len(changes) == 0 {
		return
	}

	// For each subscription, check if any of its patterns match any changed key
	for _, sub := range subs {
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

// changeMessage represents a change to be processed by the transaction goroutine.
type changeMessage struct {
	key      string
	oldValue any
}

// commitMessage signals the goroutine to process buffered changes.
type commitMessage struct {
	done chan struct{}
}

// TransactionalObservable wraps Observable with transactional change batching.
// Changes are buffered until CommitChanges is called, at which point all
// subscribers are notified with uniqued key changes.
type TransactionalObservable struct {
	*Observable
	changeChan chan changeMessage
	commitChan chan commitMessage
	closeChan  chan struct{}
	closed     bool
}

// NewTransactional creates a new TransactionalObservable with an empty map as root.
func NewTransactional() *TransactionalObservable {
	return NewTransactionalWithData(make(map[string]any))
}

// NewTransactionalWithData creates a new TransactionalObservable with the provided data.
func NewTransactionalWithData(data any) *TransactionalObservable {
	t := &TransactionalObservable{
		Observable: NewWithData(data),
		changeChan: make(chan changeMessage, 1000),
		commitChan: make(chan commitMessage),
		closeChan:  make(chan struct{}),
	}
	go t.processLoop()
	return t
}

// processLoop is the goroutine that buffers changes and processes commits.
func (t *TransactionalObservable) processLoop() {
	// Buffer for tracking original values of changed keys
	// key -> oldValue (the value before any changes in this transaction)
	pendingChanges := make(map[string]any)

	for {
		select {
		case change := <-t.changeChan:
			// Only record the first old value for each key
			if _, exists := pendingChanges[change.key]; !exists {
				pendingChanges[change.key] = change.oldValue
			}

		case commit := <-t.commitChan:
			// Drain any remaining changes from the channel first
			draining := true
			for draining {
				select {
				case change := <-t.changeChan:
					if _, exists := pendingChanges[change.key]; !exists {
						pendingChanges[change.key] = change.oldValue
					}
				default:
					draining = false
				}
			}

			// Process all pending changes
			if len(pendingChanges) > 0 {
				// Build changes map with old and current values
				changes := make(map[string]struct{ old, new any })

				t.Observable.mu.RLock()
				for key, oldValue := range pendingChanges {
					newValue := t.Observable.getValueInternal(key)
					if !reflect.DeepEqual(oldValue, newValue) {
						changes[key] = struct{ old, new any }{oldValue, newValue}
					}
				}

				// Copy subscriptions for notification
				subs := make([]*subscription, 0, len(t.Observable.subscriptions))
				for _, sub := range t.Observable.subscriptions {
					subs = append(subs, sub)
				}
				t.Observable.mu.RUnlock()

				// Notify subscribers for each changed key (unlike base Observable which
				// notifies once per SetValueAtKey, transactional notifies once per unique key)
				t.notifyPerKey(subs, changes)

				// Clear pending changes for next transaction
				pendingChanges = make(map[string]any)
			}

			// Signal completion
			close(commit.done)

		case <-t.closeChan:
			return
		}
	}
}

// SetValueAtKey sets the value at the given key path, buffering the change
// for later notification when CommitChanges is called.
func (t *TransactionalObservable) SetValueAtKey(key string, value any) {
	// Get old value before the change
	t.Observable.mu.Lock()
	oldValue := t.Observable.getValueInternal(key)
	// Also collect nested keys that might change
	oldNestedKeys := collectKeys(oldValue, key)

	// Perform the actual set
	t.Observable.setValueInternal(key, value)

	// Get new nested keys
	newValue := t.Observable.getValueInternal(key)
	newNestedKeys := collectKeys(newValue, key)
	t.Observable.mu.Unlock()

	// Send change to buffer goroutine
	if !t.closed {
		t.changeChan <- changeMessage{key: key, oldValue: oldValue}

		// Also send nested key changes
		allNestedKeys := make(map[string]bool)
		for k := range oldNestedKeys {
			allNestedKeys[k] = true
		}
		for k := range newNestedKeys {
			allNestedKeys[k] = true
		}

		for nestedKey := range allNestedKeys {
			if nestedKey != key {
				oldNested := getNestedValue(oldValue, strings.TrimPrefix(nestedKey, key+"."))
				t.changeChan <- changeMessage{key: nestedKey, oldValue: oldNested}
			}
		}
	}
}

// DeleteValueAtKey removes the value at the given key path.
// This is equivalent to SetValueAtKey(key, nil).
func (t *TransactionalObservable) DeleteValueAtKey(key string) {
	t.SetValueAtKey(key, nil)
}

// CommitChanges triggers notification of all buffered changes to subscribers.
// Changes are uniqued by key, and each subscriber is notified at most once.
// This method blocks until all notifications are complete.
func (t *TransactionalObservable) CommitChanges() {
	if t.closed {
		return
	}

	done := make(chan struct{})
	t.commitChan <- commitMessage{done: done}
	<-done
}

// Close stops the transaction processing goroutine.
// After Close, SetValueAtKey still works but changes won't be buffered or notified.
func (t *TransactionalObservable) Close() {
	if !t.closed {
		t.closed = true
		close(t.closeChan)
	}
}

// notifyPerKey notifies subscribers once per matching changed key.
// Unlike notifySubscribersOutsideLock which triggers each subscription once per batch,
// this method triggers for each key that matches a subscription's patterns.
func (t *TransactionalObservable) notifyPerKey(subs []*subscription, changes map[string]struct{ old, new any }) {
	for key, change := range changes {
		for _, sub := range subs {
			for _, pattern := range sub.patterns {
				if matchPattern(pattern, key) {
					sub.callback(key, change.old, change.new)
					break // Only trigger once per subscription per key
				}
			}
		}
	}
}

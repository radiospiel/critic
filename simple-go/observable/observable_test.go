package observable

import (
	"testing"

	"git.15b.it/eno/critic/simple-go/assert"
)

func TestNew(t *testing.T) {
	obs := New()
	assert.NotNil(t, obs, "New() should return non-nil Observable")
	assert.NotNil(t, obs.GetValue(""), "root should be an empty map")
}

func TestNewWithData(t *testing.T) {
	data := map[string]any{"foo": "bar"}
	obs := NewWithData(data)
	assert.Equals(t, obs.GetValue("foo"), "bar", "should have foo=bar")
}

func TestNewWithDataNil(t *testing.T) {
	obs := NewWithData(nil)
	assert.Nil(t, obs.GetValue(""), "root should be nil")
}

func TestNewWithDataSlice(t *testing.T) {
	data := []any{"a", "b", "c"}
	obs := NewWithData(data)
	assert.Equals(t, obs.GetValue("0"), "a", "should have index 0")
	assert.Equals(t, obs.GetValue("1"), "b", "should have index 1")
}

func TestGetValueSimple(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("foo", "bar")
	assert.Equals(t, obs.GetValue("foo"), "bar", "should get foo")
}

func TestGetValueNested(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("x.y.z", "value")
	assert.Equals(t, obs.GetValue("x.y.z"), "value", "should get x.y.z")
	assert.NotNil(t, obs.GetValue("x.y"), "should get x.y as map")
	assert.NotNil(t, obs.GetValue("x"), "should get x as map")
}

func TestGetValueMissing(t *testing.T) {
	obs := New()
	assert.Nil(t, obs.GetValue("nonexistent"), "should return nil for missing key")
	assert.Nil(t, obs.GetValue("a.b.c"), "should return nil for missing nested key")
}

func TestGetValueRoot(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("foo", "bar")
	root := obs.GetValue("")
	m, ok := root.(map[string]any)
	assert.True(t, ok, "root should be a map")
	assert.Equals(t, m["foo"], "bar", "root should contain foo")
}

func TestSetValueAtKeyCreatesIntermediateMaps(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("a.b.c", "value")

	// Check that intermediate maps were created
	a := obs.GetValue("a")
	_, ok := a.(map[string]any)
	assert.True(t, ok, "a should be a map")

	ab := obs.GetValue("a.b")
	_, ok = ab.(map[string]any)
	assert.True(t, ok, "a.b should be a map")

	assert.Equals(t, obs.GetValue("a.b.c"), "value", "a.b.c should be 'value'")
}

func TestSetValueAtKeyCreatesIntermediateSlices(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("x.1.a", "value")

	// x should be a slice (because 1 is a number)
	x := obs.GetValue("x")
	slice, ok := x.([]any)
	assert.True(t, ok, "x should be a slice, got %T", x)

	// x.1 should be a map (because "a" is not a number)
	x1 := obs.GetValue("x.1")
	_, ok = x1.(map[string]any)
	assert.True(t, ok, "x.1 should be a map, got %T", x1)

	assert.Equals(t, obs.GetValue("x.1.a"), "value", "x.1.a should be 'value'")

	// Slice should have been extended to index 1
	assert.Equals(t, len(slice), 2, "slice should have length 2")
	assert.Nil(t, slice[0], "slice[0] should be nil")
}

func TestSetValueAtKeyWithNestedMap(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("x.1", map[string]any{"a": "value"})

	assert.Equals(t, obs.GetValue("x.1.a"), "value", "x.1.a should be 'value'")
}

func TestSetValueAtKeyOverwrite(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("foo", "bar")
	obs.SetValueAtKey("foo", "baz")
	assert.Equals(t, obs.GetValue("foo"), "baz", "should overwrite value")
}

func TestDeleteValueAtKey(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("foo", "bar")
	obs.DeleteValueAtKey("foo")
	assert.Nil(t, obs.GetValue("foo"), "should delete value")
}

func TestDeleteValueAtKeyNested(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("a.b.c", "value")
	obs.DeleteValueAtKey("a.b.c")
	assert.Nil(t, obs.GetValue("a.b.c"), "should delete nested value")
	// Parent structures should still exist
	assert.NotNil(t, obs.GetValue("a.b"), "a.b should still exist")
	assert.NotNil(t, obs.GetValue("a"), "a should still exist")
}

func TestOnKeyChangeSimple(t *testing.T) {
	obs := New()
	var callCount int
	var lastKey string
	var lastOld, lastNew any

	obs.OnKeyChange([]string{"foo"}, func(key string, oldValue, newValue any) {
		callCount++
		lastKey = key
		lastOld = oldValue
		lastNew = newValue
	})

	obs.SetValueAtKey("foo", "bar")

	assert.Equals(t, callCount, 1, "callback should be called once")
	assert.Equals(t, lastKey, "foo", "key should be 'foo'")
	assert.Nil(t, lastOld, "old value should be nil")
	assert.Equals(t, lastNew, "bar", "new value should be 'bar'")
}

func TestOnKeyChangeNoChangeNoCallback(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("foo", "bar")

	var callCount int
	obs.OnKeyChange([]string{"foo"}, func(key string, oldValue, newValue any) {
		callCount++
	})

	// Setting the same value shouldn't trigger callback
	obs.SetValueAtKey("foo", "bar")

	assert.Equals(t, callCount, 0, "callback should not be called when value unchanged")
}

func TestOnKeyChangeWildcard(t *testing.T) {
	obs := New()
	var matchedKeys []string

	obs.OnKeyChange([]string{"foo.*"}, func(key string, oldValue, newValue any) {
		matchedKeys = append(matchedKeys, key)
	})

	obs.SetValueAtKey("foo.a", "value1")
	obs.SetValueAtKey("foo.b", "value2")
	obs.SetValueAtKey("bar.c", "value3") // Should not match

	assert.Equals(t, len(matchedKeys), 2, "should match 2 keys")
	assert.Contains(t, matchedKeys, "foo.a", "should contain foo.a")
	assert.Contains(t, matchedKeys, "foo.b", "should contain foo.b")
}

func TestOnKeyChangeNestedWildcard(t *testing.T) {
	obs := New()
	var matchedKeys []string

	obs.OnKeyChange([]string{"x.*.a"}, func(key string, oldValue, newValue any) {
		matchedKeys = append(matchedKeys, key)
	})

	// Use consistent key types - all non-numeric so x is a map
	obs.SetValueAtKey("x.one.a", "value1")
	obs.SetValueAtKey("x.dd.a", "value2")
	obs.SetValueAtKey("x.one.b", "value3") // Should not match

	assert.Equals(t, len(matchedKeys), 2, "should match 2 keys")
	assert.Contains(t, matchedKeys, "x.one.a", "should contain x.one.a")
	assert.Contains(t, matchedKeys, "x.dd.a", "should contain x.dd.a")
}

func TestOnKeyChangeDeepSubscriptionTriggeredByNestedSet(t *testing.T) {
	obs := New()
	var triggered bool

	// Subscribe to a deep path
	obs.OnKeyChange([]string{"x.*.a"}, func(key string, oldValue, newValue any) {
		triggered = true
	})

	// Set a nested value that contains the path
	obs.SetValueAtKey("x.1", map[string]any{"a": "value"})

	assert.True(t, triggered, "subscription on x.*.a should be triggered when setting x.1 with nested 'a'")
}

func TestOnKeyChangeCalledOncePerSetValueAtKey(t *testing.T) {
	obs := New()
	var callCount int

	// Two patterns that would both match x.1.a
	obs.OnKeyChange([]string{"x.1.a", "x.*"}, func(key string, oldValue, newValue any) {
		callCount++
	})

	obs.SetValueAtKey("x.1", map[string]any{"a": "value"})

	// Should only be called once even though multiple patterns match
	assert.Equals(t, callCount, 1, "subscription should only trigger once per SetValueAtKey call")
}

func TestOnKeyChangeMultipleSubscriptions(t *testing.T) {
	obs := New()
	var sub1Called, sub2Called bool

	obs.OnKeyChange([]string{"foo"}, func(key string, oldValue, newValue any) {
		sub1Called = true
	})
	obs.OnKeyChange([]string{"foo"}, func(key string, oldValue, newValue any) {
		sub2Called = true
	})

	obs.SetValueAtKey("foo", "bar")

	assert.True(t, sub1Called, "subscription 1 should be called")
	assert.True(t, sub2Called, "subscription 2 should be called")
}

func TestClearSubscriptions(t *testing.T) {
	obs := New()
	var callCount int

	subs := obs.OnKeyChange([]string{"foo"}, func(key string, oldValue, newValue any) {
		callCount++
	})

	obs.SetValueAtKey("foo", "bar")
	assert.Equals(t, callCount, 1, "callback should be called before clear")

	obs.ClearSubscriptions(subs...)

	obs.SetValueAtKey("foo", "baz")
	assert.Equals(t, callCount, 1, "callback should not be called after clear")
}

func TestClearSubscriptionsPartial(t *testing.T) {
	obs := New()
	var sub1Count, sub2Count int

	sub1 := obs.OnKeyChange([]string{"foo"}, func(key string, oldValue, newValue any) {
		sub1Count++
	})
	obs.OnKeyChange([]string{"foo"}, func(key string, oldValue, newValue any) {
		sub2Count++
	})

	obs.SetValueAtKey("foo", "bar")
	assert.Equals(t, sub1Count, 1, "sub1 should be called")
	assert.Equals(t, sub2Count, 1, "sub2 should be called")

	obs.ClearSubscriptions(sub1...)

	obs.SetValueAtKey("foo", "baz")
	assert.Equals(t, sub1Count, 1, "sub1 should not be called after clear")
	assert.Equals(t, sub2Count, 2, "sub2 should still be called")
}

func TestSetValueAtKeyPanicsOnTypeMismatch(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("foo", "bar") // foo is a string

	defer func() {
		r := recover()
		assert.NotNil(t, r, "should panic when trying to set foo.x when foo is not a map")
	}()

	obs.SetValueAtKey("foo.x", "value") // Should panic
}

func TestSetValueAtKeyPanicsOnSliceTypeMismatch(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("arr.0", "value") // arr[0] is a string

	defer func() {
		r := recover()
		assert.NotNil(t, r, "should panic when trying to use string as map")
	}()

	obs.SetValueAtKey("arr.0.x", "value") // Should panic because arr[0] is a string, not a map
}

func TestSetValueAtKeyPanicsOnLargeIndex(t *testing.T) {
	obs := New()

	defer func() {
		r := recover()
		assert.NotNil(t, r, "should panic when index >= 100000")
	}()

	obs.SetValueAtKey("arr.100000", "value")
}

func TestSetValueAtKeyAcceptsMaxValidIndex(t *testing.T) {
	obs := New()
	// Should not panic for index 99999 (maxArrayIndex)
	obs.SetValueAtKey("arr.99999", "value")
	assert.Equals(t, obs.GetValue("arr.99999"), "value", "should accept index 99999")
}

func TestSetValueAtKeyWithSliceValue(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("items", []any{"a", "b", "c"})

	assert.Equals(t, obs.GetValue("items.0"), "a", "items[0] should be 'a'")
	assert.Equals(t, obs.GetValue("items.1"), "b", "items[1] should be 'b'")
	assert.Equals(t, obs.GetValue("items.2"), "c", "items[2] should be 'c'")
}

func TestSetValueAtKeyWithMapValue(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("config", map[string]any{
		"name":  "test",
		"count": 42,
	})

	assert.Equals(t, obs.GetValue("config.name"), "test", "config.name should be 'test'")
	assert.Equals(t, obs.GetValue("config.count"), 42, "config.count should be 42")
}

func TestSetValueAtKeyExtendSlice(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("arr.0", "first")
	obs.SetValueAtKey("arr.5", "sixth")

	arr := obs.GetValue("arr").([]any)
	assert.Equals(t, len(arr), 6, "array should have length 6")
	assert.Equals(t, arr[0], "first", "arr[0] should be 'first'")
	assert.Nil(t, arr[1], "arr[1] should be nil")
	assert.Nil(t, arr[4], "arr[4] should be nil")
	assert.Equals(t, arr[5], "sixth", "arr[5] should be 'sixth'")
}

func TestSetValueAtKeyRoot(t *testing.T) {
	obs := NewWithData(nil)
	obs.SetValueAtKey("", map[string]any{"foo": "bar"})
	assert.Equals(t, obs.GetValue("foo"), "bar", "root should be set to new map")
}

func TestSetValueAtKeyRootToNil(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("foo", "bar")
	obs.SetValueAtKey("", nil)
	assert.Nil(t, obs.GetValue(""), "root should be nil")
	assert.Nil(t, obs.GetValue("foo"), "foo should be nil after root set to nil")
}

func TestGetValueWithArrayIndex(t *testing.T) {
	obs := NewWithData(map[string]any{
		"items": []any{"a", "b", "c"},
	})

	assert.Equals(t, obs.GetValue("items.0"), "a", "should get array element by index")
	assert.Equals(t, obs.GetValue("items.2"), "c", "should get last array element")
	assert.Nil(t, obs.GetValue("items.10"), "should return nil for out of bounds index")
}

func TestOnKeyChangeMultiplePatterns(t *testing.T) {
	obs := New()
	var matchedKeys []string

	// Subscribe with multiple patterns
	obs.OnKeyChange([]string{"foo.*", "bar.*"}, func(key string, oldValue, newValue any) {
		matchedKeys = append(matchedKeys, key)
	})

	obs.SetValueAtKey("foo.a", "value1")
	obs.SetValueAtKey("bar.b", "value2")
	obs.SetValueAtKey("baz.c", "value3") // Should not match

	assert.Equals(t, len(matchedKeys), 2, "should match 2 keys")
}

func TestDeepNestedChange(t *testing.T) {
	obs := New()
	var triggered bool
	var receivedKey string

	obs.OnKeyChange([]string{"a.b.c.d.e"}, func(key string, oldValue, newValue any) {
		triggered = true
		receivedKey = key
	})

	obs.SetValueAtKey("a.b.c.d.e", "deep value")

	assert.True(t, triggered, "should trigger on deep nested change")
	assert.Equals(t, receivedKey, "a.b.c.d.e", "should receive correct key")
}

func TestPatternDoesNotMatchDeeper(t *testing.T) {
	obs := New()
	var triggered bool

	// Pattern with single * should not match deeper paths
	obs.OnKeyChange([]string{"foo.*.bar"}, func(key string, oldValue, newValue any) {
		triggered = true
	})

	// This should NOT match foo.*.bar pattern because the path is deeper
	obs.SetValueAtKey("foo.1.bar.deep", "value")

	assert.False(t, triggered, "foo.*.bar should not match foo.1.bar.deep")
}

func TestPatternMatchesSingleLevel(t *testing.T) {
	obs := New()
	var triggered bool

	obs.OnKeyChange([]string{"foo.*.bar"}, func(key string, oldValue, newValue any) {
		triggered = true
	})

	obs.SetValueAtKey("foo.dd.bar", "value")

	assert.True(t, triggered, "foo.*.bar should match foo.dd.bar")
}

func TestComplexNestedSetTriggersMultipleChanges(t *testing.T) {
	obs := New()

	var aTriggered, bTriggered bool

	obs.OnKeyChange([]string{"data.users.*.name"}, func(key string, oldValue, newValue any) {
		aTriggered = true
	})

	obs.OnKeyChange([]string{"data.users.*.age"}, func(key string, oldValue, newValue any) {
		bTriggered = true
	})

	// Set a complex nested structure
	obs.SetValueAtKey("data.users.0", map[string]any{
		"name": "Alice",
		"age":  30,
	})

	assert.True(t, aTriggered, "name subscription should trigger")
	assert.True(t, bTriggered, "age subscription should trigger")
}

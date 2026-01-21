package observable

import (
	"fmt"
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

// Tests for typed getters

func TestGetValueAs(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("name", "Alice")
	obs.SetValueAtKey("age", 30)
	obs.SetValueAtKey("active", true)
	obs.SetValueAtKey("score", 95.5)

	// Test successful type assertions
	assert.Equals(t, GetValueAs[string](obs, "name"), "Alice", "name should be Alice")
	assert.Equals(t, GetValueAs[int](obs, "age"), 30, "age should be 30")
	assert.True(t, GetValueAs[bool](obs, "active"), "active should be true")
	assert.Equals(t, GetValueAs[float64](obs, "score"), 95.5, "score should be 95.5")
}

func TestGetValueAsPanicsOnWrongType(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("name", "Alice")

	defer func() {
		r := recover()
		assert.NotNil(t, r, "should panic on type mismatch")
	}()

	GetValueAs[int](obs, "name")
}

func TestGetValueAsMissingReturnsZero(t *testing.T) {
	obs := New()

	assert.Equals(t, GetValueAs[string](obs, "nonexistent"), "", "should return empty string for missing key")
	assert.Equals(t, GetValueAs[int](obs, "nonexistent"), 0, "should return 0 for missing key")
	assert.False(t, GetValueAs[bool](obs, "nonexistent"), "should return false for missing key")
}

func TestGetMap(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("config", map[string]any{"key": "value"})

	m := obs.GetMap("config")
	assert.NotNil(t, m, "should get map")
	assert.Equals(t, m["key"], "value", "should have correct value")

	// Test missing returns nil
	assert.Nil(t, obs.GetMap("nonexistent"), "should return nil for missing key")
}

func TestGetMapPanicsOnWrongType(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("notmap", "string")

	defer func() {
		r := recover()
		assert.NotNil(t, r, "should panic on type mismatch")
	}()

	obs.GetMap("notmap")
}

func TestGetMapReturnsCopy(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("config", map[string]any{"key": "value"})

	m := obs.GetMap("config")
	// Modify the copy at top level
	m["key"] = "modified"
	m["newkey"] = "added"

	// Original should be unchanged
	original := obs.GetMap("config")
	assert.Equals(t, original["key"], "value", "original should be unchanged")
	assert.Nil(t, original["newkey"], "original should not have new key")
}

func TestGetSlice(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("items", []any{"a", "b", "c"})

	s := obs.GetSlice("items")
	assert.NotNil(t, s, "should get slice")
	assert.Equals(t, len(s), 3, "should have 3 items")
	assert.Equals(t, s[0], "a", "first item should be 'a'")

	// Test missing returns nil
	assert.Nil(t, obs.GetSlice("nonexistent"), "should return nil for missing key")
}

func TestGetSlicePanicsOnWrongType(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("notslice", "string")

	defer func() {
		r := recover()
		assert.NotNil(t, r, "should panic on type mismatch")
	}()

	obs.GetSlice("notslice")
}

func TestGetSliceReturnsCopy(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("items", []any{"a", "b", "c"})

	s := obs.GetSlice("items")
	// Modify the copy at top level
	s[0] = "modified"
	s = append(s, "d")

	// Original should be unchanged
	original := obs.GetSlice("items")
	assert.Equals(t, original[0], "a", "original should be unchanged")
	assert.Equals(t, len(original), 3, "original length should be unchanged")
}

func TestGetString(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("name", "Alice")

	assert.Equals(t, obs.GetString("name"), "Alice", "should be Alice")
	assert.Equals(t, obs.GetString("nonexistent"), "", "should return empty for missing")
}

func TestGetStringPanicsOnWrongType(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("notstring", 42)

	defer func() {
		r := recover()
		assert.NotNil(t, r, "should panic on type mismatch")
	}()

	obs.GetString("notstring")
}

func TestGetInt(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("count", 42)

	assert.Equals(t, obs.GetInt("count"), 42, "should be 42")
	assert.Equals(t, obs.GetInt("nonexistent"), 0, "should return 0 for missing")
}

func TestGetIntPanicsOnWrongType(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("notint", "string")

	defer func() {
		r := recover()
		assert.NotNil(t, r, "should panic on type mismatch")
	}()

	obs.GetInt("notint")
}

func TestGetFloat64(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("price", 19.99)

	assert.Equals(t, obs.GetFloat64("price"), 19.99, "should be 19.99")
	assert.Equals(t, obs.GetFloat64("nonexistent"), 0.0, "should return 0 for missing")
}

func TestGetFloat64PanicsOnWrongType(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("notfloat", "string")

	defer func() {
		r := recover()
		assert.NotNil(t, r, "should panic on type mismatch")
	}()

	obs.GetFloat64("notfloat")
}

func TestGetBool(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("enabled", true)
	obs.SetValueAtKey("disabled", false)

	assert.True(t, obs.GetBool("enabled"), "should be true")
	assert.False(t, obs.GetBool("disabled"), "should be false")
	assert.False(t, obs.GetBool("nonexistent"), "should return false for missing")
}

func TestGetBoolPanicsOnWrongType(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("notbool", "string")

	defer func() {
		r := recover()
		assert.NotNil(t, r, "should panic on type mismatch")
	}()

	obs.GetBool("notbool")
}

// Tests for struct conversion

type TestUser struct {
	Name  string `json:"name"`
	Age   int    `json:"age"`
	Email string `json:"email,omitempty"`
}

type TestConfig struct {
	Host    string   `json:"host"`
	Port    int      `json:"port"`
	Tags    []string `json:"tags"`
	Enabled bool     `json:"enabled"`
}

func TestGetValueAsStruct(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("user", map[string]any{
		"name":  "Alice",
		"age":   30,
		"email": "alice@example.com",
	})

	user := GetValueAs[TestUser](obs, "user")
	assert.Equals(t, user.Name, "Alice", "name should be Alice")
	assert.Equals(t, user.Age, 30, "age should be 30")
	assert.Equals(t, user.Email, "alice@example.com", "email should match")
}

func TestGetValueAsStructWithSlice(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("config", map[string]any{
		"host":    "localhost",
		"port":    8080,
		"tags":    []any{"web", "api", "production"},
		"enabled": true,
	})

	config := GetValueAs[TestConfig](obs, "config")
	assert.Equals(t, config.Host, "localhost", "host should be localhost")
	assert.Equals(t, config.Port, 8080, "port should be 8080")
	assert.Equals(t, len(config.Tags), 3, "should have 3 tags")
	assert.Equals(t, config.Tags[0], "web", "first tag should be 'web'")
	assert.True(t, config.Enabled, "enabled should be true")
}

func TestGetValueAsStructMissingReturnsZero(t *testing.T) {
	obs := New()

	user := GetValueAs[TestUser](obs, "nonexistent")
	assert.Equals(t, user.Name, "", "name should be empty")
	assert.Equals(t, user.Age, 0, "age should be 0")
}

func TestGetValueAsStructReturnsCopy(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("user", map[string]any{
		"name": "Alice",
		"age":  30,
	})

	// Get the struct
	user1 := GetValueAs[TestUser](obs, "user")
	user1.Name = "Modified"

	// Get again - should be unchanged
	user2 := GetValueAs[TestUser](obs, "user")
	assert.Equals(t, user2.Name, "Alice", "original should be unchanged")
}

func TestGetValueAsSliceOfStructs(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("users", []any{
		map[string]any{"name": "Alice", "age": 30},
		map[string]any{"name": "Bob", "age": 25},
	})

	users := GetValueAs[[]TestUser](obs, "users")
	assert.Equals(t, len(users), 2, "should have 2 users")
	assert.Equals(t, users[0].Name, "Alice", "first user should be Alice")
	assert.Equals(t, users[1].Name, "Bob", "second user should be Bob")
}

func TestGetValueAsNestedStruct(t *testing.T) {
	type Address struct {
		City    string `json:"city"`
		Country string `json:"country"`
	}
	type Person struct {
		Name    string  `json:"name"`
		Address Address `json:"address"`
	}

	obs := New()
	obs.SetValueAtKey("person", map[string]any{
		"name": "Alice",
		"address": map[string]any{
			"city":    "New York",
			"country": "USA",
		},
	})

	person := GetValueAs[Person](obs, "person")
	assert.Equals(t, person.Name, "Alice", "name should be Alice")
	assert.Equals(t, person.Address.City, "New York", "city should be New York")
	assert.Equals(t, person.Address.Country, "USA", "country should be USA")
}

func TestGetValueAsPointerToStruct(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("user", map[string]any{
		"name": "Alice",
		"age":  30,
	})

	user := GetValueAs[*TestUser](obs, "user")
	assert.NotNil(t, user, "should get pointer to struct")
	assert.Equals(t, user.Name, "Alice", "name should be Alice")
}

func TestGetValueAsStructPanicsOnIncompatible(t *testing.T) {
	obs := New()
	obs.SetValueAtKey("data", "not a struct compatible value")

	defer func() {
		r := recover()
		assert.NotNil(t, r, "should panic on incompatible type")
	}()

	GetValueAs[TestUser](obs, "data")
}

// Tests for transactional observable

func TestTransactionalObservableChangesNotNotifiedUntilCommit(t *testing.T) {
	obs := NewTransactional()

	var callCount int
	obs.OnKeyChange([]string{"foo"}, func(key string, oldValue, newValue any) {
		callCount++
	})

	obs.Txn(func(tx *Txn) {
		tx.SetValueAtKey("foo", "bar")
		assert.Equals(t, callCount, 0, "callback should not be called before commit")
	})

	assert.Equals(t, callCount, 1, "callback should be called after commit")
}

func TestTransactionalObservableMultipleChangesToSameKeyUniqued(t *testing.T) {
	obs := NewTransactional()

	var callCount int
	var lastOld, lastNew any

	obs.OnKeyChange([]string{"foo"}, func(key string, oldValue, newValue any) {
		callCount++
		lastOld = oldValue
		lastNew = newValue
	})

	obs.Txn(func(tx *Txn) {
		tx.SetValueAtKey("foo", "first")
		tx.SetValueAtKey("foo", "second")
		tx.SetValueAtKey("foo", "third")
	})

	assert.Equals(t, callCount, 1, "callback should only be called once for multiple changes to same key")
	assert.Nil(t, lastOld, "old value should be the original (nil)")
	assert.Equals(t, lastNew, "third", "new value should be the final value")
}

func TestTransactionalObservableBatchesMultipleKeys(t *testing.T) {
	obs := NewTransactional()

	var fooCount, barCount int

	obs.OnKeyChange([]string{"foo"}, func(key string, oldValue, newValue any) {
		fooCount++
	})
	obs.OnKeyChange([]string{"bar"}, func(key string, oldValue, newValue any) {
		barCount++
	})

	obs.Txn(func(tx *Txn) {
		tx.SetValueAtKey("foo", "value1")
		tx.SetValueAtKey("bar", "value2")

		assert.Equals(t, fooCount, 0, "foo callback should not be called before commit")
		assert.Equals(t, barCount, 0, "bar callback should not be called before commit")
	})

	assert.Equals(t, fooCount, 1, "foo callback should be called once after commit")
	assert.Equals(t, barCount, 1, "bar callback should be called once after commit")
}

func TestTransactionalObservableNoNotificationWithoutChanges(t *testing.T) {
	obs := NewTransactional()

	var callCount int
	obs.OnKeyChange([]string{"foo"}, func(key string, oldValue, newValue any) {
		callCount++
	})

	obs.Txn(func(tx *Txn) {
		// Empty transaction
	})

	assert.Equals(t, callCount, 0, "callback should not be called when no changes")
}

func TestTransactionalObservableSetThenDeleteNotNotified(t *testing.T) {
	obs := NewTransactional()

	var callCount int
	obs.OnKeyChange([]string{"foo"}, func(key string, oldValue, newValue any) {
		callCount++
	})

	obs.Txn(func(tx *Txn) {
		tx.SetValueAtKey("foo", "bar")
		tx.DeleteValueAtKey("foo")
	})

	// The key was set then deleted - final state is same as initial (nil -> nil)
	// so no notification should occur
	assert.Equals(t, callCount, 0, "callback should not be called when net change is nil")
}

func TestTransactionalObservableMultipleTransactions(t *testing.T) {
	obs := NewTransactional()

	var callCount int
	obs.OnKeyChange([]string{"foo"}, func(key string, oldValue, newValue any) {
		callCount++
	})

	obs.Txn(func(tx *Txn) {
		tx.SetValueAtKey("foo", "first")
	})
	assert.Equals(t, callCount, 1, "first transaction should trigger callback")

	obs.Txn(func(tx *Txn) {
		tx.SetValueAtKey("foo", "second")
	})
	assert.Equals(t, callCount, 2, "second transaction should trigger callback")
}

func TestTransactionalObservableNestedChanges(t *testing.T) {
	obs := NewTransactional()

	var triggered bool
	obs.OnKeyChange([]string{"x.*.a"}, func(key string, oldValue, newValue any) {
		triggered = true
	})

	obs.Txn(func(tx *Txn) {
		tx.SetValueAtKey("x.1", map[string]any{"a": "value"})
	})

	assert.True(t, triggered, "nested subscription should be triggered on commit")
}

func TestTransactionalObservableManyChanges(t *testing.T) {
	// Test that we can handle more changes than would fit in any fixed buffer
	obs := NewTransactional()

	var callCount int
	obs.OnKeyChange([]string{"*"}, func(key string, oldValue, newValue any) {
		callCount++
	})

	obs.Txn(func(tx *Txn) {
		// Send 5000 changes to different keys (more than any reasonable buffer)
		for i := 0; i < 5000; i++ {
			tx.SetValueAtKey(fmt.Sprintf("key%d", i), i)
		}
	})

	assert.Equals(t, callCount, 5000, "should notify for all 5000 unique keys")
}

func TestTransactionalObservableManyChangesToSameKey(t *testing.T) {
	// Test that many changes to the same key still only notify once
	obs := NewTransactional()

	var callCount int
	var lastValue any
	obs.OnKeyChange([]string{"foo"}, func(key string, oldValue, newValue any) {
		callCount++
		lastValue = newValue
	})

	obs.Txn(func(tx *Txn) {
		// Send 10000 changes to the same key
		for i := 0; i < 10000; i++ {
			tx.SetValueAtKey("foo", i)
		}
	})

	assert.Equals(t, callCount, 1, "should only notify once for same key")
	assert.Equals(t, lastValue, 9999, "should have final value")
}

func TestTransactionalObservableAbort(t *testing.T) {
	obs := NewTransactional()

	var callCount int
	obs.OnKeyChange([]string{"foo"}, func(key string, oldValue, newValue any) {
		callCount++
	})

	obs.Txn(func(tx *Txn) {
		tx.SetValueAtKey("foo", "bar")
		tx.Abort()
	})

	assert.Equals(t, callCount, 0, "callback should not be called when transaction is aborted")
	assert.Nil(t, obs.GetValue("foo"), "value should not be set when transaction is aborted")
}

func TestTransactionalObservableAbortIgnoresSubsequentChanges(t *testing.T) {
	obs := NewTransactional()

	var callCount int
	obs.OnKeyChange([]string{"foo", "bar"}, func(key string, oldValue, newValue any) {
		callCount++
	})

	obs.Txn(func(tx *Txn) {
		tx.SetValueAtKey("foo", "value1")
		tx.Abort()
		tx.SetValueAtKey("bar", "value2") // Should be ignored
	})

	assert.Equals(t, callCount, 0, "no callbacks should be called")
	assert.Nil(t, obs.GetValue("foo"), "foo should not be set")
	assert.Nil(t, obs.GetValue("bar"), "bar should not be set")
}

// Tests for change deduplication (parent overrides children)

func TestKeyOverrides(t *testing.T) {
	// Test the keyOverrides helper function
	assert.True(t, keyOverrides("a", "a"), "same key should override")
	assert.True(t, keyOverrides("a", "a.1"), "parent should override child")
	assert.True(t, keyOverrides("a", "a.1.b"), "parent should override grandchild")
	assert.True(t, keyOverrides("a.1", "a.1.b"), "parent should override child")
	assert.True(t, keyOverrides("", "a"), "root should override everything")
	assert.True(t, keyOverrides("", "a.1.b"), "root should override everything")

	assert.False(t, keyOverrides("a.1", "a"), "child should not override parent")
	assert.False(t, keyOverrides("a.1", "a.2"), "sibling should not override sibling")
	assert.False(t, keyOverrides("a", "b"), "unrelated keys should not override")
	assert.False(t, keyOverrides("ab", "a"), "key starting with same chars but not prefix")
	assert.False(t, keyOverrides("a", "aa"), "key with same prefix but not child")
}

func TestTransactionalDeduplicationParentOverridesChild(t *testing.T) {
	obs := NewTransactional()

	// Track which keys were notified using specific patterns
	aNotified := false
	axNotified := false
	a1bNotified := false

	obs.OnKeyChange([]string{"a"}, func(key string, oldValue, newValue any) {
		aNotified = true
	})
	obs.OnKeyChange([]string{"a.x"}, func(key string, oldValue, newValue any) {
		axNotified = true
	})
	obs.OnKeyChange([]string{"a.1.b"}, func(key string, oldValue, newValue any) {
		a1bNotified = true
	})

	// Setting "a.1.b" then "a" - only "a" should be applied (a.1.b is overridden)
	obs.Txn(func(tx *Txn) {
		tx.SetValueAtKey("a.1.b", "value1")
		tx.SetValueAtKey("a", map[string]any{"x": "y"})
	})

	// "a.1.b" should be overridden by "a", so only "a" and "a.x" get notified
	assert.True(t, aNotified, "a should be notified")
	assert.True(t, axNotified, "a.x should be notified (new nested key)")
	assert.False(t, a1bNotified, "a.1.b should NOT be notified (overridden by a)")
}

func TestTransactionalDeduplicationComplexCase(t *testing.T) {
	obs := NewTransactional()

	// Use specific patterns to track exact keys
	cNotified := false
	aNotified := false
	var aValue any

	obs.OnKeyChange([]string{"c"}, func(key string, oldValue, newValue any) {
		cNotified = true
	})
	obs.OnKeyChange([]string{"a"}, func(key string, oldValue, newValue any) {
		aNotified = true
		aValue = newValue
	})

	// Example from spec: "a.1.b", "a", "a.2", "c", "a.1", "a" → only "c", "a" applied
	obs.Txn(func(tx *Txn) {
		tx.SetValueAtKey("a.1.b", "v1")
		tx.SetValueAtKey("a", map[string]any{"first": true})
		tx.SetValueAtKey("a.2", "v2")
		tx.SetValueAtKey("c", "v3")
		tx.SetValueAtKey("a.1", "v4")
		tx.SetValueAtKey("a", map[string]any{"final": true})
	})

	assert.True(t, cNotified, "c should be notified")
	assert.True(t, aNotified, "a should be notified")

	// Verify the final value of "a" is the last one set
	finalA := aValue.(map[string]any)
	assert.True(t, finalA["final"].(bool), "a should have final value")

	// Verify actual observable state
	assert.Equals(t, obs.GetValue("c"), "v3", "c should have value v3")
	assert.Nil(t, obs.GetValue("a.1.b"), "a.1.b should not exist (overridden)")
	assert.Nil(t, obs.GetValue("a.2"), "a.2 should not exist (overridden)")
}

func TestTransactionalDeduplicationRootOverridesAll(t *testing.T) {
	obs := NewTransactional()

	rootNotified := false
	obs.OnKeyChange([]string{""}, func(key string, oldValue, newValue any) {
		rootNotified = true
	})

	// Setting various keys, then setting root "" overrides all
	obs.Txn(func(tx *Txn) {
		tx.SetValueAtKey("a", "v1")
		tx.SetValueAtKey("b.c", "v2")
		tx.SetValueAtKey("d.e.f", "v3")
		tx.SetValueAtKey("", map[string]any{"root": "value"})
	})

	// Only the root change should be applied
	assert.True(t, rootNotified, "root should be notified")

	// Verify actual observable state - only root's value should exist
	assert.Nil(t, obs.GetValue("a"), "a should not exist (overridden by root)")
	assert.Nil(t, obs.GetValue("b.c"), "b.c should not exist (overridden by root)")
	assert.Equals(t, obs.GetValue("root"), "value", "root.root should have value")
}

func TestTransactionalDeduplicationVerifyState(t *testing.T) {
	obs := NewTransactional()

	// Test that deduplication produces correct final state
	obs.Txn(func(tx *Txn) {
		tx.SetValueAtKey("a", "1")
		tx.SetValueAtKey("b", "2")
		tx.SetValueAtKey("c", "3")
	})

	assert.Equals(t, obs.GetValue("a"), "1", "a should be 1")
	assert.Equals(t, obs.GetValue("b"), "2", "b should be 2")
	assert.Equals(t, obs.GetValue("c"), "3", "c should be 3")
}

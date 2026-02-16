package resolve

import (
	"encoding/json"
	"testing"

	"github.com/iancoleman/orderedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessExtraFields(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		include bool
		check   func(t *testing.T, result string)
	}{
		{
			name:    "object include=false strips properties adds count",
			input:   `{"name":"x","_additional_properties":{"a":1,"b":2}}`,
			include: false,
			check: func(t *testing.T, result string) {
				var m map[string]any
				require.NoError(t, json.Unmarshal([]byte(result), &m))
				assert.Equal(t, "x", m["name"])
				assert.Equal(t, float64(2), m["_additional_properties_count"])
				_, hasProps := m["_additional_properties"]
				assert.False(t, hasProps, "should not have _additional_properties")
			},
		},
		{
			name:    "object include=true keeps properties and adds count",
			input:   `{"name":"x","_additional_properties":{"a":1,"b":2}}`,
			include: true,
			check: func(t *testing.T, result string) {
				var m map[string]any
				require.NoError(t, json.Unmarshal([]byte(result), &m))
				assert.Equal(t, "x", m["name"])
				assert.Equal(t, float64(2), m["_additional_properties_count"])
				props, hasProps := m["_additional_properties"]
				assert.True(t, hasProps, "should have _additional_properties")
				propsMap, ok := props.(map[string]any)
				require.True(t, ok)
				assert.Len(t, propsMap, 2)
			},
		},
		{
			name:    "object without extra fields unchanged",
			input:   `{"name":"x"}`,
			include: false,
			check: func(t *testing.T, result string) {
				var m map[string]any
				require.NoError(t, json.Unmarshal([]byte(result), &m))
				assert.Equal(t, "x", m["name"])
				_, hasCount := m["_additional_properties_count"]
				assert.False(t, hasCount, "should not have count when no extra fields")
			},
		},
		{
			name:    "array include=false strips properties adds count",
			input:   `[{"_additional_properties":{"a":1}},{"_additional_properties":{"b":2,"c":3}}]`,
			include: false,
			check: func(t *testing.T, result string) {
				var items []map[string]any
				require.NoError(t, json.Unmarshal([]byte(result), &items))
				require.Len(t, items, 2)
				assert.Equal(t, float64(1), items[0]["_additional_properties_count"])
				_, hasProps0 := items[0]["_additional_properties"]
				assert.False(t, hasProps0)
				assert.Equal(t, float64(2), items[1]["_additional_properties_count"])
				_, hasProps1 := items[1]["_additional_properties"]
				assert.False(t, hasProps1)
			},
		},
		{
			name:    "array include=true keeps properties and adds count",
			input:   `[{"_additional_properties":{"a":1}},{"_additional_properties":{"b":2,"c":3}}]`,
			include: true,
			check: func(t *testing.T, result string) {
				var items []map[string]any
				require.NoError(t, json.Unmarshal([]byte(result), &items))
				require.Len(t, items, 2)
				assert.Equal(t, float64(1), items[0]["_additional_properties_count"])
				_, hasProps0 := items[0]["_additional_properties"]
				assert.True(t, hasProps0)
				assert.Equal(t, float64(2), items[1]["_additional_properties_count"])
				_, hasProps1 := items[1]["_additional_properties"]
				assert.True(t, hasProps1)
			},
		},
		{
			name:    "empty extra fields gives count 0",
			input:   `{"_additional_properties":{}}`,
			include: false,
			check: func(t *testing.T, result string) {
				var m map[string]any
				require.NoError(t, json.Unmarshal([]byte(result), &m))
				assert.Equal(t, float64(0), m["_additional_properties_count"])
				_, hasProps := m["_additional_properties"]
				assert.False(t, hasProps)
			},
		},
		{
			name:    "non-JSON input returned as-is",
			input:   `hello world`,
			include: false,
			check: func(t *testing.T, result string) {
				assert.Equal(t, "hello world", result)
			},
		},
		{
			name:    "quoted string returned as-is",
			input:   `"just a string"`,
			include: false,
			check: func(t *testing.T, result string) {
				assert.Equal(t, `"just a string"`, result)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ProcessExtraFields(tc.input, tc.include)
			require.NoError(t, err)
			tc.check(t, result)
		})
	}
}

func TestProcessExtraFields_InvalidArrayJSON(t *testing.T) {
	result, err := ProcessExtraFields(`[invalid`, false)
	assert.Error(t, err)
	assert.Equal(t, `[invalid`, result, "should return original on parse error")
}

func TestProcessExtraFields_InvalidObjectJSON(t *testing.T) {
	result, err := ProcessExtraFields(`{invalid`, false)
	assert.Error(t, err)
	assert.Equal(t, `{invalid`, result, "should return original on parse error")
}

func TestProcessExtraFieldsMap_PointerOrderedMap(t *testing.T) {
	om := orderedmap.New()
	child := orderedmap.New()
	child.Set("a", 1)
	child.Set("b", 2)
	om.Set("name", "test")
	om.Set(additionalPropertiesKey, child) // pointer type

	processExtraFieldsMap(om, false)

	count, exists := om.Get(additionalPropertiesCountKey)
	require.True(t, exists)
	assert.Equal(t, 2, count)
	_, hasProps := om.Get(additionalPropertiesKey)
	assert.False(t, hasProps)
}

func TestProcessExtraFieldsMap_MapStringAny(t *testing.T) {
	om := orderedmap.New()
	om.Set("name", "test")
	om.Set(additionalPropertiesKey, map[string]any{"a": 1, "b": 2, "c": 3})

	processExtraFieldsMap(om, false)

	count, exists := om.Get(additionalPropertiesCountKey)
	require.True(t, exists)
	assert.Equal(t, 3, count)
	_, hasProps := om.Get(additionalPropertiesKey)
	assert.False(t, hasProps)
}

func TestProcessExtraFieldsMap_NilPointerOrderedMap(t *testing.T) {
	om := orderedmap.New()
	var nilOM *orderedmap.OrderedMap
	om.Set(additionalPropertiesKey, nilOM)

	processExtraFieldsMap(om, false)

	count, exists := om.Get(additionalPropertiesCountKey)
	require.True(t, exists)
	assert.Equal(t, 0, count)
}

func TestProcessExtraFields_KeyOrder(t *testing.T) {
	input := `{"name":"x","_additional_properties":{"a":1,"b":2},"enabled":true}`

	result, err := ProcessExtraFields(input, false)
	require.NoError(t, err)

	keys := extractKeyOrder(t, result)
	// Count should appear where _additional_properties was
	expected := []string{"name", "_additional_properties_count", "enabled"}
	assert.Equal(t, expected, keys)
}

func TestProcessExtraFields_KeyOrderInclude(t *testing.T) {
	input := `{"name":"x","_additional_properties":{"a":1,"b":2},"enabled":true}`

	result, err := ProcessExtraFields(input, true)
	require.NoError(t, err)

	keys := extractKeyOrder(t, result)
	// Count comes first, then properties
	expected := []string{"name", "_additional_properties_count", "_additional_properties", "enabled"}
	assert.Equal(t, expected, keys)
}

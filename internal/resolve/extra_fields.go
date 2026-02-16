package resolve

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/iancoleman/orderedmap"
)

const (
	additionalPropertiesKey      = "_additional_properties"
	additionalPropertiesCountKey = "_additional_properties_count"
)

// ProcessExtraFields processes _additional_properties in JSON responses.
// It always injects _additional_properties_count. When include is false,
// the _additional_properties object itself is removed to save context.
func ProcessExtraFields(jsonStr string, include bool) (string, error) {
	trimmed := strings.TrimSpace(jsonStr)

	if strings.HasPrefix(trimmed, "[") {
		var items []*orderedmap.OrderedMap
		if err := json.Unmarshal([]byte(jsonStr), &items); err != nil {
			return jsonStr, fmt.Errorf("failed to parse JSON array: %w", err)
		}
		for _, item := range items {
			if item != nil {
				processExtraFieldsMap(item, include)
			}
		}
		result, err := json.MarshalIndent(items, "", "  ")
		if err != nil {
			return jsonStr, fmt.Errorf("failed to marshal JSON: %w", err)
		}
		return string(result), nil
	}

	if strings.HasPrefix(trimmed, "{") {
		om := orderedmap.New()
		if err := json.Unmarshal([]byte(jsonStr), om); err != nil {
			return jsonStr, fmt.Errorf("failed to parse JSON object: %w", err)
		}
		processExtraFieldsMap(om, include)
		result, err := json.MarshalIndent(om, "", "  ")
		if err != nil {
			return jsonStr, fmt.Errorf("failed to marshal JSON: %w", err)
		}
		return string(result), nil
	}

	return jsonStr, nil
}

func processExtraFieldsMap(om *orderedmap.OrderedMap, include bool) {
	value, exists := om.Get(additionalPropertiesKey)
	if !exists {
		return
	}

	// Count the extra fields
	count := 0
	switch v := value.(type) {
	case orderedmap.OrderedMap:
		count = len(v.Keys())
	case *orderedmap.OrderedMap:
		if v != nil {
			count = len(v.Keys())
		}
	case map[string]any:
		count = len(v)
	}

	// Rebuild ordered map: replace or remove _additional_properties,
	// insert _additional_properties_count right after (or in its place)
	newOM := orderedmap.New()
	for _, key := range om.Keys() {
		val, _ := om.Get(key)
		if key == additionalPropertiesKey {
			newOM.Set(additionalPropertiesCountKey, count)
			if include {
				newOM.Set(additionalPropertiesKey, val)
			}
		} else {
			newOM.Set(key, val)
		}
	}

	// Replace contents of original map
	oldKeys := make([]string, len(om.Keys()))
	copy(oldKeys, om.Keys())
	for _, key := range oldKeys {
		om.Delete(key)
	}
	for _, key := range newOM.Keys() {
		val, _ := newOM.Get(key)
		om.Set(key, val)
	}
}

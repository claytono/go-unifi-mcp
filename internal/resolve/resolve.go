// Package resolve provides ID-to-name resolution for UniFi MCP tool responses.
// When enabled, it scans JSON responses for _id/_ids fields, looks up referenced
// resources via the UniFi API, and injects human-readable _name/_names sibling fields.
package resolve

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"time"

	"github.com/iancoleman/orderedmap"
)

// Overrides maps non-standard field suffixes to their resource names.
// These fields don't follow the convention of foo_id -> Foo.
var Overrides = map[string]string{
	"networkconf_id":    "Network",
	"networkconf_ids":   "Network",
	"radiusprofile_id":  "RADIUSProfile",
	"firewallgroup_ids": "FirewallGroup",
	"firewall_group_id": "FirewallGroup",
}

// SkipFields contains field names that should never be resolved.
var SkipFields = map[string]bool{
	"_id":                         true,
	"attr_hidden_id":              true,
	"site_id":                     true,
	"ulp_user_id":                 true,
	"facebook_app_id":             true,
	"wechat_app_id":               true,
	"wechat_shop_id":              true,
	"google_client_id":            true,
	"facebook_wifi_gw_id":         true,
	"engine_id":                   true,
	"anqp_domain_id":              true,
	"roam_cluster_id":             true,
	"remote_site_id":              true,
	"sdwan_remote_site_id":        true,
	"virtual_network_override_id": true,
	"dev_id_override":             true,
	"filter_ids":                  true,
	"dismissed_ids":               true,
	"dpigroup_id":                 true,
}

// knownPrefixes are prefixes stripped from field names before resource lookup.
var knownPrefixes = []string{
	"igmp_proxy_downstream_",
	"multicast_router_",
	"dot1x_fallback_",
	"excluded_",
	"native_",
	"voice_",
	"src_",
	"dst_",
}

// Resolver resolves ID references in JSON responses to human-readable names.
type Resolver struct {
	client    any               // unifi.Client for reflection calls
	resources map[string]string // lowercase resource -> PascalCase
	logger    *slog.Logger
}

// New creates a new Resolver.
func New(client any, resources map[string]string, logger *slog.Logger) *Resolver {
	if logger == nil {
		logger = slog.New(discardHandler{})
	}
	return &Resolver{
		client:    client,
		resources: resources,
		logger:    logger,
	}
}

// discardHandler is an slog.Handler that discards all records.
type discardHandler struct{}

func (discardHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (d discardHandler) WithAttrs([]slog.Attr) slog.Handler      { return d }
func (d discardHandler) WithGroup(string) slog.Handler           { return d }

// requestCache holds per-request cached list results.
type requestCache struct {
	// resource -> id -> name
	data map[string]map[string]string
}

func newRequestCache() *requestCache {
	return &requestCache{data: make(map[string]map[string]string)}
}

// ResolveJSON takes a JSON string, resolves ID references, and returns the modified JSON.
// It preserves the original key order by using an ordered map for unmarshaling/marshaling.
func (r *Resolver) ResolveJSON(ctx context.Context, site, jsonStr string) (string, error) {
	start := time.Now()
	cache := newRequestCache()
	var fieldsResolved int

	trimmed := strings.TrimSpace(jsonStr)
	if strings.HasPrefix(trimmed, "[") {
		var items []*orderedmap.OrderedMap
		if err := json.Unmarshal([]byte(jsonStr), &items); err != nil {
			return jsonStr, fmt.Errorf("failed to parse JSON array: %w", err)
		}
		for _, item := range items {
			if item == nil {
				continue
			}
			fieldsResolved += r.resolveOrderedMap(ctx, site, item, cache)
		}
		result, err := json.MarshalIndent(items, "", "  ")
		if err != nil {
			return jsonStr, fmt.Errorf("failed to marshal resolved JSON: %w", err)
		}
		r.logger.Debug("resolve: completed",
			"fields_resolved", fieldsResolved,
			"duration", time.Since(start))
		return string(result), nil
	}

	if strings.HasPrefix(trimmed, "{") {
		om := orderedmap.New()
		if err := json.Unmarshal([]byte(jsonStr), om); err != nil {
			return jsonStr, fmt.Errorf("failed to parse JSON object: %w", err)
		}
		fieldsResolved = r.resolveOrderedMap(ctx, site, om, cache)
		result, err := json.MarshalIndent(om, "", "  ")
		if err != nil {
			return jsonStr, fmt.Errorf("failed to marshal resolved JSON: %w", err)
		}
		r.logger.Debug("resolve: completed",
			"fields_resolved", fieldsResolved,
			"duration", time.Since(start))
		return string(result), nil
	}

	return jsonStr, nil
}

// resolveOrderedMap resolves ID references in a single ordered map, inserting
// _name fields immediately after their corresponding _id fields.
// It recurses into nested objects and arrays.
func (r *Resolver) resolveOrderedMap(ctx context.Context, site string, om *orderedmap.OrderedMap, cache *requestCache) int {
	// First pass: collect resolutions keyed by the _id field name.
	type insertion struct {
		nameKey   string
		nameValue any
	}
	insertions := make(map[string]insertion)
	var nestedResolved int

	for _, key := range om.Keys() {
		value, _ := om.Get(key)

		// Recurse into nested objects and arrays.
		// orderedmap unmarshals nested objects as value types (orderedmap.OrderedMap),
		// so we handle both pointer and value types.
		switch nested := value.(type) {
		case *orderedmap.OrderedMap:
			if nested != nil {
				nestedResolved += r.resolveOrderedMap(ctx, site, nested, cache)
			}
		case orderedmap.OrderedMap:
			nestedResolved += r.resolveOrderedMap(ctx, site, &nested, cache)
			om.Set(key, nested)
		case []any:
			for i, elem := range nested {
				switch nestedOM := elem.(type) {
				case *orderedmap.OrderedMap:
					if nestedOM != nil {
						nestedResolved += r.resolveOrderedMap(ctx, site, nestedOM, cache)
					}
				case orderedmap.OrderedMap:
					nestedResolved += r.resolveOrderedMap(ctx, site, &nestedOM, cache)
					nested[i] = nestedOM
				}
			}
		}

		if !strings.HasSuffix(key, "_id") && !strings.HasSuffix(key, "_ids") {
			continue
		}

		resource, ok := r.ResourceForField(key)
		if !ok {
			continue
		}

		if strings.HasSuffix(key, "_ids") {
			ids, ok := value.([]any)
			if !ok || len(ids) == 0 {
				continue
			}
			names := make([]string, 0, len(ids))
			for _, idRaw := range ids {
				id, ok := idRaw.(string)
				if !ok {
					continue
				}
				name, err := r.lookupName(ctx, site, resource, id, cache)
				if err != nil || name == "" {
					continue
				}
				names = append(names, name)
			}
			if len(names) > 0 {
				nameKey := strings.TrimSuffix(key, "_ids") + "_names"
				insertions[key] = insertion{nameKey, names}
			}
		} else {
			id, ok := value.(string)
			if !ok || id == "" {
				continue
			}
			name, err := r.lookupName(ctx, site, resource, id, cache)
			if err != nil || name == "" {
				continue
			}
			nameKey := strings.TrimSuffix(key, "_id") + "_name"
			insertions[key] = insertion{nameKey, name}
		}
	}

	if len(insertions) == 0 {
		return nestedResolved
	}

	// Second pass: rebuild the ordered map with _name fields after their _id fields.
	newOM := orderedmap.New()
	for _, key := range om.Keys() {
		value, _ := om.Get(key)
		newOM.Set(key, value)
		if ins, ok := insertions[key]; ok {
			newOM.Set(ins.nameKey, ins.nameValue)
		}
	}

	// Replace contents of the original map.
	// Snapshot keys first since Delete mutates the underlying slice.
	oldKeys := make([]string, len(om.Keys()))
	copy(oldKeys, om.Keys())
	for _, key := range oldKeys {
		om.Delete(key)
	}
	for _, key := range newOM.Keys() {
		value, _ := newOM.Get(key)
		om.Set(key, value)
	}

	return len(insertions) + nestedResolved
}

// lookupName looks up the name for a resource ID, using the per-request cache.
func (r *Resolver) lookupName(ctx context.Context, site, resource, id string, cache *requestCache) (string, error) {
	if idMap, ok := cache.data[resource]; ok {
		return idMap[id], nil
	}

	// Fetch the list for this resource type
	start := time.Now()
	items, err := r.listResource(ctx, site, resource)
	if err != nil {
		// Cache empty map to avoid retrying on every field
		cache.data[resource] = make(map[string]string)
		return "", err
	}

	r.logger.Debug("resolve: fetched resource list",
		"resource", resource,
		"count", len(items),
		"duration", time.Since(start))

	// Build id -> name index
	idMap := make(map[string]string, len(items))
	for _, item := range items {
		itemID, _ := item["_id"].(string)
		if itemID == "" {
			continue
		}
		// Try "name" first, then "hostname"
		name, _ := item["name"].(string)
		if name == "" {
			name, _ = item["hostname"].(string)
		}
		if name != "" {
			idMap[itemID] = name
		}
	}
	cache.data[resource] = idMap

	return idMap[id], nil
}

// listResource calls List<Resource>(ctx, site) via reflection and returns the results as maps.
func (r *Resolver) listResource(ctx context.Context, site, resource string) ([]map[string]any, error) {
	methodName := "List" + resource
	clientVal := reflect.ValueOf(r.client)
	method := clientVal.MethodByName(methodName)
	if !method.IsValid() {
		return nil, fmt.Errorf("method %s not found", methodName)
	}

	// Verify method signature matches expected (ctx, site) -> (result, error)
	methodType := method.Type()
	if methodType.NumIn() != 2 || methodType.NumOut() != 2 {
		return nil, fmt.Errorf("method %s has unexpected signature: %d in, %d out", methodName, methodType.NumIn(), methodType.NumOut())
	}

	results := method.Call([]reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(site),
	})

	// Check error (second return value)
	if len(results) > 1 && !results[1].IsNil() {
		return nil, results[1].Interface().(error)
	}

	// Marshal then unmarshal to []map[string]any
	raw, err := json.Marshal(results[0].Interface())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal list results: %w", err)
	}

	var items []map[string]any
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list results: %w", err)
	}

	return items, nil
}

// ResourceForField returns the resource name for a given field name, or false if unresolvable.
func (r *Resolver) ResourceForField(fieldName string) (string, bool) {
	// Check skip list
	if SkipFields[fieldName] {
		return "", false
	}

	// Strip known prefixes
	suffix := fieldName
	for _, prefix := range knownPrefixes {
		if strings.HasPrefix(suffix, prefix) {
			suffix = strings.TrimPrefix(suffix, prefix)
			break
		}
	}

	// Check override table
	if resource, ok := Overrides[suffix]; ok {
		return resource, true
	}

	// Auto-resolve: strip _id/_ids suffix, snake_to_PascalCase, case-insensitive lookup
	var base string
	switch {
	case strings.HasSuffix(suffix, "_ids"):
		base = strings.TrimSuffix(suffix, "_ids")
	case strings.HasSuffix(suffix, "_id"):
		base = strings.TrimSuffix(suffix, "_id")
	default:
		return "", false
	}

	pascal := snakeToPascal(base)
	if resource, ok := r.resources[strings.ToLower(pascal)]; ok {
		return resource, true
	}

	return "", false
}

// snakeToPascal converts a snake_case string to PascalCase.
func snakeToPascal(s string) string {
	parts := strings.Split(s, "_")
	var b strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		b.WriteString(strings.ToUpper(part[:1]) + part[1:])
	}
	return b.String()
}

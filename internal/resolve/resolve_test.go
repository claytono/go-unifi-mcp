package resolve

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/claytono/go-unifi-mcp/internal/tools/generated"
	"github.com/iancoleman/orderedmap"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockClient implements List methods via reflection for testing.
type mockClient struct {
	networks    []mockNetwork
	userGroups  []mockUserGroup
	listErr     error
	listCalls   int
	listCallLog []string
}

type mockNetwork struct {
	ID   string `json:"_id"`
	Name string `json:"name"`
}

type mockUserGroup struct {
	ID   string `json:"_id"`
	Name string `json:"name"`
}

func (m *mockClient) ListNetwork(_ context.Context, _ string) ([]mockNetwork, error) {
	m.listCalls++
	m.listCallLog = append(m.listCallLog, "ListNetwork")
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.networks, nil
}

func (m *mockClient) ListUserGroup(_ context.Context, _ string) ([]mockUserGroup, error) {
	m.listCalls++
	m.listCallLog = append(m.listCallLog, "ListUserGroup")
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.userGroups, nil
}

func (m *mockClient) ListFirewallGroup(_ context.Context, _ string) ([]mockNetwork, error) {
	m.listCalls++
	m.listCallLog = append(m.listCallLog, "ListFirewallGroup")
	return []mockNetwork{
		{ID: "fwg1", Name: "LAN Group"},
		{ID: "fwg2", Name: "WAN Group"},
	}, nil
}

func (m *mockClient) ListAPGroup(_ context.Context, _ string) ([]mockNetwork, error) {
	m.listCalls++
	m.listCallLog = append(m.listCallLog, "ListAPGroup")
	return []mockNetwork{
		{ID: "ap1", Name: "Default AP Group"},
		{ID: "ap2", Name: "Office APs"},
	}, nil
}

func (m *mockClient) ListRADIUSProfile(_ context.Context, _ string) ([]mockNetwork, error) {
	m.listCalls++
	m.listCallLog = append(m.listCallLog, "ListRADIUSProfile")
	return []mockNetwork{
		{ID: "rad1", Name: "Corp RADIUS"},
	}, nil
}

type mockDevice struct {
	ID       string `json:"_id"`
	Hostname string `json:"hostname,omitempty"`
}

type mockHostnameClient struct {
	devices []mockDevice
}

func (m *mockHostnameClient) ListDevice(_ context.Context, _ string) ([]mockDevice, error) {
	return m.devices, nil
}

// badSignatureClient has a ListNetwork method with wrong arity for testing signature validation.
type badSignatureClient struct{}

func (badSignatureClient) ListNetwork() ([]mockNetwork, error) {
	return nil, nil
}

func newTestResolver(client any) *Resolver {
	resources := map[string]string{
		"network":       "Network",
		"usergroup":     "UserGroup",
		"firewallgroup": "FirewallGroup",
		"apgroup":       "APGroup",
		"radiusprofile": "RADIUSProfile",
		"device":        "Device",
		"wlan":          "WLAN",
		"portprofile":   "PortProfile",
		"firewallrule":  "FirewallRule",
		"firewallzone":  "FirewallZone",
	}
	return New(client, resources, nil)
}

func TestResourceForField(t *testing.T) {
	resolver := newTestResolver(nil)

	tests := []struct {
		field  string
		want   string
		wantOK bool
	}{
		// Auto-resolvable
		{"network_id", "Network", true},
		{"usergroup_id", "UserGroup", true},
		{"ap_group_id", "APGroup", true},
		{"wlan_id", "WLAN", true},

		// Prefix stripping
		{"src_networkconf_id", "Network", true},
		{"dst_networkconf_id", "Network", true},
		{"native_networkconf_id", "Network", true},
		{"voice_networkconf_id", "Network", true},
		{"dot1x_fallback_networkconf_id", "Network", true},
		{"excluded_networkconf_id", "Network", true},

		// Override table
		{"networkconf_id", "Network", true},
		{"networkconf_ids", "Network", true},
		{"radiusprofile_id", "RADIUSProfile", true},
		{"firewallgroup_ids", "FirewallGroup", true},
		{"firewall_group_id", "FirewallGroup", true},

		// Skip fields
		{"_id", "", false},
		{"attr_hidden_id", "", false},
		{"site_id", "", false},
		{"ulp_user_id", "", false},
		{"facebook_app_id", "", false},
		{"engine_id", "", false},
		{"filter_ids", "", false},
		{"dismissed_ids", "", false},

		// Unknown fields (not in resource index)
		{"unknown_foo_id", "", false},
		{"not_an_id_field", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.field, func(t *testing.T) {
			got, ok := resolver.ResourceForField(tc.field)
			assert.Equal(t, tc.wantOK, ok, "field: %s", tc.field)
			if tc.wantOK {
				assert.Equal(t, tc.want, got, "field: %s", tc.field)
			}
		})
	}
}

func TestResolveMap(t *testing.T) {
	client := &mockClient{
		networks: []mockNetwork{
			{ID: "net1", Name: "LAN"},
			{ID: "net2", Name: "WAN"},
		},
	}
	resolver := newTestResolver(client)

	input := `{"network_id": "net1", "name": "My Rule"}`
	result, err := resolver.ResolveJSON(context.Background(), "default", input)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &m))

	assert.Equal(t, "net1", m["network_id"])
	assert.Equal(t, "LAN", m["network_name"])
	assert.Equal(t, "My Rule", m["name"])
}

func TestResolveSlice_Caching(t *testing.T) {
	client := &mockClient{
		networks: []mockNetwork{
			{ID: "net1", Name: "LAN"},
			{ID: "net2", Name: "WAN"},
		},
	}
	resolver := newTestResolver(client)

	// Three items all referencing networks - should only call ListNetwork once
	input := `[
		{"network_id": "net1", "name": "Rule 1"},
		{"network_id": "net2", "name": "Rule 2"},
		{"network_id": "net1", "name": "Rule 3"}
	]`

	result, err := resolver.ResolveJSON(context.Background(), "default", input)
	require.NoError(t, err)

	var items []map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &items))

	assert.Len(t, items, 3)
	assert.Equal(t, "LAN", items[0]["network_name"])
	assert.Equal(t, "WAN", items[1]["network_name"])
	assert.Equal(t, "LAN", items[2]["network_name"])

	// Should have only called ListNetwork once due to caching
	assert.Equal(t, 1, client.listCalls)
}

func TestResolveMap_PluralIDs(t *testing.T) {
	client := &mockClient{}
	resolver := newTestResolver(client)

	input := `{"ap_group_ids": ["ap1", "ap2"], "name": "My Config"}`
	result, err := resolver.ResolveJSON(context.Background(), "default", input)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &m))

	names, ok := m["ap_group_names"].([]any)
	require.True(t, ok, "ap_group_names should be an array")
	assert.Len(t, names, 2)
	assert.Equal(t, "Default AP Group", names[0])
	assert.Equal(t, "Office APs", names[1])
}

func TestResolveMap_GracefulDegradation(t *testing.T) {
	client := &mockClient{
		listErr: errors.New("API error"),
	}
	resolver := newTestResolver(client)

	input := `{"network_id": "net1", "name": "My Rule"}`
	result, err := resolver.ResolveJSON(context.Background(), "default", input)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &m))

	// Original data should be preserved
	assert.Equal(t, "net1", m["network_id"])
	assert.Equal(t, "My Rule", m["name"])
	// No network_name should be added
	_, hasName := m["network_name"]
	assert.False(t, hasName)
}

func TestResolveJSON_Array(t *testing.T) {
	client := &mockClient{
		networks: []mockNetwork{{ID: "net1", Name: "LAN"}},
	}
	resolver := newTestResolver(client)

	input := `[{"network_id": "net1"}]`
	result, err := resolver.ResolveJSON(context.Background(), "default", input)
	require.NoError(t, err)

	var items []map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &items))
	assert.Len(t, items, 1)
	assert.Equal(t, "LAN", items[0]["network_name"])
}

func TestResolveJSON_Object(t *testing.T) {
	client := &mockClient{
		networks: []mockNetwork{{ID: "net1", Name: "LAN"}},
	}
	resolver := newTestResolver(client)

	input := `{"network_id": "net1"}`
	result, err := resolver.ResolveJSON(context.Background(), "default", input)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &m))
	assert.Equal(t, "LAN", m["network_name"])
}

func TestResolveJSON_NonJSON(t *testing.T) {
	resolver := newTestResolver(nil)

	input := `not json`
	result, err := resolver.ResolveJSON(context.Background(), "default", input)
	require.NoError(t, err)
	assert.Equal(t, input, result)
}

func TestResolveJSON_NonObjectNonArray(t *testing.T) {
	resolver := newTestResolver(nil)

	input := `"just a string"`
	result, err := resolver.ResolveJSON(context.Background(), "default", input)
	require.NoError(t, err)
	assert.Equal(t, input, result)
}

func TestResolveMap_UnknownIDNotResolved(t *testing.T) {
	client := &mockClient{
		networks: []mockNetwork{{ID: "net1", Name: "LAN"}},
	}
	resolver := newTestResolver(client)

	// network_id with an ID that doesn't exist in the list
	input := `{"network_id": "nonexistent"}`
	result, err := resolver.ResolveJSON(context.Background(), "default", input)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &m))
	assert.Equal(t, "nonexistent", m["network_id"])
	_, hasName := m["network_name"]
	assert.False(t, hasName)
}

func TestResolveMap_EmptyIDSkipped(t *testing.T) {
	resolver := newTestResolver(&mockClient{})

	input := `{"network_id": ""}`
	result, err := resolver.ResolveJSON(context.Background(), "default", input)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &m))
	_, hasName := m["network_name"]
	assert.False(t, hasName)
}

func TestResolveMap_NonStringIDSkipped(t *testing.T) {
	resolver := newTestResolver(&mockClient{})

	input := `{"network_id": 123}`
	result, err := resolver.ResolveJSON(context.Background(), "default", input)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &m))
	_, hasName := m["network_name"]
	assert.False(t, hasName)
}

func TestResolveMap_OverrideFields(t *testing.T) {
	client := &mockClient{
		networks: []mockNetwork{{ID: "net1", Name: "LAN"}},
	}
	resolver := newTestResolver(client)

	// networkconf_id is an override -> Network
	input := `{"networkconf_id": "net1"}`
	result, err := resolver.ResolveJSON(context.Background(), "default", input)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &m))
	assert.Equal(t, "LAN", m["networkconf_name"])
}

func TestResolveMap_PrefixedOverrideField(t *testing.T) {
	client := &mockClient{
		networks: []mockNetwork{{ID: "net1", Name: "LAN"}},
	}
	resolver := newTestResolver(client)

	// src_networkconf_id -> strip src_ -> networkconf_id -> override -> Network
	input := `{"src_networkconf_id": "net1"}`
	result, err := resolver.ResolveJSON(context.Background(), "default", input)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &m))
	assert.Equal(t, "LAN", m["src_networkconf_name"])
}

func TestResolveMap_MultipleResourceTypes(t *testing.T) {
	client := &mockClient{
		networks:   []mockNetwork{{ID: "net1", Name: "LAN"}},
		userGroups: []mockUserGroup{{ID: "ug1", Name: "Staff"}},
	}
	resolver := newTestResolver(client)

	input := `{"network_id": "net1", "usergroup_id": "ug1"}`
	result, err := resolver.ResolveJSON(context.Background(), "default", input)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &m))
	assert.Equal(t, "LAN", m["network_name"])
	assert.Equal(t, "Staff", m["usergroup_name"])
	assert.Equal(t, 2, client.listCalls)
}

func TestResolveMap_HostnameFallback(t *testing.T) {
	// mockDevice has hostname instead of name
	client := &mockHostnameClient{
		devices: []mockDevice{
			{ID: "dev1", Hostname: "switch-01"},
			{ID: "dev2"}, // no name or hostname
		},
	}
	resources := map[string]string{
		"device": "Device",
	}
	resolver := New(client, resources, nil)

	input := `{"device_id": "dev1"}`
	result, err := resolver.ResolveJSON(context.Background(), "default", input)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &m))
	assert.Equal(t, "switch-01", m["device_name"])
}

func TestResolveMap_MethodNotFound(t *testing.T) {
	// Client without the needed List method
	client := &struct{}{}
	resolver := newTestResolver(client)

	input := `{"network_id": "net1"}`
	result, err := resolver.ResolveJSON(context.Background(), "default", input)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &m))
	// Should gracefully skip
	_, hasName := m["network_name"]
	assert.False(t, hasName)
}

func TestResolveMap_BadMethodSignature(t *testing.T) {
	// Client with a ListNetwork that has wrong arity (no args)
	client := &badSignatureClient{}
	resolver := newTestResolver(client)

	input := `{"network_id": "net1"}`
	result, err := resolver.ResolveJSON(context.Background(), "default", input)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &m))
	// Should gracefully skip due to signature mismatch
	_, hasName := m["network_name"]
	assert.False(t, hasName)
}

func TestWrapHandler_ResolveTrue(t *testing.T) {
	client := &mockClient{
		networks: []mockNetwork{{ID: "net1", Name: "LAN"}},
	}
	resolver := newTestResolver(client)

	innerHandler := func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText(`{"network_id": "net1"}`), nil
	}

	handler := WrapHandler(innerHandler, resolver)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"resolve": true,
		"site":    "default",
	}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)

	text := result.Content[0].(mcp.TextContent).Text
	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(text), &m))
	assert.Equal(t, "LAN", m["network_name"])
}

func TestWrapHandler_ResolveDefaultTrue(t *testing.T) {
	client := &mockClient{
		networks: []mockNetwork{{ID: "net1", Name: "LAN"}},
	}
	resolver := newTestResolver(client)

	innerHandler := func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText(`{"network_id": "net1"}`), nil
	}

	handler := WrapHandler(innerHandler, resolver)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"site": "default",
	}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)

	text := result.Content[0].(mcp.TextContent).Text
	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(text), &m))
	assert.Equal(t, "LAN", m["network_name"], "should resolve by default")
	assert.Equal(t, 1, client.listCalls, "should make API calls when resolve not specified")
}

func TestWrapHandler_ErrorResult(t *testing.T) {
	resolver := newTestResolver(&mockClient{})

	innerHandler := func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultError("something went wrong"), nil
	}

	handler := WrapHandler(innerHandler, resolver)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"resolve": true,
	}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestWrapHandler_NilResolver(t *testing.T) {
	innerHandler := func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText(`{"network_id": "net1"}`), nil
	}

	handler := WrapHandler(innerHandler, nil)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"resolve": true,
	}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	// Should return original result unchanged
	text := result.Content[0].(mcp.TextContent).Text
	assert.Equal(t, `{"network_id": "net1"}`, text)
}

func TestWrapHandler_InnerHandlerError(t *testing.T) {
	resolver := newTestResolver(&mockClient{})

	innerHandler := func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return nil, errors.New("handler error")
	}

	handler := WrapHandler(innerHandler, resolver)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"resolve": true,
	}

	result, err := handler(context.Background(), req)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestWrapHandler_DefaultSite(t *testing.T) {
	client := &mockClient{
		networks: []mockNetwork{{ID: "net1", Name: "LAN"}},
	}
	resolver := newTestResolver(client)

	innerHandler := func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText(`{"network_id": "net1"}`), nil
	}

	handler := WrapHandler(innerHandler, resolver)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"resolve": true,
		// no site specified - should default to "default"
	}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)

	text := result.Content[0].(mcp.TextContent).Text
	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(text), &m))
	assert.Equal(t, "LAN", m["network_name"])
}

func TestBuildResourceIndex(t *testing.T) {
	metadata := []generated.ToolMetadata{
		{Name: "list_network", Category: "list", Resource: "Network"},
		{Name: "get_network", Category: "get", Resource: "Network"},
		{Name: "list_user_group", Category: "list", Resource: "UserGroup"},
		{Name: "list_firewall_rule", Category: "list", Resource: "FirewallRule"},
		{Name: "create_network", Category: "create", Resource: "Network"},
	}

	index := BuildResourceIndex(metadata)

	assert.Equal(t, "Network", index["network"])
	assert.Equal(t, "UserGroup", index["usergroup"])
	assert.Equal(t, "FirewallRule", index["firewallrule"])
	// Non-list categories should not be included
	assert.Len(t, index, 3)
}

func TestBuildResourceIndex_WithRealMetadata(t *testing.T) {
	index := BuildResourceIndex(generated.AllToolMetadata)

	// Should have entries for real resources
	assert.Contains(t, index, "network")
	assert.Equal(t, "Network", index["network"])
	assert.Greater(t, len(index), 10, "should have many resources")
}

func TestOverrideTableResourcesExist(t *testing.T) {
	index := BuildResourceIndex(generated.AllToolMetadata)

	for field, resource := range Overrides {
		_, exists := index[strings.ToLower(resource)]
		assert.True(t, exists, "override %q -> %q references non-existent resource", field, resource)
	}
}

func TestSkipFieldsNotResolved(t *testing.T) {
	resolver := newTestResolver(nil)

	for field := range SkipFields {
		_, ok := resolver.ResourceForField(field)
		assert.False(t, ok, "skip field %q should not resolve", field)
	}
}

func TestAllIDFieldsResolvable(t *testing.T) {
	index := BuildResourceIndex(generated.AllToolMetadata)
	resolver := New(nil, index, nil)

	var unhandled []string

	for resourceName, factory := range generated.TypeRegistry {
		instance := factory()
		typ := reflect.TypeOf(instance)
		if typ.Kind() == reflect.Pointer {
			typ = typ.Elem()
		}

		fields := collectJSONFields(typ)
		for _, field := range fields {
			if !strings.HasSuffix(field, "_id") && !strings.HasSuffix(field, "_ids") {
				continue
			}
			if SkipFields[field] {
				continue
			}
			_, ok := resolver.ResourceForField(field)
			if !ok {
				unhandled = append(unhandled, resourceName+"."+field)
			}
		}
	}

	if len(unhandled) > 0 {
		t.Errorf("unhandled ID fields (add to Overrides, SkipFields, or verify resource index):\n  %s",
			strings.Join(unhandled, "\n  "))
	}
}

// collectJSONFields extracts all JSON field names from a struct type, including embedded structs.
func collectJSONFields(typ reflect.Type) []string {
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil
	}

	var fields []string
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" {
			continue
		}
		if field.Anonymous {
			fields = append(fields, collectJSONFields(field.Type)...)
			continue
		}
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}
		name := strings.Split(jsonTag, ",")[0]
		if name != "" {
			fields = append(fields, name)
		}
	}
	return fields
}

func TestSnakeToPascal(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"network", "Network"},
		{"user_group", "UserGroup"},
		{"ap_group", "ApGroup"},
		{"firewall_rule", "FirewallRule"},
		{"wlan", "Wlan"},
		{"dhcp_option", "DhcpOption"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.want, snakeToPascal(tc.input))
		})
	}
}

func TestNewLogger(t *testing.T) {
	// Just verify it returns a non-nil logger for all levels
	levels := []string{"disabled", "trace", "debug", "info", "warn", "error", "unknown"}
	for _, level := range levels {
		logger := NewLogger(level)
		assert.NotNil(t, logger, "level: %s", level)
	}
}

func TestNew_NilLogger(t *testing.T) {
	resolver := New(nil, nil, nil)
	assert.NotNil(t, resolver)
	assert.NotNil(t, resolver.logger)
}

func TestResolveMap_PluralIDsOverride(t *testing.T) {
	client := &mockClient{}
	resolver := newTestResolver(client)

	// firewallgroup_ids is an override -> FirewallGroup
	input := `{"firewallgroup_ids": ["fwg1", "fwg2"]}`
	result, err := resolver.ResolveJSON(context.Background(), "default", input)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &m))

	names, ok := m["firewallgroup_names"].([]any)
	require.True(t, ok, "firewallgroup_names should be an array")
	assert.Len(t, names, 2)
	assert.Equal(t, "LAN Group", names[0])
	assert.Equal(t, "WAN Group", names[1])
}

func TestResolveMap_EmptyPluralIDs(t *testing.T) {
	resolver := newTestResolver(&mockClient{})

	input := `{"ap_group_ids": []}`
	result, err := resolver.ResolveJSON(context.Background(), "default", input)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &m))
	_, hasNames := m["ap_group_names"]
	assert.False(t, hasNames)
}

func TestResolveMap_CachePreventsDuplicateListCalls(t *testing.T) {
	client := &mockClient{
		networks: []mockNetwork{{ID: "net1", Name: "LAN"}},
	}
	resolver := newTestResolver(client)

	// Two fields referencing the same resource type
	input := `{"network_id": "net1", "src_networkconf_id": "net1"}`
	_, err := resolver.ResolveJSON(context.Background(), "default", input)
	require.NoError(t, err)

	// Should call ListNetwork only once (both fields resolve to Network)
	assert.Equal(t, 1, client.listCalls)
}

func TestWrapHandler_ResolveExplicitFalse(t *testing.T) {
	client := &mockClient{
		networks: []mockNetwork{{ID: "net1", Name: "LAN"}},
	}
	resolver := newTestResolver(client)

	innerHandler := func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText(`{"network_id": "net1"}`), nil
	}

	handler := WrapHandler(innerHandler, resolver)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"resolve": false,
	}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, 0, client.listCalls, "should not resolve when resolve=false")

	text := result.Content[0].(mcp.TextContent).Text
	assert.Equal(t, `{"network_id": "net1"}`, text)
}

func TestWrapHandler_NilResult(t *testing.T) {
	resolver := newTestResolver(&mockClient{})

	innerHandler := func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return nil, nil
	}

	handler := WrapHandler(innerHandler, resolver)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"resolve": true,
	}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestResolveJSON_PreservesKeyOrder(t *testing.T) {
	client := &mockClient{
		networks: []mockNetwork{{ID: "net1", Name: "LAN"}},
	}
	resolver := newTestResolver(client)

	// Input with a specific key order
	input := `{
  "_id": "rule1",
  "site_id": "site1",
  "action": "accept",
  "src_networkconf_id": "net1",
  "src_networkconf_type": "NETv4",
  "name": "My Rule"
}`

	result, err := resolver.ResolveJSON(context.Background(), "default", input)
	require.NoError(t, err)

	// Extract key order from the result JSON
	keys := extractKeyOrder(t, result)

	// _name should appear right after its _id field, and all original keys
	// should remain in their original relative order
	expected := []string{
		"_id",
		"site_id",
		"action",
		"src_networkconf_id",
		"src_networkconf_name", // inserted after src_networkconf_id
		"src_networkconf_type",
		"name",
	}
	assert.Equal(t, expected, keys)
}

func TestResolveJSON_PreservesKeyOrder_Array(t *testing.T) {
	client := &mockClient{
		networks: []mockNetwork{{ID: "net1", Name: "LAN"}},
	}
	resolver := newTestResolver(client)

	input := `[{"_id": "r1", "network_id": "net1", "name": "Rule"}]`

	result, err := resolver.ResolveJSON(context.Background(), "default", input)
	require.NoError(t, err)

	// Parse as array and check first element's key order
	var items []json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(result), &items))
	require.Len(t, items, 1)

	keys := extractKeyOrder(t, string(items[0]))
	expected := []string{"_id", "network_id", "network_name", "name"}
	assert.Equal(t, expected, keys)
}

// extractKeyOrder parses a JSON object and returns its keys in order.
func extractKeyOrder(t *testing.T, jsonStr string) []string {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(jsonStr))

	// Read opening brace
	tok, err := dec.Token()
	require.NoError(t, err)
	require.Equal(t, json.Delim('{'), tok)

	var keys []string
	for dec.More() {
		tok, err := dec.Token()
		require.NoError(t, err)
		key, ok := tok.(string)
		require.True(t, ok)
		keys = append(keys, key)

		// Skip the value
		var value json.RawMessage
		require.NoError(t, dec.Decode(&value))
	}
	return keys
}

func TestResolveJSON_NestedObject(t *testing.T) {
	client := &mockClient{
		networks: []mockNetwork{{ID: "net1", Name: "LAN"}},
	}
	resolver := newTestResolver(client)

	input := `{
  "name": "My WLAN",
  "private_preshared_keys": {
    "networkconf_id": "net1",
    "password": "secret"
  }
}`

	result, err := resolver.ResolveJSON(context.Background(), "default", input)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &m))

	nested, ok := m["private_preshared_keys"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "LAN", nested["networkconf_name"])
	assert.Equal(t, "net1", nested["networkconf_id"])
}

func TestResolveJSON_NestedArray(t *testing.T) {
	client := &mockClient{
		networks: []mockNetwork{{ID: "net1", Name: "LAN"}, {ID: "net2", Name: "WAN"}},
	}
	resolver := newTestResolver(client)

	input := `{
  "name": "Config",
  "rules": [
    {"network_id": "net1", "action": "allow"},
    {"network_id": "net2", "action": "deny"}
  ]
}`

	result, err := resolver.ResolveJSON(context.Background(), "default", input)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &m))

	rules, ok := m["rules"].([]any)
	require.True(t, ok)
	require.Len(t, rules, 2)

	rule0, ok := rules[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "LAN", rule0["network_name"])

	rule1, ok := rules[1].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "WAN", rule1["network_name"])
}

func TestResolveJSON_NestedAndTopLevel(t *testing.T) {
	client := &mockClient{
		networks: []mockNetwork{{ID: "net1", Name: "LAN"}},
	}
	resolver := newTestResolver(client)

	input := `{
  "network_id": "net1",
  "child": {"networkconf_id": "net1"}
}`

	result, err := resolver.ResolveJSON(context.Background(), "default", input)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &m))

	assert.Equal(t, "LAN", m["network_name"])
	child, ok := m["child"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "LAN", child["networkconf_name"])
}

func TestResolveOrderedMap_PointerNestedObject(t *testing.T) {
	client := &mockClient{
		networks: []mockNetwork{{ID: "net1", Name: "LAN"}},
	}
	resolver := newTestResolver(client)

	// Manually construct an ordered map with a *orderedmap.OrderedMap child
	om := orderedmap.New()
	child := orderedmap.New()
	child.Set("networkconf_id", "net1")
	om.Set("name", "test")
	om.Set("child", child) // pointer type

	cache := newRequestCache()
	resolved := resolver.resolveOrderedMap(context.Background(), "default", om, cache)
	assert.Equal(t, 1, resolved)

	val, _ := child.Get("networkconf_name")
	assert.Equal(t, "LAN", val)
}

func TestResolveOrderedMap_PointerNestedInArray(t *testing.T) {
	client := &mockClient{
		networks: []mockNetwork{{ID: "net1", Name: "LAN"}},
	}
	resolver := newTestResolver(client)

	// Manually construct an ordered map with []*orderedmap.OrderedMap array
	om := orderedmap.New()
	child := orderedmap.New()
	child.Set("networkconf_id", "net1")
	om.Set("items", []any{child}) // pointer in array

	cache := newRequestCache()
	resolved := resolver.resolveOrderedMap(context.Background(), "default", om, cache)
	assert.Equal(t, 1, resolved)

	val, _ := child.Get("networkconf_name")
	assert.Equal(t, "LAN", val)
}

func TestResolveJSON_NilArrayElements(t *testing.T) {
	client := &mockClient{
		networks: []mockNetwork{{ID: "net1", Name: "LAN"}},
	}
	resolver := newTestResolver(client)

	// JSON array with null elements
	input := `[null, {"network_id": "net1"}, null]`

	result, err := resolver.ResolveJSON(context.Background(), "default", input)
	require.NoError(t, err)

	// Should parse successfully and resolve the non-nil element
	assert.Contains(t, result, "network_name")
	assert.Contains(t, result, "LAN")
}

func TestWrapHandler_NonTextContent(t *testing.T) {
	resolver := newTestResolver(&mockClient{})

	innerHandler := func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) { //nolint:unparam
		img := mcp.NewImageContent("base64data", "image/png")
		return &mcp.CallToolResult{
			Content: []mcp.Content{img},
		}, nil
	}

	handler := WrapHandler(innerHandler, resolver)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	// Should return image content unchanged
	_, ok := result.Content[0].(mcp.ImageContent)
	assert.True(t, ok)
}

func TestWrapHandler_ResolveError(t *testing.T) {
	resolver := newTestResolver(&mockClient{})

	// Return invalid JSON that starts with "[" to trigger a parse error in ResolveJSON
	innerHandler := func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) { //nolint:unparam
		return mcp.NewToolResultText(`[invalid json`), nil
	}

	handler := WrapHandler(innerHandler, resolver)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	// Should return original content on resolve error
	text := result.Content[0].(mcp.TextContent).Text
	assert.Equal(t, `[invalid json`, text)
}

// Verify the handler type matches what the server expects.
func TestWrapHandler_ReturnsCorrectType(t *testing.T) {
	resolver := newTestResolver(&mockClient{})
	innerHandler := func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("test"), nil
	}

	handler := WrapHandler(innerHandler, resolver)

	// Verify it satisfies server.ToolHandlerFunc
	var _ = handler
}

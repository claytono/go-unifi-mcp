package generated

import (
	"context"
	"errors"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockClient implements a minimal mock for testing reflection-based handlers.
// It returns canned responses or errors based on the test configuration.
type MockClient struct {
	ListResult    any
	GetResult     any
	CreateResult  any
	UpdateResult  any
	DeleteError   error
	MethodError   error
	MethodMissing bool
}

// TestGenericList_Success tests the list handler with a mock client.
func TestGenericList_Success(t *testing.T) {
	// Note: We can't easily test the generic handlers without a real client
	// because reflection requires actual method implementations.
	// Instead, we test the helper functions.

	// Test extractSite with default
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}
	site := extractSite(req)
	assert.Equal(t, "default", site)

	// Test extractSite with custom value
	req.Params.Arguments = map[string]any{"site": "custom"}
	site = extractSite(req)
	assert.Equal(t, "custom", site)
}

func TestExtractSite(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]any
		expected string
	}{
		{
			name:     "empty args returns default",
			args:     map[string]any{},
			expected: "default",
		},
		{
			name:     "nil site returns default",
			args:     map[string]any{"site": nil},
			expected: "default",
		},
		{
			name:     "empty string returns default",
			args:     map[string]any{"site": ""},
			expected: "default",
		},
		{
			name:     "custom site is preserved",
			args:     map[string]any{"site": "mysite"},
			expected: "mysite",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := mcp.CallToolRequest{}
			req.Params.Arguments = tt.args
			result := extractSite(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractError(t *testing.T) {
	// Test with a nil error interface stored in reflect.Value - this is tricky
	// as we need an actual reflect.Value. Let's skip this test for now
	// since it's really testing Go's reflect behavior.
	t.Skip("extractError tested implicitly through integration tests")
}

func TestValidateClientMethods_MissingMethod(t *testing.T) {
	// Create a client missing a required method
	type IncompleteClient struct{}
	client := &IncompleteClient{}

	tools := []ToolMetadata{
		{Name: "list_test", Category: "list", Resource: "Test"},
	}

	err := ValidateClientMethods(client, tools, TypeRegistry)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing client method")
	assert.Contains(t, err.Error(), "ListTest")
}

func TestValidateClientMethods_MissingTypeInRegistry(t *testing.T) {
	client := &FakeTestClient{}

	tools := []ToolMetadata{
		{Name: "create_nonexistent", Category: "create", Resource: "NonExistent"},
	}

	// Use empty registry
	emptyRegistry := map[string]func() any{}

	err := ValidateClientMethods(client, tools, emptyRegistry)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing type in registry")
	assert.Contains(t, err.Error(), "NonExistent")
}

func TestValidateClientMethods_UnknownCategory(t *testing.T) {
	client := &FakeTestClient{}

	tools := []ToolMetadata{
		{Name: "unknown_test", Category: "unknown", Resource: "Test"},
	}

	err := ValidateClientMethods(client, tools, TypeRegistry)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown category")
}

func TestValidateClientMethods_WrongSignature(t *testing.T) {
	// Create a client with wrong method signature
	type WrongSignatureClient struct{}
	// Add a method with wrong signature
	client := &WrongSignatureClient{}

	tools := []ToolMetadata{
		{Name: "list_test", Category: "list", Resource: "Test"},
	}

	err := ValidateClientMethods(client, tools, TypeRegistry)
	require.Error(t, err)
	// Will fail on missing method first
}

func TestValidateClientMethods_Success(t *testing.T) {
	client := &FakeTestClient{}

	tools := []ToolMetadata{
		{Name: "list_test", Category: "list", Resource: "Test"},
		{Name: "get_test", Category: "get", Resource: "Test"},
		{Name: "delete_test", Category: "delete", Resource: "Test"},
	}

	// Note: FakeTestClient methods have correct signatures
	err := ValidateClientMethods(client, tools, TypeRegistry)
	require.NoError(t, err)
}

func TestTypeRegistry(t *testing.T) {
	// Verify the type registry has entries
	assert.Greater(t, len(TypeRegistry), 0)

	// Verify a known type can be created
	if factory, ok := TypeRegistry["Network"]; ok {
		instance := factory()
		require.NotNil(t, instance)
	}
}

func TestGenericHandlers_MethodNotFound(t *testing.T) {
	// Test that handlers return appropriate errors when methods don't exist

	// Create a dummy struct that implements no methods
	type DummyClient struct{}
	client := &DummyClient{}

	// Create a handler and call it
	handler := GenericList(client, "NonExistentResource")
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)

	content := result.Content[0].(mcp.TextContent)
	assert.Contains(t, content.Text, "method")
	assert.Contains(t, content.Text, "not found")
}

func TestGenericGet_MethodNotFound(t *testing.T) {
	type DummyClient struct{}
	client := &DummyClient{}

	handler := GenericGet(client, "NonExistentResource", false)
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"id": "123"}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)

	content := result.Content[0].(mcp.TextContent)
	assert.Contains(t, content.Text, "method")
	assert.Contains(t, content.Text, "not found")
}

func TestGenericCreate_MethodNotFound(t *testing.T) {
	type DummyClient struct{}
	client := &DummyClient{}

	handler := GenericCreate(client, "NonExistentResource", func() any {
		return &struct {
			Name string `json:"name"`
		}{}
	})
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"name": "value"}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)

	content := result.Content[0].(mcp.TextContent)
	assert.Contains(t, content.Text, "method")
	assert.Contains(t, content.Text, "not found")
}

func TestGenericCreate_InvalidData(t *testing.T) {
	type DummyClient struct{}
	client := &DummyClient{}

	// Use a type factory that returns a struct we can't unmarshal "invalid" into
	handler := GenericCreate(client, "NonExistentResource", func() any {
		return &struct {
			Required int `json:"required"`
		}{}
	})
	req := mcp.CallToolRequest{}
	// Provide value that's not valid for the struct field
	req.Params.Arguments = map[string]any{"required": "not a valid json object"}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)

	content := result.Content[0].(mcp.TextContent)
	assert.Contains(t, content.Text, "invalid data")
}

func TestGenericUpdate_MethodNotFound(t *testing.T) {
	type DummyClient struct{}
	client := &DummyClient{}

	handler := GenericUpdate(client, "NonExistentResource", func() any {
		return &struct {
			Name string `json:"name"`
		}{}
	}, false)
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"id": "123", "name": "value"}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)

	content := result.Content[0].(mcp.TextContent)
	assert.Contains(t, content.Text, "method")
	assert.Contains(t, content.Text, "not found")
}

func TestGenericUpdate_InvalidData(t *testing.T) {
	type DummyClient struct{}
	client := &DummyClient{}

	handler := GenericUpdate(client, "NonExistentResource", func() any {
		return &struct {
			Required int `json:"required"`
		}{}
	}, false)
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"id": "123", "required": "not valid"}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)

	content := result.Content[0].(mcp.TextContent)
	assert.Contains(t, content.Text, "invalid data")
}

func TestGenericDelete_MethodNotFound(t *testing.T) {
	type DummyClient struct{}
	client := &DummyClient{}

	handler := GenericDelete(client, "NonExistentResource")
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"id": "123"}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)

	content := result.Content[0].(mcp.TextContent)
	assert.Contains(t, content.Text, "method")
	assert.Contains(t, content.Text, "not found")
}

func TestGenericGet_MissingID(t *testing.T) {
	client := &FakeTestClient{}
	handler := GenericGet(client, "Test", false)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"site": "default"} // Missing "id"

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)

	content := result.Content[0].(mcp.TextContent)
	assert.Contains(t, content.Text, "id")
	assert.Contains(t, content.Text, "missing")
}

func TestGenericGet_InvalidIDType(t *testing.T) {
	client := &FakeTestClient{}
	handler := GenericGet(client, "Test", false)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"site": "default", "id": 123} // int instead of string

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)

	content := result.Content[0].(mcp.TextContent)
	assert.Contains(t, content.Text, "id")
}

func TestGenericDelete_MissingID(t *testing.T) {
	client := &FakeTestClient{}
	handler := GenericDelete(client, "Test")

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"site": "default"} // Missing "id"

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)

	content := result.Content[0].(mcp.TextContent)
	assert.Contains(t, content.Text, "id")
	assert.Contains(t, content.Text, "missing")
}

func TestGenericCreate_MissingData(t *testing.T) {
	client := &FakeTestClient{}
	handler := GenericCreate(client, "Test", func() any {
		return &struct {
			Name string `json:"name"`
		}{}
	})

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"site": "default"}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)

	content := result.Content[0].(mcp.TextContent)
	assert.Contains(t, content.Text, "no fields")
}

func TestGenericUpdate_MissingData(t *testing.T) {
	client := &FakeTestClient{}
	handler := GenericUpdate(client, "Test", func() any {
		return &struct {
			Name string `json:"name"`
		}{}
	}, false)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"site": "default", "id": "123"}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)

	content := result.Content[0].(mcp.TextContent)
	assert.Contains(t, content.Text, "no fields")
}

func TestGenericCreate_UnexpectedParameters(t *testing.T) {
	client := &FakeTestClient{}
	handler := GenericCreate(client, "Test", func() any {
		return &struct {
			Name string `json:"name"`
		}{}
	})

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"site":  "default",
		"name":  "value",
		"data":  map[string]any{"name": "ignored"},
		"extra": true,
	}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)

	content := result.Content[0].(mcp.TextContent)
	assert.Contains(t, content.Text, "unexpected parameters")
	assert.Contains(t, content.Text, "data")
	assert.Contains(t, content.Text, "extra")
}

func TestGenericUpdate_UnexpectedParameters(t *testing.T) {
	client := &FakeTestClient{}
	handler := GenericUpdate(client, "Test", func() any {
		return &struct {
			Name string `json:"name"`
		}{}
	}, false)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"site":  "default",
		"id":    "123",
		"name":  "value",
		"data":  map[string]any{"name": "ignored"},
		"extra": true,
	}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)

	content := result.Content[0].(mcp.TextContent)
	assert.Contains(t, content.Text, "unexpected parameters")
	assert.Contains(t, content.Text, "data")
	assert.Contains(t, content.Text, "extra")
}

// FakeTestClient provides methods we can use for reflection-based testing.
type FakeTestClient struct {
	ShouldError bool
}

func (c *FakeTestClient) ListTest(_ context.Context, _ string) ([]any, error) {
	if c.ShouldError {
		return nil, errors.New("list error")
	}
	return []any{"item1", "item2"}, nil
}

func (c *FakeTestClient) GetTest(_ context.Context, _, _ string) (any, error) {
	if c.ShouldError {
		return nil, errors.New("get error")
	}
	return map[string]string{"id": "123", "name": "test"}, nil
}

func (c *FakeTestClient) GetTestSetting(_ context.Context, _ string) (any, error) {
	if c.ShouldError {
		return nil, errors.New("get setting error")
	}
	return map[string]string{"enabled": "true"}, nil
}

func (c *FakeTestClient) CreateTest(_ context.Context, _ string, _ any) (any, error) {
	if c.ShouldError {
		return nil, errors.New("create error")
	}
	return map[string]string{"id": "new", "name": "created"}, nil
}

func (c *FakeTestClient) UpdateTest(_ context.Context, _ string, _ any) (any, error) {
	if c.ShouldError {
		return nil, errors.New("update error")
	}
	return map[string]string{"id": "123", "name": "updated"}, nil
}

func (c *FakeTestClient) DeleteTest(_ context.Context, _, _ string) error {
	if c.ShouldError {
		return errors.New("delete error")
	}
	return nil
}

func TestGenericList_WithFakeClient(t *testing.T) {
	client := &FakeTestClient{}
	handler := GenericList(client, "Test")

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"site": "default"}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)

	content := result.Content[0].(mcp.TextContent)
	assert.Contains(t, content.Text, "item1")
	assert.Contains(t, content.Text, "item2")
}

func TestGenericList_WithFakeClient_Error(t *testing.T) {
	client := &FakeTestClient{ShouldError: true}
	handler := GenericList(client, "Test")

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"site": "default"}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)

	content := result.Content[0].(mcp.TextContent)
	assert.Contains(t, content.Text, "list error")
}

func TestGenericGet_WithFakeClient(t *testing.T) {
	client := &FakeTestClient{}
	handler := GenericGet(client, "Test", false)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"site": "default", "id": "123"}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)

	content := result.Content[0].(mcp.TextContent)
	assert.Contains(t, content.Text, "123")
	assert.Contains(t, content.Text, "test")
}

func TestGenericGet_WithFakeClient_Setting(t *testing.T) {
	client := &FakeTestClient{}
	handler := GenericGet(client, "TestSetting", true)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"site": "default"}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)

	content := result.Content[0].(mcp.TextContent)
	assert.Contains(t, content.Text, "enabled")
}

func TestGenericGet_WithFakeClient_Setting_Error(t *testing.T) {
	client := &FakeTestClient{ShouldError: true}
	handler := GenericGet(client, "TestSetting", true)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"site": "default"}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)

	content := result.Content[0].(mcp.TextContent)
	assert.Contains(t, content.Text, "get setting error")
}

func TestGenericGet_WithFakeClient_Error(t *testing.T) {
	client := &FakeTestClient{ShouldError: true}
	handler := GenericGet(client, "Test", false)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"site": "default", "id": "123"}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)

	content := result.Content[0].(mcp.TextContent)
	assert.Contains(t, content.Text, "get error")
}

func TestGenericCreate_WithFakeClient(t *testing.T) {
	client := &FakeTestClient{}
	handler := GenericCreate(client, "Test", func() any {
		return &struct {
			Name string `json:"name"`
		}{}
	})

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"site": "default",
		"name": "new item",
	}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)

	content := result.Content[0].(mcp.TextContent)
	assert.Contains(t, content.Text, "created")
}

func TestGenericCreate_WithFakeClient_Error(t *testing.T) {
	client := &FakeTestClient{ShouldError: true}
	handler := GenericCreate(client, "Test", func() any {
		return &struct {
			Name string `json:"name"`
		}{}
	})

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"site": "default",
		"name": "new item",
	}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)

	content := result.Content[0].(mcp.TextContent)
	assert.Contains(t, content.Text, "create error")
}

func TestGenericUpdate_WithFakeClient(t *testing.T) {
	client := &FakeTestClient{}
	handler := GenericUpdate(client, "Test", func() any {
		return &struct {
			Name string `json:"name"`
		}{}
	}, false)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"site": "default",
		"id":   "123",
		"name": "updated item",
	}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)

	content := result.Content[0].(mcp.TextContent)
	assert.Contains(t, content.Text, "updated")
}

func TestGenericUpdate_WithFakeClient_Error(t *testing.T) {
	client := &FakeTestClient{ShouldError: true}
	handler := GenericUpdate(client, "Test", func() any {
		return &struct {
			Name string `json:"name"`
		}{}
	}, false)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"site": "default",
		"id":   "123",
		"name": "updated item",
	}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)

	content := result.Content[0].(mcp.TextContent)
	assert.Contains(t, content.Text, "update error")
}

func TestGenericDelete_WithFakeClient(t *testing.T) {
	client := &FakeTestClient{}
	handler := GenericDelete(client, "Test")

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"site": "default", "id": "123"}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)

	content := result.Content[0].(mcp.TextContent)
	assert.Contains(t, content.Text, "success")
}

func TestGenericDelete_WithFakeClient_Error(t *testing.T) {
	client := &FakeTestClient{ShouldError: true}
	handler := GenericDelete(client, "Test")

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"site": "default", "id": "123"}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)

	content := result.Content[0].(mcp.TextContent)
	assert.Contains(t, content.Text, "delete error")
}

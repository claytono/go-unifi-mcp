package meta

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"

	"github.com/claytono/go-unifi-mcp/internal/tools/generated"
	"github.com/filipowm/go-unifi/unifi"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolIndex_ReturnsAllTools(t *testing.T) {
	handler := ToolIndexHandler()

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)

	// Parse the result
	var tools []generated.ToolMetadata
	content := result.Content[0].(mcp.TextContent)
	err = json.Unmarshal([]byte(content.Text), &tools)
	require.NoError(t, err)

	// Should return all tools
	assert.Equal(t, len(generated.AllToolMetadata), len(tools))
}

func TestToolIndex_FilterByCategory(t *testing.T) {
	handler := ToolIndexHandler()

	tests := []struct {
		category string
	}{
		{"list"},
		{"get"},
		{"create"},
		{"update"},
		{"delete"},
	}

	for _, tc := range tests {
		t.Run(tc.category, func(t *testing.T) {
			req := mcp.CallToolRequest{}
			req.Params.Arguments = map[string]any{
				"category": tc.category,
			}

			result, err := handler(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.False(t, result.IsError)

			var tools []generated.ToolMetadata
			content := result.Content[0].(mcp.TextContent)
			err = json.Unmarshal([]byte(content.Text), &tools)
			require.NoError(t, err)

			// All returned tools should have the specified category
			for _, tool := range tools {
				assert.Equal(t, tc.category, tool.Category)
			}

			// Should have at least some tools
			assert.Greater(t, len(tools), 0)
		})
	}
}

func TestToolIndex_FilterByResource(t *testing.T) {
	handler := ToolIndexHandler()

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"resource": "network",
	}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)

	var tools []generated.ToolMetadata
	content := result.Content[0].(mcp.TextContent)
	err = json.Unmarshal([]byte(content.Text), &tools)
	require.NoError(t, err)

	// All returned tools should have "network" in resource name (case-insensitive)
	for _, tool := range tools {
		assert.Contains(t, tool.Resource, "Network")
	}

	// Should have multiple network-related tools
	assert.Greater(t, len(tools), 0)
}

func TestToolIndex_FilterByCategoryAndResource(t *testing.T) {
	handler := ToolIndexHandler()

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"category": "list",
		"resource": "firewall",
	}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)

	var tools []generated.ToolMetadata
	content := result.Content[0].(mcp.TextContent)
	err = json.Unmarshal([]byte(content.Text), &tools)
	require.NoError(t, err)

	for _, tool := range tools {
		assert.Equal(t, "list", tool.Category)
		assert.Contains(t, tool.Resource, "Firewall")
	}
}

func TestToolIndex_CaseInsensitiveFilters(t *testing.T) {
	handler := ToolIndexHandler()

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"category": "LIST",
		"resource": "NETWORK",
	}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)

	var tools []generated.ToolMetadata
	content := result.Content[0].(mcp.TextContent)
	err = json.Unmarshal([]byte(content.Text), &tools)
	require.NoError(t, err)

	// Should still return results with case-insensitive matching
	assert.Greater(t, len(tools), 0)
}

func TestExecute_UnknownToolReturnsError(t *testing.T) {
	registry := make(map[string]generated.HandlerFunc)
	handler := ExecuteHandler(nil, registry)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"tool":      "unknown_tool",
		"arguments": map[string]any{},
	}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)

	content := result.Content[0].(mcp.TextContent)
	assert.Contains(t, content.Text, "unknown tool")
}

func TestExecute_MissingToolNameReturnsError(t *testing.T) {
	registry := make(map[string]generated.HandlerFunc)
	handler := ExecuteHandler(nil, registry)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)

	content := result.Content[0].(mcp.TextContent)
	assert.Contains(t, content.Text, "tool name is required")
}

func TestExecute_CallsUnderlyingTool(t *testing.T) {
	called := false
	mockHandler := func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) { //nolint:unparam
		called = true
		return mcp.NewToolResultText(`{"success": true}`), nil
	}

	registry := map[string]generated.HandlerFunc{
		"test_tool": func(_ unifi.Client) server.ToolHandlerFunc {
			return mockHandler
		},
	}

	handler := ExecuteHandler(nil, registry)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"tool":      "test_tool",
		"arguments": map[string]any{"site": "default"},
	}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)
	assert.True(t, called)
}

func TestBatch_EmptyCallsReturnsError(t *testing.T) {
	registry := make(map[string]generated.HandlerFunc)
	handler := BatchHandler(nil, registry)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"calls": []any{},
	}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)

	content := result.Content[0].(mcp.TextContent)
	assert.Contains(t, content.Text, "calls array is required")
}

func TestBatch_MissingCallsReturnsError(t *testing.T) {
	registry := make(map[string]generated.HandlerFunc)
	handler := BatchHandler(nil, registry)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)

	content := result.Content[0].(mcp.TextContent)
	assert.Contains(t, content.Text, "calls array is required")
}

func TestBatch_ParallelExecution(t *testing.T) {
	var callCount int32
	mockHandler := func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) { //nolint:unparam
		atomic.AddInt32(&callCount, 1)
		return mcp.NewToolResultText(`{"result": "ok"}`), nil
	}

	registry := map[string]generated.HandlerFunc{
		"test_tool": func(_ unifi.Client) server.ToolHandlerFunc {
			return mockHandler
		},
	}

	handler := BatchHandler(nil, registry)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"calls": []any{
			map[string]any{"tool": "test_tool", "arguments": map[string]any{}},
			map[string]any{"tool": "test_tool", "arguments": map[string]any{}},
			map[string]any{"tool": "test_tool", "arguments": map[string]any{}},
		},
	}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)
	assert.Equal(t, int32(3), atomic.LoadInt32(&callCount))

	// Parse results
	var results []map[string]any
	content := result.Content[0].(mcp.TextContent)
	err = json.Unmarshal([]byte(content.Text), &results)
	require.NoError(t, err)
	assert.Len(t, results, 3)

	// Check each result
	for _, r := range results {
		assert.Contains(t, r, "tool")
		assert.Contains(t, r, "result")
		assert.Equal(t, "test_tool", r["tool"])
	}
}

func TestBatch_PartialFailure(t *testing.T) {
	mockHandler := func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText(`{"result": "ok"}`), nil
	}

	registry := map[string]generated.HandlerFunc{
		"test_tool": func(_ unifi.Client) server.ToolHandlerFunc {
			return mockHandler
		},
	}

	handler := BatchHandler(nil, registry)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"calls": []any{
			map[string]any{"tool": "test_tool", "arguments": map[string]any{}},
			map[string]any{"tool": "unknown_tool", "arguments": map[string]any{}},
			map[string]any{"tool": "test_tool", "arguments": map[string]any{}},
		},
	}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)

	// Parse results
	var results []map[string]any
	content := result.Content[0].(mcp.TextContent)
	err = json.Unmarshal([]byte(content.Text), &results)
	require.NoError(t, err)
	assert.Len(t, results, 3)

	// First call should succeed
	assert.Contains(t, results[0], "result")
	assert.NotContains(t, results[0], "error")

	// Second call should have error
	assert.Contains(t, results[1], "error")
	assert.Contains(t, results[1]["error"], "unknown tool")

	// Third call should succeed
	assert.Contains(t, results[2], "result")
	assert.NotContains(t, results[2], "error")
}

func TestBatch_InvalidCallFormat(t *testing.T) {
	registry := make(map[string]generated.HandlerFunc)
	handler := BatchHandler(nil, registry)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"calls": []any{
			"not a map",
		},
	}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)

	// Parse results
	var results []map[string]any
	content := result.Content[0].(mcp.TextContent)
	err = json.Unmarshal([]byte(content.Text), &results)
	require.NoError(t, err)
	assert.Len(t, results, 1)

	// Should have error about invalid format
	assert.Contains(t, results[0], "error")
	assert.Contains(t, results[0]["error"], "invalid call format")
}

func TestBatch_MissingToolName(t *testing.T) {
	registry := make(map[string]generated.HandlerFunc)
	handler := BatchHandler(nil, registry)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"calls": []any{
			map[string]any{"arguments": map[string]any{}},
		},
	}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)

	// Parse results
	var results []map[string]any
	content := result.Content[0].(mcp.TextContent)
	err = json.Unmarshal([]byte(content.Text), &results)
	require.NoError(t, err)
	assert.Len(t, results, 1)

	// Should have error about missing tool name
	assert.Contains(t, results[0], "error")
	assert.Contains(t, results[0]["error"], "tool name is required")
}

func TestFilterTools_EmptyFilters(t *testing.T) {
	tools := []generated.ToolMetadata{
		{Name: "tool1", Category: "list", Resource: "Network"},
		{Name: "tool2", Category: "get", Resource: "Device"},
	}

	result := filterTools(tools, "", "")
	assert.Equal(t, tools, result)
}

func TestFilterTools_CategoryOnly(t *testing.T) {
	tools := []generated.ToolMetadata{
		{Name: "tool1", Category: "list", Resource: "Network"},
		{Name: "tool2", Category: "get", Resource: "Device"},
		{Name: "tool3", Category: "list", Resource: "Device"},
	}

	result := filterTools(tools, "list", "")
	assert.Len(t, result, 2)
	assert.Equal(t, "tool1", result[0].Name)
	assert.Equal(t, "tool3", result[1].Name)
}

func TestFilterTools_ResourceOnly(t *testing.T) {
	tools := []generated.ToolMetadata{
		{Name: "tool1", Category: "list", Resource: "Network"},
		{Name: "tool2", Category: "get", Resource: "Device"},
		{Name: "tool3", Category: "list", Resource: "NetworkConf"},
	}

	result := filterTools(tools, "", "network")
	assert.Len(t, result, 2)
	assert.Equal(t, "tool1", result[0].Name)
	assert.Equal(t, "tool3", result[1].Name)
}

func TestFilterTools_BothFilters(t *testing.T) {
	tools := []generated.ToolMetadata{
		{Name: "tool1", Category: "list", Resource: "Network"},
		{Name: "tool2", Category: "get", Resource: "Network"},
		{Name: "tool3", Category: "list", Resource: "Device"},
	}

	result := filterTools(tools, "list", "network")
	assert.Len(t, result, 1)
	assert.Equal(t, "tool1", result[0].Name)
}

func TestRegisterMetaTools(t *testing.T) {
	s := server.NewMCPServer("test", "1.0", server.WithToolCapabilities(true))

	// Register meta tools (client can be nil for this test)
	RegisterMetaTools(s, nil)

	// We can't easily inspect registered tools without accessing internal state,
	// but we can verify the function doesn't panic with nil client
	assert.NotNil(t, s)
}

func TestExecute_NilArguments(t *testing.T) {
	mockHandler := func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) { //nolint:unparam
		return mcp.NewToolResultText(`{"success": true}`), nil
	}

	registry := map[string]generated.HandlerFunc{
		"test_tool": func(_ unifi.Client) server.ToolHandlerFunc {
			return mockHandler
		},
	}

	handler := ExecuteHandler(nil, registry)

	// Call without arguments field - should use empty map
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"tool": "test_tool",
		// no "arguments" key
	}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)
}

func TestBatch_NilArguments(t *testing.T) {
	mockHandler := func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) { //nolint:unparam
		return mcp.NewToolResultText(`{"success": true}`), nil
	}

	registry := map[string]generated.HandlerFunc{
		"test_tool": func(_ unifi.Client) server.ToolHandlerFunc {
			return mockHandler
		},
	}

	handler := BatchHandler(nil, registry)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"calls": []any{
			map[string]any{"tool": "test_tool"}, // no "arguments" key
		},
	}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)

	var results []map[string]any
	content := result.Content[0].(mcp.TextContent)
	err = json.Unmarshal([]byte(content.Text), &results)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Contains(t, results[0], "result")
}

func TestBatch_HandlerReturnsError(t *testing.T) {
	mockHandler := func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return nil, assert.AnError // Return an actual error
	}

	registry := map[string]generated.HandlerFunc{
		"error_tool": func(_ unifi.Client) server.ToolHandlerFunc {
			return mockHandler
		},
	}

	handler := BatchHandler(nil, registry)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"calls": []any{
			map[string]any{"tool": "error_tool", "arguments": map[string]any{}},
		},
	}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)

	var results []map[string]any
	content := result.Content[0].(mcp.TextContent)
	err = json.Unmarshal([]byte(content.Text), &results)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Contains(t, results[0], "error")
}

func TestBatch_NonJSONResult(t *testing.T) {
	mockHandler := func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) { //nolint:unparam
		return mcp.NewToolResultText("plain text, not JSON"), nil
	}

	registry := map[string]generated.HandlerFunc{
		"text_tool": func(_ unifi.Client) server.ToolHandlerFunc {
			return mockHandler
		},
	}

	handler := BatchHandler(nil, registry)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"calls": []any{
			map[string]any{"tool": "text_tool", "arguments": map[string]any{}},
		},
	}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)

	var results []map[string]any
	content := result.Content[0].(mcp.TextContent)
	err = json.Unmarshal([]byte(content.Text), &results)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	// Result should be stored as plain text string
	assert.Equal(t, "plain text, not JSON", results[0]["result"])
}

package server

import (
	"context"
	"encoding/json"
	"testing"

	servermocks "github.com/claytono/go-unifi-mcp/internal/server/mocks"
	"github.com/claytono/go-unifi-mcp/internal/tools/generated"
	"github.com/filipowm/go-unifi/unifi"
	clientpkg "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestServerToolCount verifies all tools are registered.
func TestServerToolCount(t *testing.T) {
	client := servermocks.NewClient(t)

	s, err := New(Options{Client: client, Mode: ModeEager})
	assert.NoError(t, err)
	assert.NotNil(t, s)
	assert.Len(t, s.ListTools(), 252)
}

func TestLazyModeEndToEnd(t *testing.T) {
	ctx := context.Background()
	client := servermocks.NewClient(t)
	client.On("ListNetwork", mock.Anything, "default").Return([]unifi.Network{}, nil).Twice()
	client.On("ListDevice", mock.Anything, "default").Return([]unifi.Device{}, nil).Once()

	// Build a lazy-mode server and in-process MCP client for end-to-end calls.
	s, err := New(Options{Client: client, Mode: ModeLazy})
	require.NoError(t, err)
	require.NotNil(t, s)

	mcpClient, err := clientpkg.NewInProcessClient(s)
	require.NoError(t, err)
	defer func() {
		err = mcpClient.Close()
		require.NoError(t, err)
	}()

	// Start and initialize the MCP session.
	require.NoError(t, mcpClient.Start(ctx))
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{Name: "integration-test", Version: "1.0.0"}
	_, err = mcpClient.Initialize(ctx, initRequest)
	require.NoError(t, err)

	// Verify lazy mode exposes only meta tools.
	toolList, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	require.NoError(t, err)
	require.NotNil(t, toolList)
	assert.Len(t, toolList.Tools, 3)

	// Fetch the generated tool catalog via the meta tool.
	indexRequest := mcp.CallToolRequest{}
	indexRequest.Params.Name = "tool_index"
	indexRequest.Params.Arguments = map[string]any{}
	indexResult, err := mcpClient.CallTool(ctx, indexRequest)
	require.NoError(t, err)
	require.NotNil(t, indexResult)
	assert.False(t, indexResult.IsError)

	var toolCatalog []generated.ToolMetadata
	require.NotEmpty(t, indexResult.Content)
	indexContent, ok := indexResult.Content[0].(mcp.TextContent)
	require.True(t, ok)
	err = json.Unmarshal([]byte(indexContent.Text), &toolCatalog)
	require.NoError(t, err)
	assert.Len(t, toolCatalog, len(generated.AllToolMetadata))

	// Execute a single tool via the meta execute wrapper.
	executeRequest := mcp.CallToolRequest{}
	executeRequest.Params.Name = "execute"
	executeRequest.Params.Arguments = map[string]any{
		"tool":      "list_network",
		"arguments": map[string]any{},
	}
	executeResult, err := mcpClient.CallTool(ctx, executeRequest)
	require.NoError(t, err)
	require.NotNil(t, executeResult)
	assert.False(t, executeResult.IsError)

	// Execute multiple tools via the meta batch wrapper.
	batchRequest := mcp.CallToolRequest{}
	batchRequest.Params.Name = "batch"
	batchRequest.Params.Arguments = map[string]any{
		"calls": []any{
			map[string]any{"tool": "list_network", "arguments": map[string]any{}},
			map[string]any{"tool": "list_device", "arguments": map[string]any{}},
		},
	}
	batchResult, err := mcpClient.CallTool(ctx, batchRequest)
	require.NoError(t, err)
	require.NotNil(t, batchResult)
	assert.False(t, batchResult.IsError)

	var batchResults []map[string]any
	require.NotEmpty(t, batchResult.Content)
	batchContent, ok := batchResult.Content[0].(mcp.TextContent)
	require.True(t, ok)
	err = json.Unmarshal([]byte(batchContent.Text), &batchResults)
	require.NoError(t, err)
	require.Len(t, batchResults, 2)
	assert.Equal(t, "list_network", batchResults[0]["tool"])
	assert.Equal(t, "list_device", batchResults[1]["tool"])
	assert.NotContains(t, batchResults[0], "error")
	assert.NotContains(t, batchResults[1], "error")

	client.AssertExpectations(t)
}

// TestToolHandlerPanicIsRecovered guards against regressions in the
// underlying go-unifi client (e.g. the go-playground/validator "Bad field
// type []string" panic triggered by a WLAN update with wlan_bands set,
// github.com/filipowm/go-unifi issue reproduced 2026-07-02 and fixed
// upstream in v1.9.0) taking down the whole MCP connection instead of
// surfacing as a normal tool error. Any panic raised inside a tool handler
// -- from this bug, a different validator bug, or anything else in the
// client -- must be converted into a clean JSON-RPC error, and the
// connection must stay usable for subsequent calls afterward. Before the
// fix this panics straight out of CallTool and takes the whole test binary
// down with it, mirroring how the panic took down the real stdio MCP
// connection.
func TestToolHandlerPanicIsRecovered(t *testing.T) {
	ctx := context.Background()
	client := servermocks.NewClient(t)
	client.On("ListNetwork", mock.Anything, "default").
		Panic("reflect: Bad field type []string for field WLANBands").Once()
	client.On("ListDevice", mock.Anything, "default").Return([]unifi.Device{}, nil).Once()

	s, err := New(Options{Client: client, Mode: ModeEager})
	require.NoError(t, err)
	require.NotNil(t, s)

	mcpClient, err := clientpkg.NewInProcessClient(s)
	require.NoError(t, err)
	defer func() {
		err = mcpClient.Close()
		require.NoError(t, err)
	}()

	require.NoError(t, mcpClient.Start(ctx))
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{Name: "integration-test", Version: "1.0.0"}
	_, err = mcpClient.Initialize(ctx, initRequest)
	require.NoError(t, err)

	listNetworkRequest := mcp.CallToolRequest{}
	listNetworkRequest.Params.Name = "list_network"
	listNetworkRequest.Params.Arguments = map[string]any{}

	// The panicking call must come back as a normal (non-panicking) error,
	// not crash the test binary / connection.
	_, err = mcpClient.CallTool(ctx, listNetworkRequest)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "panic recovered")
	assert.Contains(t, err.Error(), "list_network")

	// The connection must still be usable for a subsequent call -- this is
	// the actual regression the fix guards against.
	listDeviceRequest := mcp.CallToolRequest{}
	listDeviceRequest.Params.Name = "list_device"
	listDeviceRequest.Params.Arguments = map[string]any{}
	result, err := mcpClient.CallTool(ctx, listDeviceRequest)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)

	client.AssertExpectations(t)
}

func TestEagerModeEndToEnd(t *testing.T) {
	ctx := context.Background()
	client := servermocks.NewClient(t)
	client.On("ListNetwork", mock.Anything, "default").Return([]unifi.Network{}, nil).Once()
	client.On("ListDevice", mock.Anything, "default").Return([]unifi.Device{}, nil).Once()

	// Build an eager-mode server with full tool registration.
	s, err := New(Options{Client: client, Mode: ModeEager})
	require.NoError(t, err)
	require.NotNil(t, s)

	mcpClient, err := clientpkg.NewInProcessClient(s)
	require.NoError(t, err)
	defer func() {
		err = mcpClient.Close()
		require.NoError(t, err)
	}()

	// Start and initialize the MCP session.
	require.NoError(t, mcpClient.Start(ctx))
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{Name: "integration-test", Version: "1.0.0"}
	_, err = mcpClient.Initialize(ctx, initRequest)
	require.NoError(t, err)

	// Verify eager mode exposes the full tool catalog.
	toolList, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	require.NoError(t, err)
	require.NotNil(t, toolList)
	assert.Len(t, toolList.Tools, len(generated.AllToolMetadata))

	// Call direct tools to ensure routing works without meta wrappers.
	listNetworkRequest := mcp.CallToolRequest{}
	listNetworkRequest.Params.Name = "list_network"
	listNetworkRequest.Params.Arguments = map[string]any{}
	listNetworkResult, err := mcpClient.CallTool(ctx, listNetworkRequest)
	require.NoError(t, err)
	require.NotNil(t, listNetworkResult)
	assert.False(t, listNetworkResult.IsError)

	listDeviceRequest := mcp.CallToolRequest{}
	listDeviceRequest.Params.Name = "list_device"
	listDeviceRequest.Params.Arguments = map[string]any{}
	listDeviceResult, err := mcpClient.CallTool(ctx, listDeviceRequest)
	require.NoError(t, err)
	require.NotNil(t, listDeviceResult)
	assert.False(t, listDeviceResult.IsError)

	client.AssertExpectations(t)
}

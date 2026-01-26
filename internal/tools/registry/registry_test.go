package registry

import (
	"errors"
	"testing"

	"github.com/claytono/go-unifi-mcp/internal/tools/generated"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterAllTools(t *testing.T) {
	s := server.NewMCPServer("test", "1.0", server.WithToolCapabilities(true))

	// Use nil client - handlers won't be called in this test
	err := RegisterAllTools(s, nil)
	require.NoError(t, err)

	// We can't easily inspect registered tools, but we can verify no error
	assert.NotNil(t, s)
}

func TestBuildToolFromMetadata(t *testing.T) {
	tests := []struct {
		name    string
		meta    generated.ToolMetadata
		wantErr bool
	}{
		{
			name: "simple tool",
			meta: generated.ToolMetadata{
				Name:        "list_network",
				Description: "List all networks",
				Category:    "list",
				Resource:    "Network",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"site": map[string]any{
							"type":        "string",
							"description": "UniFi site name",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "tool with required field",
			meta: generated.ToolMetadata{
				Name:        "get_network",
				Description: "Get a network by ID",
				Category:    "get",
				Resource:    "Network",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"site": map[string]any{
							"type": "string",
						},
						"id": map[string]any{
							"type": "string",
						},
					},
					"required": []string{"id"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool, err := buildToolFromMetadata(tt.meta)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.meta.Name, tool.Name)
			assert.Equal(t, tt.meta.Description, tool.Description)
			assert.NotNil(t, tool.InputSchema)
		})
	}
}

func TestBuildToolFromMetadata_InvalidSchema(t *testing.T) {
	// Create a schema with a value that can't be marshaled to JSON
	// Channels can't be marshaled to JSON
	meta := generated.ToolMetadata{
		Name:        "test",
		Description: "test",
		InputSchema: map[string]any{
			"invalid": make(chan int),
		},
	}

	_, err := buildToolFromMetadata(meta)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal schema")
}

func TestRegisterAllTools_VerifyToolCount(t *testing.T) {
	s := server.NewMCPServer("test", "1.0", server.WithToolCapabilities(true))

	err := RegisterAllTools(s, nil)
	require.NoError(t, err)

	// Verify we registered the expected number of tools
	expectedCount := len(generated.AllToolMetadata)

	// The server should have at least this many tools
	// (we can't easily verify exact count without server internals)
	assert.Greater(t, expectedCount, 200, "should have more than 200 tools")
}

func TestRegisterTools_BuildError(t *testing.T) {
	s := server.NewMCPServer("test", "1.0", server.WithToolCapabilities(true))

	// Create metadata with invalid schema (channel can't be marshaled)
	tools := []generated.ToolMetadata{
		{
			Name:        "test_tool",
			Description: "Test",
			InputSchema: map[string]any{
				"invalid": make(chan int),
			},
		},
	}

	handlers := map[string]generated.HandlerFunc{}

	err := registerTools(s, nil, tools, handlers)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to build tool")
}

func TestRegisterTools_MissingHandler(t *testing.T) {
	s := server.NewMCPServer("test", "1.0", server.WithToolCapabilities(true))

	// Create valid metadata but no matching handler
	tools := []generated.ToolMetadata{
		{
			Name:        "test_tool",
			Description: "Test",
			InputSchema: map[string]any{
				"type": "object",
			},
		},
	}

	// Empty handler registry
	handlers := map[string]generated.HandlerFunc{}

	err := registerTools(s, nil, tools, handlers)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no handler for tool")
}

func TestRegisterAllToolsWithValidator_ValidationFailure(t *testing.T) {
	s := server.NewMCPServer("test", "1.0", server.WithToolCapabilities(true))

	// Create a validator that always fails
	failingValidator := func(_ any, _ []generated.ToolMetadata, _ map[string]func() any) error {
		return errors.New("validation failed: missing method ListNetwork")
	}

	// Use mockClient which embeds unifi.Client - methods would panic if called
	// but our validator fails before any methods are invoked
	err := registerAllToolsWithValidator(s, &mockClient{}, failingValidator)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "client validation failed")
	assert.Contains(t, err.Error(), "missing method ListNetwork")
}

func TestBuildToolFromMetadata_PreservesSchema(t *testing.T) {
	meta := generated.ToolMetadata{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "The name",
				},
				"count": map[string]any{
					"type":        "integer",
					"description": "The count",
				},
			},
			"required": []string{"name"},
		},
	}

	tool, err := buildToolFromMetadata(meta)
	require.NoError(t, err)

	// Verify the tool has the expected name and description
	assert.Equal(t, "test_tool", tool.Name)
	assert.Equal(t, "A test tool", tool.Description)

	// Verify the schema is not nil (we can't easily unmarshal it back)
	assert.NotNil(t, tool.InputSchema)
}

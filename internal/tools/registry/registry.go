// Package registry provides data-driven tool registration for the MCP server.
package registry

import (
	"encoding/json"
	"fmt"

	"github.com/claytono/go-unifi-mcp/internal/tools/generated"
	"github.com/filipowm/go-unifi/unifi"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterAllTools registers all generated UniFi MCP tools with the server.
// It builds tools dynamically from the metadata and maps each to its
// corresponding handler from the handler registry.
func RegisterAllTools(s *server.MCPServer, client unifi.Client) error {
	return registerTools(s, client, generated.AllToolMetadata, generated.GetHandlerRegistry())
}

// registerTools is the internal implementation that allows testing with custom metadata.
func registerTools(s *server.MCPServer, client unifi.Client, tools []generated.ToolMetadata, handlers map[string]generated.HandlerFunc) error {
	for _, meta := range tools {
		tool, err := buildToolFromMetadata(meta)
		if err != nil {
			return fmt.Errorf("failed to build tool %s: %w", meta.Name, err)
		}

		handlerFactory, ok := handlers[meta.Name]
		if !ok {
			return fmt.Errorf("no handler for tool %s", meta.Name)
		}

		s.AddTool(tool, handlerFactory(client))
	}

	return nil
}

// buildToolFromMetadata creates an MCP tool from tool metadata.
func buildToolFromMetadata(meta generated.ToolMetadata) (mcp.Tool, error) {
	schemaBytes, err := json.Marshal(meta.InputSchema)
	if err != nil {
		return mcp.Tool{}, fmt.Errorf("failed to marshal schema: %w", err)
	}

	return mcp.NewToolWithRawSchema(
		meta.Name,
		meta.Description,
		json.RawMessage(schemaBytes),
	), nil
}

// Package registry provides data-driven tool registration for the MCP server.
package registry

import (
	"encoding/json"
	"fmt"

	"github.com/claytono/go-unifi-mcp/internal/resolve"
	"github.com/claytono/go-unifi-mcp/internal/tools/generated"
	"github.com/filipowm/go-unifi/unifi"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ValidatorFunc validates a client against tool metadata.
type ValidatorFunc func(client any, tools []generated.ToolMetadata, typeRegistry map[string]func() any) error

// defaultValidator is the production validator.
var defaultValidator ValidatorFunc = generated.ValidateClientMethods

// RegisterAllTools registers all generated UniFi MCP tools with the server.
// It builds tools dynamically from the metadata and maps each to its
// corresponding handler from the handler registry.
func RegisterAllTools(s *server.MCPServer, client unifi.Client, resolver *resolve.Resolver) error {
	return registerAllToolsWithValidator(s, client, resolver, defaultValidator)
}

// registerAllToolsWithValidator is the internal implementation that allows testing with custom validators.
func registerAllToolsWithValidator(s *server.MCPServer, client unifi.Client, resolver *resolve.Resolver, validator ValidatorFunc) error {
	// Validate all client methods exist with correct signatures before registration.
	// Skip validation for nil client (used only in tests).
	if client != nil {
		if err := validator(client, generated.AllToolMetadata, generated.TypeRegistry); err != nil {
			return fmt.Errorf("client validation failed: %w", err)
		}
	}
	return registerTools(s, client, resolver, generated.AllToolMetadata, generated.GetHandlerRegistry())
}

// registerTools is the internal implementation that allows testing with custom metadata.
func registerTools(s *server.MCPServer, client unifi.Client, resolver *resolve.Resolver, tools []generated.ToolMetadata, handlers map[string]generated.HandlerFunc) error {
	for _, meta := range tools {
		tool, err := buildToolFromMetadata(meta)
		if err != nil {
			return fmt.Errorf("failed to build tool %s: %w", meta.Name, err)
		}

		handlerFactory, ok := handlers[meta.Name]
		if !ok {
			return fmt.Errorf("no handler for tool %s", meta.Name)
		}

		handler := handlerFactory(client)
		if meta.Category != "delete" {
			handler = resolve.WrapHandler(handler, resolver)
		}
		s.AddTool(tool, handler)
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

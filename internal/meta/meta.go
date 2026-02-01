// Package meta provides meta-tools for lazy mode operation.
// In lazy mode, only 3 meta-tools are registered instead of 242 direct tools,
// reducing context size from ~5000 tokens to ~200 tokens.
package meta

import (
	"github.com/claytono/go-unifi-mcp/internal/resolve"
	"github.com/claytono/go-unifi-mcp/internal/tools/generated"
	"github.com/filipowm/go-unifi/unifi"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterMetaTools registers the 3 meta-tools for lazy mode operation.
func RegisterMetaTools(s *server.MCPServer, client unifi.Client, resolver *resolve.Resolver) {
	registry := generated.GetHandlerRegistry()

	// tool_index - Returns filtered tool catalog
	s.AddTool(mcp.NewTool("tool_index",
		mcp.WithDescription("Returns the catalog of all available UniFi tools. Use this to discover tools before calling execute."),
		mcp.WithString("category", mcp.Description("Filter by operation type: list, get, create, update, delete")),
		mcp.WithString("resource", mcp.Description("Filter by resource name (case-insensitive partial match)")),
	), ToolIndexHandler())

	// execute - Dispatches to any tool by name
	s.AddTool(mcp.NewTool("execute",
		mcp.WithDescription("Executes any UniFi tool by name. Use tool_index first to discover available tools."),
		mcp.WithString("tool", mcp.Required(), mcp.Description("Name of the tool to execute (e.g., 'list_network')")),
		mcp.WithObject("arguments", mcp.Description("Arguments to pass to the tool")),
	), ExecuteHandler(client, registry, resolver))

	// batch - Executes multiple tools in parallel
	s.AddTool(mcp.NewTool("batch",
		mcp.WithDescription("Executes multiple UniFi tools in parallel. Each call specifies a tool name and its arguments."),
		mcp.WithArray("calls", mcp.Required(), mcp.Description("Array of tool calls, each with 'tool' (string) and 'arguments' (object)")),
	), BatchHandler(client, registry, resolver))
}

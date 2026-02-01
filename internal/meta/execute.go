package meta

import (
	"context"
	"strings"

	"github.com/claytono/go-unifi-mcp/internal/resolve"
	"github.com/claytono/go-unifi-mcp/internal/tools/generated"
	"github.com/filipowm/go-unifi/unifi"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ExecuteHandler returns a handler that dispatches to any tool by name.
func ExecuteHandler(client unifi.Client, registry map[string]generated.HandlerFunc, resolver *resolve.Resolver) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		toolName, ok := args["tool"].(string)
		if !ok || toolName == "" {
			return mcp.NewToolResultError("tool name is required"), nil
		}

		toolArgs, _ := args["arguments"].(map[string]any)
		if toolArgs == nil {
			toolArgs = make(map[string]any)
		}

		handlerFactory, ok := registry[toolName]
		if !ok {
			return mcp.NewToolResultError("unknown tool: " + toolName), nil
		}

		// Build inner request with the tool arguments
		innerReq := mcp.CallToolRequest{}
		innerReq.Params.Name = toolName
		innerReq.Params.Arguments = toolArgs

		handler := handlerFactory(client)
		if !strings.HasPrefix(toolName, "delete_") {
			handler = resolve.WrapHandler(handler, resolver)
		}
		return handler(ctx, innerReq)
	}
}

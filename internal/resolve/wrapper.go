package resolve

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// WrapHandler decorates a tool handler to add ID resolution support.
// Resolution is enabled by default. Set "resolve": false to disable it.
func WrapHandler(handler server.ToolHandlerFunc, resolver *Resolver) server.ToolHandlerFunc {
	if resolver == nil {
		return handler
	}

	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Call the inner handler first
		result, err := handler(ctx, req)
		if err != nil {
			return result, err
		}

		// Resolution is on by default; only skip if explicitly set to false
		args := req.GetArguments()
		if resolveArg, ok := args["resolve"].(bool); ok && !resolveArg {
			return result, nil
		}

		// Don't resolve error results
		if result == nil || result.IsError || len(result.Content) == 0 {
			return result, nil
		}

		// Extract text content
		textContent, ok := result.Content[0].(mcp.TextContent)
		if !ok {
			return result, nil
		}

		// Extract site from args
		site, _ := args["site"].(string)
		if site == "" {
			site = "default"
		}

		// Resolve ID references
		resolved, resolveErr := resolver.ResolveJSON(ctx, site, textContent.Text)
		if resolveErr != nil {
			resolver.logger.Debug("resolve: error resolving JSON, returning original",
				"error", resolveErr)
			return result, nil
		}

		return mcp.NewToolResultText(resolved), nil
	}
}

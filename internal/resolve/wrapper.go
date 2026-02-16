package resolve

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// WrapHandler decorates a tool handler to add ID resolution and extra fields processing.
// Resolution is enabled by default. Set "resolve": false to disable it.
// Extra fields are stripped by default. Set "include_extra_fields": true to include them.
func WrapHandler(handler server.ToolHandlerFunc, resolver *Resolver) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, err := handler(ctx, req)
		if err != nil {
			return result, err
		}

		// Don't post-process error results
		if result == nil || result.IsError || len(result.Content) == 0 {
			return result, nil
		}

		// Extract text content
		textContent, ok := result.Content[0].(mcp.TextContent)
		if !ok {
			return result, nil
		}

		args := req.GetArguments()
		text := textContent.Text

		// ID resolution (on by default, skip if explicitly false)
		if resolver != nil {
			if resolveArg, ok := args["resolve"].(bool); !ok || resolveArg {
				site, _ := args["site"].(string)
				if site == "" {
					site = "default"
				}
				resolved, resolveErr := resolver.ResolveJSON(ctx, site, text)
				if resolveErr != nil {
					resolver.logger.Debug("resolve: error resolving JSON, returning original",
						"error", resolveErr)
				} else {
					text = resolved
				}
			}
		}

		// Extra fields processing (strip by default, include if explicitly true)
		includeExtra, _ := args["include_extra_fields"].(bool)
		processed, processErr := ProcessExtraFields(text, includeExtra)
		if processErr == nil {
			text = processed
		}

		return mcp.NewToolResultText(text), nil
	}
}

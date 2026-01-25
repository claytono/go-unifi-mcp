package meta

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/claytono/go-unifi-mcp/internal/tools/generated"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolIndexHandler returns a handler that returns the filtered tool catalog.
func ToolIndexHandler() server.ToolHandlerFunc {
	return func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		category, _ := args["category"].(string)
		resource, _ := args["resource"].(string)

		results := filterTools(generated.AllToolMetadata, category, resource)
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return mcp.NewToolResultError("failed to marshal results: " + err.Error()), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// filterTools filters the tool metadata based on category and resource.
func filterTools(tools []generated.ToolMetadata, category, resource string) []generated.ToolMetadata {
	if category == "" && resource == "" {
		return tools
	}

	category = strings.ToLower(category)
	resource = strings.ToLower(resource)

	var results []generated.ToolMetadata
	for _, tool := range tools {
		if category != "" && strings.ToLower(tool.Category) != category {
			continue
		}
		if resource != "" && !strings.Contains(strings.ToLower(tool.Resource), resource) {
			continue
		}
		results = append(results, tool)
	}
	return results
}

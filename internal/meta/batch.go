package meta

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/claytono/go-unifi-mcp/internal/tools/generated"
	"github.com/filipowm/go-unifi/unifi"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// BatchHandler returns a handler that executes multiple tools in parallel.
func BatchHandler(client unifi.Client, registry map[string]generated.HandlerFunc) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		calls, ok := args["calls"].([]any)
		if !ok || len(calls) == 0 {
			return mcp.NewToolResultError("calls array is required and must not be empty"), nil
		}

		results := make([]map[string]any, len(calls))
		var wg sync.WaitGroup
		var mu sync.Mutex

		for i, call := range calls {
			wg.Add(1)
			go func(idx int, c any) {
				defer wg.Done()

				result := map[string]any{
					"index": idx,
				}

				callMap, ok := c.(map[string]any)
				if !ok {
					result["error"] = "invalid call format: expected object with 'tool' and 'arguments'"
					mu.Lock()
					results[idx] = result
					mu.Unlock()
					return
				}

				toolName, ok := callMap["tool"].(string)
				if !ok || toolName == "" {
					result["error"] = "tool name is required"
					mu.Lock()
					results[idx] = result
					mu.Unlock()
					return
				}
				result["tool"] = toolName

				toolArgs, _ := callMap["arguments"].(map[string]any)
				if toolArgs == nil {
					toolArgs = make(map[string]any)
				}

				handlerFactory, ok := registry[toolName]
				if !ok {
					result["error"] = "unknown tool: " + toolName
					mu.Lock()
					results[idx] = result
					mu.Unlock()
					return
				}

				// Build inner request
				innerReq := mcp.CallToolRequest{}
				innerReq.Params.Name = toolName
				innerReq.Params.Arguments = toolArgs

				handler := handlerFactory(client)
				toolResult, err := handler(ctx, innerReq)
				if err != nil {
					result["error"] = err.Error()
					mu.Lock()
					results[idx] = result
					mu.Unlock()
					return
				}

				// Extract the result content
				if toolResult != nil && len(toolResult.Content) > 0 {
					if textContent, ok := toolResult.Content[0].(mcp.TextContent); ok {
						// Try to parse as JSON for cleaner output
						var parsed any
						if err := json.Unmarshal([]byte(textContent.Text), &parsed); err == nil {
							result["result"] = parsed
						} else {
							result["result"] = textContent.Text
						}
					}
					result["isError"] = toolResult.IsError
				}

				mu.Lock()
				results[idx] = result
				mu.Unlock()
			}(i, call)
		}

		wg.Wait()

		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return mcp.NewToolResultError("failed to marshal results: " + err.Error()), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

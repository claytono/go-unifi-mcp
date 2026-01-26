package generated

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// GenericList creates a handler that calls client.List<Resource>(ctx, site) via reflection.
// The client parameter accepts any type (typically unifi.Client) and uses reflection
// to call the appropriate method.
func GenericList(client any, resourceName string) server.ToolHandlerFunc {
	methodName := "List" + resourceName

	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		site := extractSite(req)

		clientVal := reflect.ValueOf(client)
		method := clientVal.MethodByName(methodName)
		if !method.IsValid() {
			return mcp.NewToolResultError(fmt.Sprintf("method %s not found", methodName)), nil
		}

		results := method.Call([]reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(site),
		})

		if err := extractError(results[1]); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		data, err := json.MarshalIndent(results[0].Interface(), "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// GenericGet creates a handler that calls client.Get<Resource>(ctx, site, id) via reflection.
// For settings resources (isSetting=true), it calls client.Get<Resource>(ctx, site) without ID.
func GenericGet(client any, resourceName string, isSetting bool) server.ToolHandlerFunc {
	methodName := "Get" + resourceName

	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		site := extractSite(req)

		clientVal := reflect.ValueOf(client)
		method := clientVal.MethodByName(methodName)
		if !method.IsValid() {
			return mcp.NewToolResultError(fmt.Sprintf("method %s not found", methodName)), nil
		}

		var results []reflect.Value
		if isSetting {
			// Settings don't have IDs
			results = method.Call([]reflect.Value{
				reflect.ValueOf(ctx),
				reflect.ValueOf(site),
			})
		} else {
			id, ok := req.GetArguments()["id"].(string)
			if !ok || id == "" {
				return mcp.NewToolResultError("required parameter 'id' is missing or invalid"), nil
			}
			results = method.Call([]reflect.Value{
				reflect.ValueOf(ctx),
				reflect.ValueOf(site),
				reflect.ValueOf(id),
			})
		}

		if err := extractError(results[1]); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		data, err := json.MarshalIndent(results[0].Interface(), "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// GenericCreate creates a handler that calls client.Create<Resource>(ctx, site, &input) via reflection.
func GenericCreate(client any, resourceName string, newTypeFunc func() any) server.ToolHandlerFunc {
	methodName := "Create" + resourceName

	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		site := extractSite(req)

		// Create new instance of the resource type
		input := newTypeFunc()

		// Unmarshal the data argument into the input struct
		dataArg, ok := req.GetArguments()["data"]
		if !ok {
			return mcp.NewToolResultError("required parameter 'data' is missing"), nil
		}
		dataRaw, err := json.Marshal(dataArg)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse data: %v", err)), nil
		}
		if err := json.Unmarshal(dataRaw, input); err != nil {
			return mcp.NewToolResultError("invalid data: " + err.Error()), nil
		}

		clientVal := reflect.ValueOf(client)
		method := clientVal.MethodByName(methodName)
		if !method.IsValid() {
			return mcp.NewToolResultError(fmt.Sprintf("method %s not found", methodName)), nil
		}

		results := method.Call([]reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(site),
			reflect.ValueOf(input),
		})

		if err := extractError(results[1]); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		data, err := json.MarshalIndent(results[0].Interface(), "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// GenericUpdate creates a handler that calls client.Update<Resource>(ctx, site, &input) via reflection.
func GenericUpdate(client any, resourceName string, newTypeFunc func() any) server.ToolHandlerFunc {
	methodName := "Update" + resourceName

	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		site := extractSite(req)

		// Create new instance of the resource type
		input := newTypeFunc()

		// Unmarshal the data argument into the input struct
		dataArg, ok := req.GetArguments()["data"]
		if !ok {
			return mcp.NewToolResultError("required parameter 'data' is missing"), nil
		}
		dataRaw, err := json.Marshal(dataArg)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse data: %v", err)), nil
		}
		if err := json.Unmarshal(dataRaw, input); err != nil {
			return mcp.NewToolResultError("invalid data: " + err.Error()), nil
		}

		clientVal := reflect.ValueOf(client)
		method := clientVal.MethodByName(methodName)
		if !method.IsValid() {
			return mcp.NewToolResultError(fmt.Sprintf("method %s not found", methodName)), nil
		}

		results := method.Call([]reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(site),
			reflect.ValueOf(input),
		})

		if err := extractError(results[1]); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		data, err := json.MarshalIndent(results[0].Interface(), "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// GenericDelete creates a handler that calls client.Delete<Resource>(ctx, site, id) via reflection.
func GenericDelete(client any, resourceName string) server.ToolHandlerFunc {
	methodName := "Delete" + resourceName

	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		site := extractSite(req)
		id, ok := req.GetArguments()["id"].(string)
		if !ok || id == "" {
			return mcp.NewToolResultError("required parameter 'id' is missing or invalid"), nil
		}

		clientVal := reflect.ValueOf(client)
		method := clientVal.MethodByName(methodName)
		if !method.IsValid() {
			return mcp.NewToolResultError(fmt.Sprintf("method %s not found", methodName)), nil
		}

		results := method.Call([]reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(site),
			reflect.ValueOf(id),
		})

		if err := extractError(results[0]); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(`{"success": true}`), nil
	}
}

// extractSite extracts the site parameter from the request, defaulting to "default".
func extractSite(req mcp.CallToolRequest) string {
	site, _ := req.GetArguments()["site"].(string)
	if site == "" {
		site = "default"
	}
	return site
}

// extractError converts a reflect.Value to an error if it's not nil.
func extractError(val reflect.Value) error {
	if val.IsNil() {
		return nil
	}
	return val.Interface().(error)
}

// ValidateClientMethods checks that all expected client methods exist at startup.
// This catches any go-unifi method naming changes immediately rather than on first tool call.
func ValidateClientMethods(client any, tools []ToolMetadata, typeRegistry map[string]func() any) error {
	if client == nil {
		return fmt.Errorf("client cannot be nil")
	}
	clientVal := reflect.ValueOf(client)

	for _, meta := range tools {
		var methodName string

		switch meta.Category {
		case "list":
			methodName = "List" + meta.Resource
		case "get":
			methodName = "Get" + meta.Resource
		case "create":
			methodName = "Create" + meta.Resource
			// Also verify type exists in registry
			if _, ok := typeRegistry[meta.Resource]; !ok {
				return fmt.Errorf("missing type in registry for resource %s (tool %s)", meta.Resource, meta.Name)
			}
		case "update":
			methodName = "Update" + meta.Resource
			// Also verify type exists in registry
			if _, ok := typeRegistry[meta.Resource]; !ok {
				return fmt.Errorf("missing type in registry for resource %s (tool %s)", meta.Resource, meta.Name)
			}
		case "delete":
			methodName = "Delete" + meta.Resource
		default:
			return fmt.Errorf("unknown category %s for tool %s", meta.Category, meta.Name)
		}

		method := clientVal.MethodByName(methodName)
		if !method.IsValid() {
			return fmt.Errorf("missing client method: %s (for tool %s)", methodName, meta.Name)
		}

		// Validate method signature
		methodType := method.Type()
		var expectedIn, expectedOut int
		switch meta.Category {
		case "list":
			expectedIn, expectedOut = 2, 2 // (ctx, site) -> (result, error)
		case "get":
			if meta.IsSetting {
				expectedIn, expectedOut = 2, 2 // (ctx, site) -> (result, error)
			} else {
				expectedIn, expectedOut = 3, 2 // (ctx, site, id) -> (result, error)
			}
		case "create", "update":
			expectedIn, expectedOut = 3, 2 // (ctx, site, input) -> (result, error)
		case "delete":
			expectedIn, expectedOut = 3, 1 // (ctx, site, id) -> error
		}
		if methodType.NumIn() != expectedIn {
			return fmt.Errorf("method %s has %d parameters, expected %d (for tool %s)",
				methodName, methodType.NumIn(), expectedIn, meta.Name)
		}
		if methodType.NumOut() != expectedOut {
			return fmt.Errorf("method %s has %d return values, expected %d (for tool %s)",
				methodName, methodType.NumOut(), expectedOut, meta.Name)
		}
	}

	return nil
}

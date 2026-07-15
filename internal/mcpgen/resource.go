package mcpgen

import (
	"github.com/claytono/go-unifi-mcp/internal/gounifi"
)

// InferOperations determines which CRUD operations are available for a resource,
// minus any operations excluded by go-unifi's resource customizations.
func InferOperations(r *gounifi.Resource, excludedFunctions ...string) []string {
	var operations []string

	// Settings resources only have Get and Update
	switch {
	case r.IsSetting():
		operations = []string{"Get", "Update"}
	case r.StructName == "Device":
		// Device resource is read-only (List, Get only)
		operations = []string{"List", "Get"}
	default:
		// All other resources have full CRUD
		operations = []string{"List", "Get", "Create", "Update", "Delete"}
	}

	excluded := make(map[string]bool, len(excludedFunctions))
	for _, operation := range excludedFunctions {
		excluded[operation] = true
	}
	filtered := make([]string, 0, len(operations))
	for _, operation := range operations {
		if !excluded[operation] {
			filtered = append(filtered, operation)
		}
	}
	return filtered
}

// HasOperation checks if a resource supports a specific operation.
func HasOperation(r *gounifi.Resource, op string) bool {
	ops := InferOperations(r)
	for _, o := range ops {
		if o == op {
			return true
		}
	}
	return false
}

package resolve

import (
	"strings"

	"github.com/claytono/go-unifi-mcp/internal/tools/generated"
)

// BuildResourceIndex builds a case-insensitive lookup map from list-category tools.
// Keys are lowercase resource names, values are PascalCase resource names.
func BuildResourceIndex(metadata []generated.ToolMetadata) map[string]string {
	index := make(map[string]string)
	for _, meta := range metadata {
		if meta.Category == "list" {
			index[strings.ToLower(meta.Resource)] = meta.Resource
		}
	}
	return index
}

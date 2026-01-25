// Package mcpgen generates MCP tool handlers from UniFi API field definitions.
package mcpgen

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"text/template"

	"github.com/claytono/go-unifi-mcp/internal/gounifi"
	"github.com/iancoleman/strcase"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

// FieldSchema contains schema information for a single field.
type FieldSchema struct {
	Name        string   // JSON field name
	GoName      string   // Go field name
	Type        string   // MCP type: string, number, integer, boolean, array, object
	Description string   // Field description/validation comment
	Pattern     string   // Regex pattern for validation
	Enum        []string // Enum values if applicable
	IsArray     bool     // Whether this is an array type
	ItemType    string   // Type of array items if IsArray
	Required    bool     // Whether field is required (non-omitempty)
}

// ToolInfo contains metadata about a tool to be generated.
type ToolInfo struct {
	Name       string   // e.g., "Network"
	SnakeName  string   // e.g., "network"
	Operations []string // e.g., ["List", "Get", "Create", "Update", "Delete"]
	IsSetting  bool
	IsV2       bool
	Fields     []FieldSchema // Field schemas for create/update operations
}

// GeneratorConfig holds configuration for the generator.
type GeneratorConfig struct {
	FieldsDir string // Path to v1 field JSONs
	V2Dir     string // Path to v2 field JSONs
	OutDir    string // Output directory
}

// Generate generates MCP tool handlers from UniFi API field definitions.
func Generate(cfg GeneratorConfig) error {
	// Load customizer for field processing
	customizer, err := gounifi.NewCodeCustomizer("")
	if err != nil {
		return fmt.Errorf("failed to create customizer: %w", err)
	}

	// Find the versioned fields directory (e.g., .tmp/fields/v9.0.114)
	fieldsDir, err := findFieldsDir(cfg.FieldsDir)
	if err != nil {
		return fmt.Errorf("failed to find fields directory: %w", err)
	}

	// Parse v1 resources from downloaded fields
	v1Resources, err := gounifi.BuildResourcesFromDownloadedFields(fieldsDir, *customizer, false)
	if err != nil {
		return fmt.Errorf("failed to parse v1 resources: %w", err)
	}

	// Parse v2 resources from internal/gounifi/v2/
	v2Resources, err := gounifi.BuildResourcesFromDownloadedFields(cfg.V2Dir, *customizer, true)
	if err != nil {
		return fmt.Errorf("failed to parse v2 resources: %w", err)
	}

	// Combine and process resources
	allResources := make([]*gounifi.Resource, 0, len(v1Resources)+len(v2Resources))
	allResources = append(allResources, v1Resources...)
	allResources = append(allResources, v2Resources...)
	tools := make([]ToolInfo, 0, len(allResources))

	for _, r := range allResources {
		if customizer.IsExcludedFromClient(r.Name()) {
			continue
		}

		tool := ToolInfo{
			Name:       r.StructName,
			SnakeName:  strcase.ToSnake(r.StructName),
			IsSetting:  r.IsSetting(),
			IsV2:       r.IsV2(),
			Operations: InferOperations(r),
			Fields:     extractFieldSchemas(r),
		}
		tools = append(tools, tool)
	}

	// Sort for deterministic output
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Name < tools[j].Name
	})

	// Ensure output directory exists
	if err := os.MkdirAll(cfg.OutDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Render templates
	if err := renderTemplate("templates/tools.go.tmpl", filepath.Join(cfg.OutDir, "tools.gen.go"), tools); err != nil {
		return fmt.Errorf("failed to render tools template: %w", err)
	}

	if err := renderTemplate("templates/registry.go.tmpl", filepath.Join(cfg.OutDir, "registry.gen.go"), tools); err != nil {
		return fmt.Errorf("failed to render registry template: %w", err)
	}

	if err := renderTemplate("templates/metadata.go.tmpl", filepath.Join(cfg.OutDir, "metadata.gen.go"), tools); err != nil {
		return fmt.Errorf("failed to render metadata template: %w", err)
	}

	if err := renderTemplate("templates/handlers.go.tmpl", filepath.Join(cfg.OutDir, "handlers.gen.go"), tools); err != nil {
		return fmt.Errorf("failed to render handlers template: %w", err)
	}

	return nil
}

// findFieldsDir finds the versioned subdirectory in the fields directory.
func findFieldsDir(baseDir string) (string, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if entry.IsDir() && len(entry.Name()) > 0 && entry.Name()[0] == 'v' {
			return filepath.Join(baseDir, entry.Name()), nil
		}
	}

	// If no versioned directory, assume files are directly in baseDir
	return baseDir, nil
}

// renderTemplate renders a template to a file.
func renderTemplate(templatePath, outputPath string, data interface{}) error {
	content, err := templateFS.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", templatePath, err)
	}

	funcMap := template.FuncMap{
		"has": func(needle string, haystack []string) bool {
			for _, s := range haystack {
				if s == needle {
					return true
				}
			}
			return false
		},
		"fieldProperty": fieldPropertyFunc,
	}

	tmpl, err := template.New(filepath.Base(templatePath)).Funcs(funcMap).Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", templatePath, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", templatePath, err)
	}

	// Format the generated Go code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Write unformatted for debugging
		_ = os.WriteFile(outputPath+".unformatted", buf.Bytes(), 0644)
		return fmt.Errorf("failed to format generated code: %w", err)
	}

	if err := os.WriteFile(outputPath, formatted, 0644); err != nil {
		return fmt.Errorf("failed to write output file %s: %w", outputPath, err)
	}

	return nil
}

// extractFieldSchemas extracts field schema information from a Resource.
func extractFieldSchemas(r *gounifi.Resource) []FieldSchema {
	baseType := r.BaseType()
	if baseType == nil || baseType.Fields == nil {
		return nil
	}

	schemas := make([]FieldSchema, 0, len(baseType.Fields))
	for _, f := range baseType.Fields {
		// Skip nil fields
		if f == nil {
			continue
		}
		// Skip internal fields (spacers have names starting with space)
		if f.JSONName == "" || len(f.JSONName) > 0 && f.JSONName[0] == ' ' {
			continue
		}
		// Skip ID field - it's handled separately
		if f.JSONName == "_id" {
			continue
		}

		schema := convertFieldToSchema(f)
		schemas = append(schemas, schema)
	}

	// Sort for deterministic output
	sort.Slice(schemas, func(i, j int) bool {
		return schemas[i].Name < schemas[j].Name
	})

	return schemas
}

// convertFieldToSchema converts a gounifi FieldInfo to a FieldSchema.
func convertFieldToSchema(f *gounifi.FieldInfo) FieldSchema {
	schema := FieldSchema{
		Name:     f.JSONName,
		GoName:   f.FieldName,
		Required: !f.OmitEmpty,
		IsArray:  f.IsArray,
	}

	// Determine MCP type from Go type
	schema.Type, schema.ItemType = goTypeToMCPType(f.FieldType, f.IsArray)

	// FieldValidationComment contains the raw validation pattern from the JSON
	rawPattern := f.FieldValidationComment

	// Extract enum values from validation pattern if it looks like an enum
	if rawPattern != "" && isEnumPattern(rawPattern) {
		schema.Enum = parseEnumValues(rawPattern)
		// Use enum values as description
		schema.Description = "One of: " + rawPattern
	} else if rawPattern != "" && rawPattern != "^$" {
		// Use as regex pattern if not an enum and not just "empty allowed"
		schema.Pattern = rawPattern
	}

	return schema
}

// goTypeToMCPType converts a Go type to an MCP schema type.
func goTypeToMCPType(goType string, isArray bool) (mcpType, itemType string) {
	// Handle array types
	if isArray {
		// Extract element type from slice
		elemType := goType
		if len(goType) > 2 && goType[:2] == "[]" {
			elemType = goType[2:]
		}
		itemType, _ = goTypeToMCPType(elemType, false)
		return "array", itemType
	}

	switch goType {
	case "string":
		return "string", ""
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64":
		return "integer", ""
	case "float32", "float64":
		return "number", ""
	case "bool":
		return "boolean", ""
	default:
		// Complex types become objects
		return "object", ""
	}
}

// isEnumPattern checks if a validation pattern looks like an enum (values separated by |).
func isEnumPattern(pattern string) bool {
	// Enum patterns are simple alternations like "vpn|802.1x|custom" or "tcp|udp"
	if pattern == "" {
		return false
	}

	// If it starts with ^ or ends with $, it's a regex, not a simple enum
	if pattern[0] == '^' || pattern[len(pattern)-1] == '$' {
		return false
	}

	// If it contains regex metacharacters (except | and .), it's not a simple enum
	// Note: . can appear in enum values like "802.1x"
	for _, c := range pattern {
		switch c {
		case '*', '+', '?', '[', ']', '(', ')', '{', '}', '\\':
			return false
		}
	}

	// Must contain at least one |
	for _, c := range pattern {
		if c == '|' {
			return true
		}
	}
	return false
}

// parseEnumValues extracts enum values from a pattern like "a|b|c".
func parseEnumValues(pattern string) []string {
	var values []string
	current := ""
	for _, c := range pattern {
		if c == '|' {
			if current != "" {
				values = append(values, current)
			}
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		values = append(values, current)
	}
	return values
}

// fieldPropertyFunc generates the mcp.With* call for a field schema.
func fieldPropertyFunc(f FieldSchema) string {
	var b bytes.Buffer

	// Determine the With* function based on type
	switch f.Type {
	case "boolean":
		b.WriteString(fmt.Sprintf("mcp.WithBoolean(%q", f.Name))
	case "integer":
		b.WriteString(fmt.Sprintf("mcp.WithNumber(%q", f.Name))
	case "number":
		b.WriteString(fmt.Sprintf("mcp.WithNumber(%q", f.Name))
	case "array":
		b.WriteString(fmt.Sprintf("mcp.WithArray(%q", f.Name))
	case "object":
		b.WriteString(fmt.Sprintf("mcp.WithObject(%q", f.Name))
	default: // string
		b.WriteString(fmt.Sprintf("mcp.WithString(%q", f.Name))
	}

	// Add options
	if f.Description != "" {
		b.WriteString(fmt.Sprintf(", mcp.Description(%q)", f.Description))
	}

	if len(f.Enum) > 0 {
		b.WriteString(", mcp.Enum(")
		for i, v := range f.Enum {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(fmt.Sprintf("%q", v))
		}
		b.WriteString(")")
	}

	if f.Pattern != "" {
		b.WriteString(fmt.Sprintf(", mcp.Pattern(%q)", f.Pattern))
	}

	// Add array item type if applicable
	if f.Type == "array" && f.ItemType != "" {
		switch f.ItemType {
		case "string":
			b.WriteString(", mcp.WithStringItems()")
		case "number", "integer":
			b.WriteString(", mcp.WithNumberItems()")
		case "boolean":
			b.WriteString(", mcp.WithBooleanItems()")
		}
	}

	b.WriteString(")")
	return b.String()
}

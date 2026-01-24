package mcpgen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/claytono/go-unifi-mcp/internal/gounifi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindFieldsDir(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()

	tests := []struct {
		name      string
		setup     func() string
		wantMatch string
		wantErr   bool
	}{
		{
			name: "finds versioned subdirectory",
			setup: func() string {
				baseDir := filepath.Join(tmpDir, "test1")
				vDir := filepath.Join(baseDir, "v9.0.114")
				if err := os.MkdirAll(vDir, 0755); err != nil {
					t.Fatal(err)
				}
				return baseDir
			},
			wantMatch: "v9.0.114",
			wantErr:   false,
		},
		{
			name: "returns base dir if no versioned subdirectory",
			setup: func() string {
				baseDir := filepath.Join(tmpDir, "test2")
				if err := os.MkdirAll(baseDir, 0755); err != nil {
					t.Fatal(err)
				}
				// Create a non-versioned file
				if err := os.WriteFile(filepath.Join(baseDir, "test.json"), []byte("{}"), 0644); err != nil {
					t.Fatal(err)
				}
				return baseDir
			},
			wantMatch: "test2",
			wantErr:   false,
		},
		{
			name: "returns error for non-existent directory",
			setup: func() string {
				return filepath.Join(tmpDir, "nonexistent")
			},
			wantMatch: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := tt.setup()
			got, err := findFieldsDir(baseDir)

			if (err != nil) != tt.wantErr {
				t.Errorf("findFieldsDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !strings.Contains(got, tt.wantMatch) {
				t.Errorf("findFieldsDir() = %v, want to contain %v", got, tt.wantMatch)
			}
		})
	}
}

func TestGenerate_Integration(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Check that field definitions exist
	fieldsDir := ".tmp/fields"
	if _, err := os.Stat("../../" + fieldsDir); os.IsNotExist(err) {
		t.Skip("field definitions not downloaded, run 'task download-fields' first")
	}

	// Create temporary output directory
	outDir := t.TempDir()

	cfg := GeneratorConfig{
		FieldsDir: "../../.tmp/fields",
		V2Dir:     "../../internal/gounifi/v2",
		OutDir:    outDir,
	}

	// Generate
	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Verify files were created
	toolsFile := filepath.Join(outDir, "tools.gen.go")
	registryFile := filepath.Join(outDir, "registry.gen.go")

	if _, err := os.Stat(toolsFile); os.IsNotExist(err) {
		t.Errorf("tools.gen.go was not created")
	}

	if _, err := os.Stat(registryFile); os.IsNotExist(err) {
		t.Errorf("registry.gen.go was not created")
	}

	// Check that the generated files have content
	toolsContent, err := os.ReadFile(toolsFile)
	if err != nil {
		t.Fatalf("Failed to read tools.gen.go: %v", err)
	}
	if len(toolsContent) < 1000 {
		t.Errorf("tools.gen.go seems too small: %d bytes", len(toolsContent))
	}

	registryContent, err := os.ReadFile(registryFile)
	if err != nil {
		t.Fatalf("Failed to read registry.gen.go: %v", err)
	}
	if len(registryContent) < 1000 {
		t.Errorf("registry.gen.go seems too small: %d bytes", len(registryContent))
	}

	// Verify expected content
	if !strings.Contains(string(toolsContent), "package generated") {
		t.Error("tools.gen.go missing package declaration")
	}
	if !strings.Contains(string(registryContent), "RegisterAllTools") {
		t.Error("registry.gen.go missing RegisterAllTools function")
	}

	// Verify generated code compiles
	cmd := exec.Command("go", "build", toolsFile, registryFile)
	cmd.Env = append(os.Environ(),
		"GOPATH="+filepath.Join(os.Getenv("PWD"), "../../.go"),
		"GOMODCACHE="+filepath.Join(os.Getenv("PWD"), "../../.go/pkg/mod"),
		"GOCACHE="+filepath.Join(os.Getenv("PWD"), "../../.go/cache"),
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("Generated code does not compile: %v\n%s", err, output)
	}
}

func TestGenerate_ToolCounts(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Check that field definitions exist
	if _, err := os.Stat("../../.tmp/fields"); os.IsNotExist(err) {
		t.Skip("field definitions not downloaded, run 'task download-fields' first")
	}

	outDir := t.TempDir()

	cfg := GeneratorConfig{
		FieldsDir: "../../.tmp/fields",
		V2Dir:     "../../internal/gounifi/v2",
		OutDir:    outDir,
	}

	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Read the generated tools file
	toolsFile := filepath.Join(outDir, "tools.gen.go")
	content, err := os.ReadFile(toolsFile)
	if err != nil {
		t.Fatalf("Failed to read tools.gen.go: %v", err)
	}

	// Count function definitions - should be at least 200
	funcCount := strings.Count(string(content), "\nfunc ")
	if funcCount < 200 {
		t.Errorf("Expected at least 200 tool functions, got %d", funcCount)
	}
}

func TestGenerate_MissingFieldsDir(t *testing.T) {
	outDir := t.TempDir()

	cfg := GeneratorConfig{
		FieldsDir: "/nonexistent/path",
		V2Dir:     "../../internal/gounifi/v2",
		OutDir:    outDir,
	}

	err := Generate(cfg)
	if err == nil {
		t.Error("Generate() should return error for missing fields dir")
	}
}

// TestGenerate_WithMockFields tests Generate with minimal mock field data.
// This test doesn't require downloading real field definitions.
func TestGenerate_WithMockFields(t *testing.T) {
	// Create temp directory structure for mock fields
	tmpDir := t.TempDir()
	fieldsDir := filepath.Join(tmpDir, "fields", "v1.0.0")
	require.NoError(t, os.MkdirAll(fieldsDir, 0755))

	// Create a minimal mock field JSON file
	// This mimics the format used by gounifi
	mockFieldJSON := `{
		"name": ".{1,256}",
		"enabled": "true|false",
		"vlan_id": "^[0-9]{1,4}$"
	}`
	require.NoError(t, os.WriteFile(
		filepath.Join(fieldsDir, "Network.json"),
		[]byte(mockFieldJSON),
		0644,
	))

	// Create output directory
	outDir := filepath.Join(tmpDir, "output")

	cfg := GeneratorConfig{
		FieldsDir: filepath.Join(tmpDir, "fields"),
		V2Dir:     "../../internal/gounifi/v2",
		OutDir:    outDir,
	}

	// Generate should succeed
	err := Generate(cfg)
	require.NoError(t, err)

	// Verify files were created
	_, err = os.Stat(filepath.Join(outDir, "tools.gen.go"))
	assert.NoError(t, err, "tools.gen.go should exist")

	_, err = os.Stat(filepath.Join(outDir, "registry.gen.go"))
	assert.NoError(t, err, "registry.gen.go should exist")

	// Verify generated code compiles by checking it has expected content
	toolsContent, err := os.ReadFile(filepath.Join(outDir, "tools.gen.go"))
	require.NoError(t, err)
	assert.Contains(t, string(toolsContent), "package generated")
	assert.Contains(t, string(toolsContent), "Network") // Our mock resource
}

func TestGenerate_MissingV2Dir(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("skipping test requiring field definitions in short mode")
	}

	// Check that field definitions exist
	if _, err := os.Stat("../../.tmp/fields"); os.IsNotExist(err) {
		t.Skip("field definitions not downloaded, run 'task download-fields' first")
	}

	outDir := t.TempDir()

	cfg := GeneratorConfig{
		FieldsDir: "../../.tmp/fields",
		V2Dir:     "/nonexistent/path",
		OutDir:    outDir,
	}

	err := Generate(cfg)
	if err == nil {
		t.Error("Generate() should return error for missing V2 dir")
	}
}

func TestRenderTemplate_InvalidTemplate(t *testing.T) {
	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "test.go")

	// Test with a template that doesn't exist
	err := renderTemplate("templates/nonexistent.tmpl", outPath, nil)
	if err == nil {
		t.Error("renderTemplate() should return error for non-existent template")
	}
	if !strings.Contains(err.Error(), "failed to read template") {
		t.Errorf("Expected 'failed to read template' error, got: %v", err)
	}
}

func TestRenderTemplate_WriteError(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	// Create a readonly directory
	tmpDir := t.TempDir()
	readonlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.Mkdir(readonlyDir, 0555); err != nil {
		t.Skip("failed to create readonly directory")
	}
	defer func() { _ = os.Chmod(readonlyDir, 0755) }()

	outPath := filepath.Join(readonlyDir, "test.go")

	// Use a valid template with simple data
	data := []ToolInfo{
		{Name: "Test", SnakeName: "test", Operations: []string{"Get"}, IsSetting: false},
	}

	err := renderTemplate("templates/tools.go.tmpl", outPath, data)
	if err == nil {
		t.Error("renderTemplate() should return error when write fails")
	}
}

func TestGenerate_OutputDirCreationFailure(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	// Check that field definitions exist
	if _, err := os.Stat("../../.tmp/fields"); os.IsNotExist(err) {
		t.Skip("field definitions not downloaded, run 'task download-fields' first")
	}

	// Use a path that will fail to create (inside a file)
	tmpFile := filepath.Join(t.TempDir(), "afile")
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := GeneratorConfig{
		FieldsDir: "../../.tmp/fields",
		V2Dir:     "../../internal/gounifi/v2",
		OutDir:    filepath.Join(tmpFile, "subdir"), // Can't create dir inside a file
	}

	err := Generate(cfg)
	if err == nil {
		t.Error("Generate() should return error when output dir creation fails")
	}
}

func TestGoTypeToMCPType(t *testing.T) {
	tests := []struct {
		name         string
		goType       string
		isArray      bool
		wantMCPType  string
		wantItemType string
	}{
		{
			name:         "string type",
			goType:       "string",
			isArray:      false,
			wantMCPType:  "string",
			wantItemType: "",
		},
		{
			name:         "int type",
			goType:       "int",
			isArray:      false,
			wantMCPType:  "integer",
			wantItemType: "",
		},
		{
			name:         "int8 type",
			goType:       "int8",
			isArray:      false,
			wantMCPType:  "integer",
			wantItemType: "",
		},
		{
			name:         "int16 type",
			goType:       "int16",
			isArray:      false,
			wantMCPType:  "integer",
			wantItemType: "",
		},
		{
			name:         "int32 type",
			goType:       "int32",
			isArray:      false,
			wantMCPType:  "integer",
			wantItemType: "",
		},
		{
			name:         "int64 type",
			goType:       "int64",
			isArray:      false,
			wantMCPType:  "integer",
			wantItemType: "",
		},
		{
			name:         "uint type",
			goType:       "uint",
			isArray:      false,
			wantMCPType:  "integer",
			wantItemType: "",
		},
		{
			name:         "uint8 type",
			goType:       "uint8",
			isArray:      false,
			wantMCPType:  "integer",
			wantItemType: "",
		},
		{
			name:         "uint16 type",
			goType:       "uint16",
			isArray:      false,
			wantMCPType:  "integer",
			wantItemType: "",
		},
		{
			name:         "uint32 type",
			goType:       "uint32",
			isArray:      false,
			wantMCPType:  "integer",
			wantItemType: "",
		},
		{
			name:         "uint64 type",
			goType:       "uint64",
			isArray:      false,
			wantMCPType:  "integer",
			wantItemType: "",
		},
		{
			name:         "float32 type",
			goType:       "float32",
			isArray:      false,
			wantMCPType:  "number",
			wantItemType: "",
		},
		{
			name:         "float64 type",
			goType:       "float64",
			isArray:      false,
			wantMCPType:  "number",
			wantItemType: "",
		},
		{
			name:         "bool type",
			goType:       "bool",
			isArray:      false,
			wantMCPType:  "boolean",
			wantItemType: "",
		},
		{
			name:         "struct type becomes object",
			goType:       "SomeStruct",
			isArray:      false,
			wantMCPType:  "object",
			wantItemType: "",
		},
		{
			name:         "array of strings with isArray flag",
			goType:       "string",
			isArray:      true,
			wantMCPType:  "array",
			wantItemType: "string",
		},
		{
			name:         "array of ints with isArray flag",
			goType:       "int",
			isArray:      true,
			wantMCPType:  "array",
			wantItemType: "integer",
		},
		{
			name:         "array with slice prefix []string",
			goType:       "[]string",
			isArray:      true,
			wantMCPType:  "array",
			wantItemType: "string",
		},
		{
			name:         "array with slice prefix []int",
			goType:       "[]int",
			isArray:      true,
			wantMCPType:  "array",
			wantItemType: "integer",
		},
		{
			name:         "array with slice prefix []bool",
			goType:       "[]bool",
			isArray:      true,
			wantMCPType:  "array",
			wantItemType: "boolean",
		},
		{
			name:         "pointer type becomes object",
			goType:       "*string",
			isArray:      false,
			wantMCPType:  "object",
			wantItemType: "",
		},
		{
			name:         "pointer to int becomes object",
			goType:       "*int",
			isArray:      false,
			wantMCPType:  "object",
			wantItemType: "",
		},
		{
			name:         "map type becomes object",
			goType:       "map[string]interface{}",
			isArray:      false,
			wantMCPType:  "object",
			wantItemType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotItemType := goTypeToMCPType(tt.goType, tt.isArray)
			assert.Equal(t, tt.wantMCPType, gotType, "MCP type mismatch")
			assert.Equal(t, tt.wantItemType, gotItemType, "item type mismatch")
		})
	}
}

func TestIsEnumPattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    bool
	}{
		{
			name:    "simple enum with pipe",
			pattern: "tcp|udp",
			want:    true,
		},
		{
			name:    "enum with dots",
			pattern: "vpn|802.1x|custom",
			want:    true,
		},
		{
			name:    "starts with caret - regex",
			pattern: "^[a-z]+$",
			want:    false,
		},
		{
			name:    "has regex brackets",
			pattern: "[0-9]+",
			want:    false,
		},
		{
			name:    "has asterisk - regex",
			pattern: "value*",
			want:    false,
		},
		{
			name:    "has plus - regex",
			pattern: "value+",
			want:    false,
		},
		{
			name:    "has question mark - regex",
			pattern: "value?",
			want:    false,
		},
		{
			name:    "has parentheses - regex",
			pattern: "(a|b)",
			want:    false,
		},
		{
			name:    "has curly braces - regex",
			pattern: "a{2,3}",
			want:    false,
		},
		{
			name:    "has backslash - regex",
			pattern: "\\d+",
			want:    false,
		},
		{
			name:    "single value without pipe",
			pattern: "one",
			want:    false,
		},
		{
			name:    "empty string",
			pattern: "",
			want:    false,
		},
		{
			name:    "ends with dollar - regex",
			pattern: "value$",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isEnumPattern(tt.pattern)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseEnumValues(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    []string
	}{
		{
			name:    "multiple values",
			pattern: "a|b|c",
			want:    []string{"a", "b", "c"},
		},
		{
			name:    "two values",
			pattern: "tcp|udp",
			want:    []string{"tcp", "udp"},
		},
		{
			name:    "single value",
			pattern: "single",
			want:    []string{"single"},
		},
		{
			name:    "empty string",
			pattern: "",
			want:    nil,
		},
		{
			name:    "values with dots",
			pattern: "802.1x|vpn",
			want:    []string{"802.1x", "vpn"},
		},
		{
			name:    "consecutive pipes filters empty",
			pattern: "a||b",
			want:    []string{"a", "b"},
		},
		{
			name:    "leading pipe filters empty",
			pattern: "|a|b",
			want:    []string{"a", "b"},
		},
		{
			name:    "trailing pipe filters empty",
			pattern: "a|b|",
			want:    []string{"a", "b"},
		},
		{
			name:    "only pipes",
			pattern: "||",
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseEnumValues(tt.pattern)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConvertFieldToSchema(t *testing.T) {
	tests := []struct {
		name  string
		field *gounifi.FieldInfo
		want  FieldSchema
	}{
		{
			name: "string field with no validation",
			field: &gounifi.FieldInfo{
				FieldName: "Name",
				JSONName:  "name",
				FieldType: "string",
				OmitEmpty: true,
				IsArray:   false,
			},
			want: FieldSchema{
				Name:     "name",
				GoName:   "Name",
				Type:     "string",
				Required: false,
				IsArray:  false,
			},
		},
		{
			name: "integer field required",
			field: &gounifi.FieldInfo{
				FieldName: "Port",
				JSONName:  "port",
				FieldType: "int",
				OmitEmpty: false,
				IsArray:   false,
			},
			want: FieldSchema{
				Name:     "port",
				GoName:   "Port",
				Type:     "integer",
				Required: true,
				IsArray:  false,
			},
		},
		{
			name: "field with enum pattern",
			field: &gounifi.FieldInfo{
				FieldName:              "Protocol",
				JSONName:               "protocol",
				FieldType:              "string",
				FieldValidationComment: "tcp|udp",
				OmitEmpty:              true,
				IsArray:                false,
			},
			want: FieldSchema{
				Name:        "protocol",
				GoName:      "Protocol",
				Type:        "string",
				Description: "One of: tcp|udp",
				Enum:        []string{"tcp", "udp"},
				Required:    false,
				IsArray:     false,
			},
		},
		{
			name: "field with regex pattern",
			field: &gounifi.FieldInfo{
				FieldName:              "IPAddress",
				JSONName:               "ip_address",
				FieldType:              "string",
				FieldValidationComment: "^[0-9.]+$",
				OmitEmpty:              true,
				IsArray:                false,
			},
			want: FieldSchema{
				Name:     "ip_address",
				GoName:   "IPAddress",
				Type:     "string",
				Pattern:  "^[0-9.]+$",
				Required: false,
				IsArray:  false,
			},
		},
		{
			name: "array field",
			field: &gounifi.FieldInfo{
				FieldName: "Tags",
				JSONName:  "tags",
				FieldType: "string",
				OmitEmpty: true,
				IsArray:   true,
			},
			want: FieldSchema{
				Name:     "tags",
				GoName:   "Tags",
				Type:     "array",
				ItemType: "string",
				Required: false,
				IsArray:  true,
			},
		},
		{
			name: "field with empty allowed pattern is ignored",
			field: &gounifi.FieldInfo{
				FieldName:              "Optional",
				JSONName:               "optional",
				FieldType:              "string",
				FieldValidationComment: "^$",
				OmitEmpty:              true,
				IsArray:                false,
			},
			want: FieldSchema{
				Name:     "optional",
				GoName:   "Optional",
				Type:     "string",
				Required: false,
				IsArray:  false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertFieldToSchema(tt.field)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractFieldSchemas(t *testing.T) {
	t.Run("nil base type returns nil", func(t *testing.T) {
		r := &gounifi.Resource{
			StructName: "Test",
			Types:      map[string]*gounifi.FieldInfo{},
		}
		got := extractFieldSchemas(r)
		assert.Nil(t, got)
	})

	t.Run("nil fields in base type returns nil", func(t *testing.T) {
		r := &gounifi.Resource{
			StructName: "Test",
			Types: map[string]*gounifi.FieldInfo{
				"Test": {
					FieldName: "Test",
					Fields:    nil,
				},
			},
		}
		got := extractFieldSchemas(r)
		assert.Nil(t, got)
	})

	t.Run("skips nil fields in map", func(t *testing.T) {
		r := &gounifi.Resource{
			StructName: "Test",
			Types: map[string]*gounifi.FieldInfo{
				"Test": {
					FieldName: "Test",
					Fields: map[string]*gounifi.FieldInfo{
						"Name": {
							FieldName: "Name",
							JSONName:  "name",
							FieldType: "string",
							OmitEmpty: true,
						},
						"Spacer": nil,
					},
				},
			},
		}
		got := extractFieldSchemas(r)
		require.Len(t, got, 1)
		assert.Equal(t, "name", got[0].Name)
	})

	t.Run("skips internal fields with space prefix", func(t *testing.T) {
		r := &gounifi.Resource{
			StructName: "Test",
			Types: map[string]*gounifi.FieldInfo{
				"Test": {
					FieldName: "Test",
					Fields: map[string]*gounifi.FieldInfo{
						"Name": {
							FieldName: "Name",
							JSONName:  "name",
							FieldType: "string",
							OmitEmpty: true,
						},
						"Internal": {
							FieldName: "Internal",
							JSONName:  " _internal", // JSONName has space prefix - this is what the code checks
							FieldType: "string",
							OmitEmpty: true,
						},
					},
				},
			},
		}
		got := extractFieldSchemas(r)
		require.Len(t, got, 1)
		assert.Equal(t, "name", got[0].Name)
	})

	t.Run("skips fields with empty JSONName", func(t *testing.T) {
		r := &gounifi.Resource{
			StructName: "Test",
			Types: map[string]*gounifi.FieldInfo{
				"Test": {
					FieldName: "Test",
					Fields: map[string]*gounifi.FieldInfo{
						"Name": {
							FieldName: "Name",
							JSONName:  "name",
							FieldType: "string",
							OmitEmpty: true,
						},
						"Empty": {
							FieldName: "Empty",
							JSONName:  "",
							FieldType: "string",
							OmitEmpty: true,
						},
					},
				},
			},
		}
		got := extractFieldSchemas(r)
		require.Len(t, got, 1)
		assert.Equal(t, "name", got[0].Name)
	})

	t.Run("skips _id field", func(t *testing.T) {
		r := &gounifi.Resource{
			StructName: "Test",
			Types: map[string]*gounifi.FieldInfo{
				"Test": {
					FieldName: "Test",
					Fields: map[string]*gounifi.FieldInfo{
						"ID": {
							FieldName: "ID",
							JSONName:  "_id",
							FieldType: "string",
							OmitEmpty: true,
						},
						"Name": {
							FieldName: "Name",
							JSONName:  "name",
							FieldType: "string",
							OmitEmpty: true,
						},
					},
				},
			},
		}
		got := extractFieldSchemas(r)
		require.Len(t, got, 1)
		assert.Equal(t, "name", got[0].Name)
	})

	t.Run("sorts fields by name", func(t *testing.T) {
		r := &gounifi.Resource{
			StructName: "Test",
			Types: map[string]*gounifi.FieldInfo{
				"Test": {
					FieldName: "Test",
					Fields: map[string]*gounifi.FieldInfo{
						"Zebra": {
							FieldName: "Zebra",
							JSONName:  "zebra",
							FieldType: "string",
							OmitEmpty: true,
						},
						"Alpha": {
							FieldName: "Alpha",
							JSONName:  "alpha",
							FieldType: "string",
							OmitEmpty: true,
						},
						"Middle": {
							FieldName: "Middle",
							JSONName:  "middle",
							FieldType: "string",
							OmitEmpty: true,
						},
					},
				},
			},
		}
		got := extractFieldSchemas(r)
		require.Len(t, got, 3)
		assert.Equal(t, "alpha", got[0].Name)
		assert.Equal(t, "middle", got[1].Name)
		assert.Equal(t, "zebra", got[2].Name)
	})
}

func TestFieldPropertyFunc(t *testing.T) {
	tests := []struct {
		name  string
		field FieldSchema
		want  string
	}{
		{
			name: "boolean field",
			field: FieldSchema{
				Name: "enabled",
				Type: "boolean",
			},
			want: `mcp.WithBoolean("enabled")`,
		},
		{
			name: "integer field",
			field: FieldSchema{
				Name: "port",
				Type: "integer",
			},
			want: `mcp.WithNumber("port")`,
		},
		{
			name: "number field",
			field: FieldSchema{
				Name: "rate",
				Type: "number",
			},
			want: `mcp.WithNumber("rate")`,
		},
		{
			name: "array field",
			field: FieldSchema{
				Name: "items",
				Type: "array",
			},
			want: `mcp.WithArray("items")`,
		},
		{
			name: "object field",
			field: FieldSchema{
				Name: "config",
				Type: "object",
			},
			want: `mcp.WithObject("config")`,
		},
		{
			name: "string field (default)",
			field: FieldSchema{
				Name: "name",
				Type: "string",
			},
			want: `mcp.WithString("name")`,
		},
		{
			name: "field with description",
			field: FieldSchema{
				Name:        "protocol",
				Type:        "string",
				Description: "The protocol to use",
			},
			want: `mcp.WithString("protocol", mcp.Description("The protocol to use"))`,
		},
		{
			name: "field with enum",
			field: FieldSchema{
				Name: "protocol",
				Type: "string",
				Enum: []string{"tcp", "udp"},
			},
			want: `mcp.WithString("protocol", mcp.Enum("tcp", "udp"))`,
		},
		{
			name: "field with pattern",
			field: FieldSchema{
				Name:    "ip",
				Type:    "string",
				Pattern: "^[0-9.]+$",
			},
			want: `mcp.WithString("ip", mcp.Pattern("^[0-9.]+$"))`,
		},
		{
			name: "array with string items",
			field: FieldSchema{
				Name:     "tags",
				Type:     "array",
				ItemType: "string",
			},
			want: `mcp.WithArray("tags", mcp.WithStringItems())`,
		},
		{
			name: "array with number items",
			field: FieldSchema{
				Name:     "values",
				Type:     "array",
				ItemType: "number",
			},
			want: `mcp.WithArray("values", mcp.WithNumberItems())`,
		},
		{
			name: "array with integer items",
			field: FieldSchema{
				Name:     "ports",
				Type:     "array",
				ItemType: "integer",
			},
			want: `mcp.WithArray("ports", mcp.WithNumberItems())`,
		},
		{
			name: "array with boolean items",
			field: FieldSchema{
				Name:     "flags",
				Type:     "array",
				ItemType: "boolean",
			},
			want: `mcp.WithArray("flags", mcp.WithBooleanItems())`,
		},
		{
			name: "array with object items has no item type",
			field: FieldSchema{
				Name:     "configs",
				Type:     "array",
				ItemType: "object",
			},
			want: `mcp.WithArray("configs")`,
		},
		{
			name: "field with all options",
			field: FieldSchema{
				Name:        "protocol",
				Type:        "string",
				Description: "One of: tcp|udp",
				Enum:        []string{"tcp", "udp"},
				Pattern:     "",
			},
			want: `mcp.WithString("protocol", mcp.Description("One of: tcp|udp"), mcp.Enum("tcp", "udp"))`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fieldPropertyFunc(tt.field)
			assert.Equal(t, tt.want, got)
		})
	}
}

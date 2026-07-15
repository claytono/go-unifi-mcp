package mcpgen

import (
	"reflect"
	"testing"

	"github.com/claytono/go-unifi-mcp/internal/gounifi"
)

func TestInferOperations(t *testing.T) {
	tests := []struct {
		name       string
		structName string
		want       []string
	}{
		{
			name:       "regular resource has full CRUD",
			structName: "Network",
			want:       []string{"List", "Get", "Create", "Update", "Delete"},
		},
		{
			name:       "setting resource has only Get and Update",
			structName: "SettingMgmt",
			want:       []string{"Get", "Update"},
		},
		{
			name:       "device resource has only List and Get",
			structName: "Device",
			want:       []string{"List", "Get"},
		},
		{
			name:       "another regular resource",
			structName: "FirewallRule",
			want:       []string{"List", "Get", "Create", "Update", "Delete"},
		},
		{
			name:       "another setting",
			structName: "SettingUsg",
			want:       []string{"Get", "Update"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gounifi.NewResource(tt.structName, "test")
			got := InferOperations(r)

			if len(got) != len(tt.want) {
				t.Errorf("InferOperations() = %v, want %v", got, tt.want)
				return
			}

			for i, op := range got {
				if op != tt.want[i] {
					t.Errorf("InferOperations()[%d] = %v, want %v", i, op, tt.want[i])
				}
			}
		})
	}
}

func TestInferOperations_ExcludedFunctions(t *testing.T) {
	r := gounifi.NewResource("ContentFiltering", "test")

	got := InferOperations(r, "Get")

	want := []string{"List", "Create", "Update", "Delete"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("InferOperations() = %v, want %v", got, want)
	}
}

func TestHasOperation(t *testing.T) {
	tests := []struct {
		name       string
		structName string
		operation  string
		want       bool
	}{
		{
			name:       "regular resource has List",
			structName: "Network",
			operation:  "List",
			want:       true,
		},
		{
			name:       "regular resource has Delete",
			structName: "Network",
			operation:  "Delete",
			want:       true,
		},
		{
			name:       "setting does not have List",
			structName: "SettingMgmt",
			operation:  "List",
			want:       false,
		},
		{
			name:       "setting has Get",
			structName: "SettingMgmt",
			operation:  "Get",
			want:       true,
		},
		{
			name:       "device does not have Create",
			structName: "Device",
			operation:  "Create",
			want:       false,
		},
		{
			name:       "device has List",
			structName: "Device",
			operation:  "List",
			want:       true,
		},
		{
			name:       "unknown operation returns false",
			structName: "Network",
			operation:  "Unknown",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gounifi.NewResource(tt.structName, "test")
			got := HasOperation(r, tt.operation)

			if got != tt.want {
				t.Errorf("HasOperation() = %v, want %v", got, tt.want)
			}
		})
	}
}

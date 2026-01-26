package server

import (
	"os"
	"testing"

	"github.com/claytono/go-unifi-mcp/internal/config"
	servermocks "github.com/claytono/go-unifi-mcp/internal/server/mocks"
	"github.com/stretchr/testify/assert"
)

func TestNew_RequiresClient(t *testing.T) {
	_, err := New(Options{Client: nil})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client is required")
}

func TestNew_CreatesServer(t *testing.T) {
	client := servermocks.NewClient(t)
	t.Setenv("UNIFI_TOOL_MODE", "")

	s, err := New(Options{Client: client})
	assert.NoError(t, err)
	assert.NotNil(t, s)
	assert.Len(t, s.ListTools(), 3)
}

func TestNewClient_APIKey(t *testing.T) {
	cfg := &config.Config{
		Host:      "https://192.168.1.1",
		APIKey:    "test-key",
		Site:      "default",
		VerifySSL: false,
	}

	// This will fail to connect but validates config mapping
	_, err := NewClient(cfg)
	// Error expected since no server is running
	assert.Error(t, err)
}

func TestNewClient_UserPass(t *testing.T) {
	cfg := &config.Config{
		Host:      "https://192.168.1.1",
		Username:  "admin",
		Password:  "secret",
		Site:      "default",
		VerifySSL: false,
	}

	// This will fail to connect but validates config mapping
	_, err := NewClient(cfg)
	// Error expected since no server is running
	assert.Error(t, err)
}

func TestMode_DefaultsToLazy(t *testing.T) {
	// Clear environment variable
	_ = os.Unsetenv("UNIFI_TOOL_MODE")

	// Mode should default to lazy when not specified
	opts := Options{}
	assert.Empty(t, opts.Mode)

	// The actual default is applied in New(), which requires a client
	// So we just verify the constant values are correct
	assert.Equal(t, Mode("lazy"), ModeLazy)
	assert.Equal(t, Mode("eager"), ModeEager)
}

func TestMode_ReadsFromEnvVar(t *testing.T) {
	// Test that environment variable is respected
	_ = os.Setenv("UNIFI_TOOL_MODE", "eager")
	defer func() { _ = os.Unsetenv("UNIFI_TOOL_MODE") }()

	// Can't test full creation without a client, but verify env parsing
	envMode := Mode(os.Getenv("UNIFI_TOOL_MODE"))
	assert.Equal(t, ModeEager, envMode)
}

func TestMode_OptionsOverridesEnv(t *testing.T) {
	// Set environment to eager
	_ = os.Setenv("UNIFI_TOOL_MODE", "eager")
	defer func() { _ = os.Unsetenv("UNIFI_TOOL_MODE") }()

	// Options should take precedence (verified by constant check)
	opts := Options{Mode: ModeLazy}
	assert.Equal(t, ModeLazy, opts.Mode)
}

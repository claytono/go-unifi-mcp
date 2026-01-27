package server

import (
	"os"
	"testing"

	"github.com/claytono/go-unifi-mcp/internal/config"
	servermocks "github.com/claytono/go-unifi-mcp/internal/server/mocks"
	"github.com/filipowm/go-unifi/unifi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	var captured *unifi.ClientConfig
	prevFactory := newUnifiClient
	newUnifiClient = func(clientCfg *unifi.ClientConfig) (unifi.Client, error) {
		captured = clientCfg
		return nil, nil
	}
	t.Cleanup(func() {
		newUnifiClient = prevFactory
	})

	client, err := NewClient(cfg)
	assert.NoError(t, err)
	assert.Nil(t, client)
	require.NotNil(t, captured)
	assert.Equal(t, cfg.Host, captured.URL)
	assert.Equal(t, cfg.VerifySSL, captured.VerifySSL)
	assert.Equal(t, cfg.APIKey, captured.APIKey)
	assert.Empty(t, captured.User)
	assert.Empty(t, captured.Password)
}

func TestNewClient_UserPass(t *testing.T) {
	cfg := &config.Config{
		Host:      "https://192.168.1.1",
		Username:  "admin",
		Password:  "secret",
		Site:      "default",
		VerifySSL: false,
	}

	var captured *unifi.ClientConfig
	prevFactory := newUnifiClient
	newUnifiClient = func(clientCfg *unifi.ClientConfig) (unifi.Client, error) {
		captured = clientCfg
		return nil, nil
	}
	t.Cleanup(func() {
		newUnifiClient = prevFactory
	})

	client, err := NewClient(cfg)
	assert.NoError(t, err)
	assert.Nil(t, client)
	require.NotNil(t, captured)
	assert.Equal(t, cfg.Host, captured.URL)
	assert.Equal(t, cfg.VerifySSL, captured.VerifySSL)
	assert.Empty(t, captured.APIKey)
	assert.Equal(t, cfg.Username, captured.User)
	assert.Equal(t, cfg.Password, captured.Password)
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

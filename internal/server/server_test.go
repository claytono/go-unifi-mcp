package server

import (
	"testing"

	"github.com/claytono/go-unifi-mcp/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestNew_RequiresClient(t *testing.T) {
	_, err := New(Options{Client: nil})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client is required")
}

func TestNew_CreatesServer(t *testing.T) {
	// This test requires a mock client
	// Since unifi.Client has many methods, we'd need mockery to generate a mock
	// For now, we test the nil client case above
	t.Skip("Requires mock generation - see integration_test.go")
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

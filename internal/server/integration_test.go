package server

import (
	"testing"

	servermocks "github.com/claytono/go-unifi-mcp/internal/server/mocks"
	"github.com/stretchr/testify/assert"
)

// TestServerToolCount verifies all tools are registered.
func TestServerToolCount(t *testing.T) {
	client := servermocks.NewClient(t)

	s, err := New(Options{Client: client, Mode: ModeEager})
	assert.NoError(t, err)
	assert.NotNil(t, s)
	assert.Len(t, s.ListTools(), 242)
}

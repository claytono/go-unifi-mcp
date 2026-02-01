package main

import (
	"bytes"
	"errors"
	"log"
	"testing"

	"github.com/claytono/go-unifi-mcp/internal/config"
	"github.com/claytono/go-unifi-mcp/internal/server"
	"github.com/filipowm/go-unifi/unifi"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunLoadConfigError(t *testing.T) {
	expectedErr := errors.New("load")
	r := baseRunner()
	r.loadConfig = func() (*config.Config, error) {
		return nil, expectedErr
	}

	err := runWith(r)
	require.ErrorIs(t, err, expectedErr)
}

func TestRunNewClientError(t *testing.T) {
	expectedErr := errors.New("client")
	r := baseRunner()
	r.newClient = func(cfg *config.Config) (unifi.Client, error) {
		return nil, expectedErr
	}

	err := runWith(r)
	require.ErrorIs(t, err, expectedErr)
}

func TestRunNewServerError(t *testing.T) {
	expectedErr := errors.New("server")
	r := baseRunner()
	r.newServer = func(opts server.Options) (*mcpserver.MCPServer, error) {
		return nil, expectedErr
	}

	err := runWith(r)
	require.ErrorIs(t, err, expectedErr)
}

func TestRunServeError(t *testing.T) {
	expectedErr := errors.New("serve")
	r := baseRunner()
	r.serve = func(s *mcpserver.MCPServer) error {
		return expectedErr
	}

	err := runWith(r)
	require.ErrorIs(t, err, expectedErr)
}

func TestRunSuccess(t *testing.T) {
	r := baseRunner()
	called := false
	r.serve = func(s *mcpserver.MCPServer) error {
		called = true
		return nil
	}

	err := runWith(r)
	require.NoError(t, err)
	require.True(t, called)
}

func TestMainLogsAndExitsOnError(t *testing.T) {
	expectedErr := errors.New("boom")
	r := baseRunner()
	buf := &bytes.Buffer{}
	logger := log.New(buf, "", 0)
	r.loadConfig = func() (*config.Config, error) {
		return nil, expectedErr
	}
	exited := false
	exitCode := 0
	exitFn := func(code int) {
		exited = true
		exitCode = code
	}

	mainWith(r, exitFn, logger, []string{"go-unifi-mcp"}, buf)
	require.True(t, exited)
	require.Equal(t, 1, exitCode)
	require.Contains(t, buf.String(), "Error: boom")
}

func TestMainPrintsUsageOnError(t *testing.T) {
	r := baseRunner()
	r.loadConfig = func() (*config.Config, error) {
		return nil, errors.New("UNIFI_HOST environment variable is required")
	}
	buf := &bytes.Buffer{}
	logger := log.New(buf, "", 0)
	exitCode := -1
	exitFn := func(code int) { exitCode = code }

	mainWith(r, exitFn, logger, []string{"go-unifi-mcp"}, buf)
	require.Equal(t, 1, exitCode)
	output := buf.String()
	assert.Contains(t, output, "UNIFI_HOST")
	assert.Contains(t, output, "Usage:")
	assert.Contains(t, output, "Error:")
}

func TestVersionFlag(t *testing.T) {
	r := baseRunner()
	buf := &bytes.Buffer{}
	logger := log.New(buf, "", 0)
	exitCode := -1
	exitFn := func(code int) { exitCode = code }

	mainWith(r, exitFn, logger, []string{"go-unifi-mcp", "--version"}, buf)
	require.Equal(t, 0, exitCode)
	assert.Contains(t, buf.String(), "go-unifi-mcp")
	assert.Contains(t, buf.String(), server.Version)
}

func TestHelpFlag(t *testing.T) {
	r := baseRunner()
	buf := &bytes.Buffer{}
	logger := log.New(buf, "", 0)
	exitCode := -1
	exitFn := func(code int) { exitCode = code }

	mainWith(r, exitFn, logger, []string{"go-unifi-mcp", "--help"}, buf)
	require.Equal(t, 0, exitCode)
	output := buf.String()
	assert.Contains(t, output, "MCP server for UniFi Network Controller")
	assert.Contains(t, output, "Usage:")
	assert.Contains(t, output, "UNIFI_HOST")
	assert.Contains(t, output, "UNIFI_API_KEY")
	assert.Contains(t, output, "UNIFI_LOG_LEVEL")
	assert.Contains(t, output, "UNIFI_TOOL_MODE")
}

func TestUnknownFlagExitsWithCode2(t *testing.T) {
	r := baseRunner()
	buf := &bytes.Buffer{}
	logger := log.New(buf, "", 0)
	exitCode := -1
	exitFn := func(code int) { exitCode = code }

	mainWith(r, exitFn, logger, []string{"go-unifi-mcp", "--bogus"}, buf)
	require.Equal(t, 2, exitCode)
}

func TestMainNoUsageOnRuntimeError(t *testing.T) {
	r := baseRunner()
	r.serve = func(s *mcpserver.MCPServer) error {
		return errors.New("connection lost")
	}
	buf := &bytes.Buffer{}
	logger := log.New(buf, "", 0)
	exitCode := -1
	exitFn := func(code int) { exitCode = code }

	mainWith(r, exitFn, logger, []string{"go-unifi-mcp"}, buf)
	require.Equal(t, 1, exitCode)
	output := buf.String()
	assert.Contains(t, output, "Error: connection lost")
	assert.NotContains(t, output, "Usage:")
}

func TestMainExitsOnError(t *testing.T) {
	t.Setenv("UNIFI_HOST", "")
	t.Setenv("UNIFI_API_KEY", "")
	t.Setenv("UNIFI_USERNAME", "")
	t.Setenv("UNIFI_PASSWORD", "")

	r := defaultRunner()
	buf := &bytes.Buffer{}
	logger := log.New(buf, "", 0)
	exited := false
	exitCode := 0
	exitFn := func(code int) {
		exited = true
		exitCode = code
	}

	mainWith(r, exitFn, logger, []string{"go-unifi-mcp"}, buf)
	require.True(t, exited)
	require.Equal(t, 1, exitCode)
}

func TestDefaultRunner(t *testing.T) {
	r := defaultRunner()
	require.NotNil(t, r.loadConfig)
	require.NotNil(t, r.newClient)
	require.NotNil(t, r.newServer)
	require.NotNil(t, r.serve)
}

func baseRunner() runner {
	return runner{
		loadConfig: func() (*config.Config, error) {
			return &config.Config{}, nil
		},
		newClient: func(cfg *config.Config) (unifi.Client, error) {
			return nil, nil
		},
		newServer: func(opts server.Options) (*mcpserver.MCPServer, error) {
			return nil, nil
		},
		serve: func(s *mcpserver.MCPServer) error {
			return nil
		},
	}
}

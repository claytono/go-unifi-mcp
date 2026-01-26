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

	mainWith(r, exitFn, logger)
	require.True(t, exited)
	require.Equal(t, 1, exitCode)
	require.Contains(t, buf.String(), "Error: boom")
}

func TestMainExitsOnError(t *testing.T) {
	t.Setenv("UNIFI_HOST", "")
	t.Setenv("UNIFI_API_KEY", "")
	t.Setenv("UNIFI_USERNAME", "")
	t.Setenv("UNIFI_PASSWORD", "")

	originalExit := exit
	exited := false
	exitCode := 0
	exit = func(code int) {
		exited = true
		exitCode = code
	}
	t.Cleanup(func() {
		exit = originalExit
	})

	main()
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

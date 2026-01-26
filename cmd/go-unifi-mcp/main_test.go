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
	restore := stubRunDeps()
	loadConfig = func() (*config.Config, error) {
		return nil, expectedErr
	}
	defer restore()

	err := run()
	require.ErrorIs(t, err, expectedErr)
}

func TestRunNewClientError(t *testing.T) {
	expectedErr := errors.New("client")
	restore := stubRunDeps()
	loadConfig = func() (*config.Config, error) {
		return &config.Config{}, nil
	}
	newClient = func(cfg *config.Config) (unifi.Client, error) {
		return nil, expectedErr
	}
	defer restore()

	err := run()
	require.ErrorIs(t, err, expectedErr)
}

func TestRunNewServerError(t *testing.T) {
	expectedErr := errors.New("server")
	restore := stubRunDeps()
	loadConfig = func() (*config.Config, error) {
		return &config.Config{}, nil
	}
	newClient = func(cfg *config.Config) (unifi.Client, error) {
		return nil, nil
	}
	newServer = func(opts server.Options) (*mcpserver.MCPServer, error) {
		return nil, expectedErr
	}
	defer restore()

	err := run()
	require.ErrorIs(t, err, expectedErr)
}

func TestRunServeError(t *testing.T) {
	expectedErr := errors.New("serve")
	restore := stubRunDeps()
	loadConfig = func() (*config.Config, error) {
		return &config.Config{}, nil
	}
	newClient = func(cfg *config.Config) (unifi.Client, error) {
		return nil, nil
	}
	newServer = func(opts server.Options) (*mcpserver.MCPServer, error) {
		return nil, nil
	}
	serve = func(s *mcpserver.MCPServer) error {
		return expectedErr
	}
	defer restore()

	err := run()
	require.ErrorIs(t, err, expectedErr)
}

func TestRunSuccess(t *testing.T) {
	restore := stubRunDeps()
	called := false
	loadConfig = func() (*config.Config, error) {
		return &config.Config{}, nil
	}
	newClient = func(cfg *config.Config) (unifi.Client, error) {
		return nil, nil
	}
	newServer = func(opts server.Options) (*mcpserver.MCPServer, error) {
		return nil, nil
	}
	serve = func(s *mcpserver.MCPServer) error {
		called = true
		return nil
	}
	defer restore()

	err := run()
	require.NoError(t, err)
	require.True(t, called)
}

func TestMainLogsAndExitsOnError(t *testing.T) {
	expectedErr := errors.New("boom")
	restore := stubRunDeps()
	buf := &bytes.Buffer{}
	log.SetOutput(buf)
	loadConfig = func() (*config.Config, error) {
		return nil, expectedErr
	}
	exited := false
	exitCode := 0
	exit = func(code int) {
		exited = true
		exitCode = code
	}
	defer func() {
		restore()
		log.SetOutput(log.Default().Writer())
	}()

	main()
	require.True(t, exited)
	require.Equal(t, 1, exitCode)
	require.Contains(t, buf.String(), "Error: boom")
}

func stubRunDeps() func() {
	originalLoad := loadConfig
	originalNewClient := newClient
	originalNewServer := newServer
	originalServe := serve
	originalExit := exit

	return func() {
		loadConfig = originalLoad
		newClient = originalNewClient
		newServer = originalNewServer
		serve = originalServe
		exit = originalExit
	}
}

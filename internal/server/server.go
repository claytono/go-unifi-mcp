package server

import (
	"fmt"

	"github.com/claytono/go-unifi-mcp/internal/config"
	"github.com/claytono/go-unifi-mcp/internal/tools/generated"
	"github.com/filipowm/go-unifi/unifi"
	"github.com/mark3labs/mcp-go/server"
)

// Version is set at build time.
var Version = "dev"

const ServerName = "go-unifi-mcp"

// Options configures server creation.
type Options struct {
	Client unifi.Client
}

// New creates a new MCP server with all UniFi tools registered.
func New(opts Options) (*server.MCPServer, error) {
	if opts.Client == nil {
		return nil, fmt.Errorf("client is required")
	}

	s := server.NewMCPServer(
		ServerName,
		Version,
		server.WithToolCapabilities(true),
	)

	// Register all generated tools
	generated.RegisterAllTools(s, opts.Client)

	return s, nil
}

// NewClient creates a UniFi client from configuration.
func NewClient(cfg *config.Config) (unifi.Client, error) {
	clientCfg := &unifi.ClientConfig{
		URL:       cfg.Host,
		VerifySSL: cfg.VerifySSL,
	}

	if cfg.UseAPIKey() {
		clientCfg.APIKey = cfg.APIKey
	} else {
		clientCfg.User = cfg.Username
		clientCfg.Password = cfg.Password
	}

	return unifi.NewClient(clientCfg)
}

// Serve starts the MCP server on stdio.
func Serve(s *server.MCPServer) error {
	return server.ServeStdio(s)
}

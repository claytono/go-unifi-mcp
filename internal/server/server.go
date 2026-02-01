package server

import (
	"fmt"
	"os"

	"github.com/claytono/go-unifi-mcp/internal/config"
	"github.com/claytono/go-unifi-mcp/internal/meta"
	"github.com/claytono/go-unifi-mcp/internal/resolve"
	"github.com/claytono/go-unifi-mcp/internal/tools/generated"
	"github.com/claytono/go-unifi-mcp/internal/tools/registry"
	"github.com/filipowm/go-unifi/unifi"
	"github.com/mark3labs/mcp-go/server"
)

// Version is set at build time.
var Version = "dev"

var newUnifiClient = unifi.NewClient

const ServerName = "go-unifi-mcp"

// Mode determines how tools are registered with the MCP server.
type Mode string

const (
	// ModeLazy registers only 3 meta-tools (~200 tokens context).
	ModeLazy Mode = "lazy"
	// ModeEager registers all 242 direct tools (~55K tokens context).
	ModeEager Mode = "eager"
)

// Options configures server creation.
type Options struct {
	Client   unifi.Client
	Mode     Mode   // defaults to ModeLazy if empty
	LogLevel string // log level string for resolve debug logging
}

// New creates a new MCP server with UniFi tools registered.
// In lazy mode (default), only 3 meta-tools are registered for reduced context.
// In eager mode, all 242 direct tools are registered.
func New(opts Options) (*server.MCPServer, error) {
	if opts.Client == nil {
		return nil, fmt.Errorf("client is required")
	}

	// Determine mode from options, environment, or default
	mode := opts.Mode
	if mode == "" {
		mode = Mode(os.Getenv("UNIFI_TOOL_MODE"))
	}
	if mode == "" {
		mode = ModeLazy
	}

	// Build resolver for ID reference resolution
	resourceIndex := resolve.BuildResourceIndex(generated.AllToolMetadata)
	logger := resolve.NewLogger(opts.LogLevel)
	resolver := resolve.New(opts.Client, resourceIndex, logger)

	s := server.NewMCPServer(
		ServerName,
		Version,
		server.WithToolCapabilities(true),
	)

	if mode == ModeEager {
		// Register all direct tools from metadata
		if err := registry.RegisterAllTools(s, opts.Client, resolver); err != nil {
			return nil, fmt.Errorf("failed to register tools: %w", err)
		}
	} else {
		// Register 3 meta-tools for lazy mode
		meta.RegisterMetaTools(s, opts.Client, resolver)
	}

	return s, nil
}

// ParseLogLevel maps a config log level string to a unifi.LoggingLevel.
func ParseLogLevel(level string) unifi.LoggingLevel {
	switch level {
	case "disabled":
		return unifi.DisabledLevel
	case "trace":
		return unifi.TraceLevel
	case "debug":
		return unifi.DebugLevel
	case "info":
		return unifi.InfoLevel
	case "warn":
		return unifi.WarnLevel
	case "error":
		return unifi.ErrorLevel
	default:
		return unifi.ErrorLevel
	}
}

// NewClient creates a UniFi client from configuration.
func NewClient(cfg *config.Config) (unifi.Client, error) {
	clientCfg := &unifi.ClientConfig{
		URL:       cfg.Host,
		VerifySSL: cfg.VerifySSL,
		Logger:    unifi.NewDefaultLogger(ParseLogLevel(cfg.LogLevel)),
	}

	if cfg.UseAPIKey() {
		clientCfg.APIKey = cfg.APIKey
	} else {
		clientCfg.User = cfg.Username
		clientCfg.Password = cfg.Password
	}

	return newUnifiClient(clientCfg)
}

// Serve starts the MCP server on stdio.
func Serve(s *server.MCPServer) error {
	return server.ServeStdio(s)
}

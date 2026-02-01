package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/claytono/go-unifi-mcp/internal/config"
	"github.com/claytono/go-unifi-mcp/internal/server"
	"github.com/filipowm/go-unifi/unifi"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

var exit = os.Exit

func main() {
	mainWith(defaultRunner(), exit, log.Default(), os.Args, os.Stderr)
}

type runner struct {
	loadConfig func() (*config.Config, error)
	newClient  func(*config.Config) (unifi.Client, error)
	newServer  func(server.Options) (*mcpserver.MCPServer, error)
	serve      func(*mcpserver.MCPServer) error
}

func defaultRunner() runner {
	return runner{
		loadConfig: config.Load,
		newClient:  server.NewClient,
		newServer:  server.New,
		serve:      server.Serve,
	}
}

func printUsage(w io.Writer) {
	_, _ = fmt.Fprintf(w, `go-unifi-mcp - MCP server for UniFi Network Controller

Usage: go-unifi-mcp [flags]

Flags:
  -version      Print version and exit
  -help, -h     Show this help message

Environment variables:
  UNIFI_HOST        UniFi controller URL (required)
  UNIFI_API_KEY     API key (preferred auth method)
  UNIFI_USERNAME    Username for password auth
  UNIFI_PASSWORD    Password for password auth
  UNIFI_SITE        UniFi site name (default: "default")
  UNIFI_VERIFY_SSL  Verify SSL certificates (default: true)
  UNIFI_LOG_LEVEL   go-unifi log level: disabled|trace|debug|info|warn|error (default: "error")
  UNIFI_TOOL_MODE   Tool registration mode: lazy|eager (default: "lazy")
`)
}

func mainWith(r runner, exit func(int), logger *log.Logger, args []string, output io.Writer) {
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.SetOutput(output)
	versionFlag := fs.Bool("version", false, "Print version and exit")

	fs.Usage = func() { printUsage(output) }

	if err := fs.Parse(args[1:]); err != nil {
		// -help/-h triggers ErrHelp, print usage and exit 0
		if errors.Is(err, flag.ErrHelp) {
			exit(0)
			return
		}
		exit(2)
		return
	}

	if *versionFlag {
		_, _ = fmt.Fprintf(output, "go-unifi-mcp %s\n", server.Version)
		exit(0)
		return
	}

	if err := runWith(r); err != nil {
		var cfgErr *configError
		if errors.As(err, &cfgErr) {
			printUsage(output)
			_, _ = fmt.Fprintln(output)
		}
		logger.Printf("Error: %v", err)
		exit(1)
	}
}

type configError struct {
	err error
}

func (e *configError) Error() string { return e.err.Error() }
func (e *configError) Unwrap() error { return e.err }

func runWith(r runner) error {
	// Load configuration
	cfg, err := r.loadConfig()
	if err != nil {
		return &configError{err: err}
	}

	// Create UniFi client
	client, err := r.newClient(cfg)
	if err != nil {
		return err
	}

	// Create MCP server
	s, err := r.newServer(server.Options{
		Client:   client,
		LogLevel: cfg.LogLevel,
	})
	if err != nil {
		return err
	}

	// Start serving
	return r.serve(s)
}

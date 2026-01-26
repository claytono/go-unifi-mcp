package main

import (
	"log"
	"os"

	"github.com/claytono/go-unifi-mcp/internal/config"
	"github.com/claytono/go-unifi-mcp/internal/server"
	"github.com/filipowm/go-unifi/unifi"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

var exit = os.Exit

func main() {
	mainWith(defaultRunner(), exit, log.Default())
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

func mainWith(r runner, exit func(int), logger *log.Logger) {
	if err := runWith(r); err != nil {
		logger.Printf("Error: %v", err)
		exit(1)
	}
}

func runWith(r runner) error {
	// Load configuration
	cfg, err := r.loadConfig()
	if err != nil {
		return err
	}

	// Create UniFi client
	client, err := r.newClient(cfg)
	if err != nil {
		return err
	}

	// Create MCP server
	s, err := r.newServer(server.Options{
		Client: client,
	})
	if err != nil {
		return err
	}

	// Start serving
	return r.serve(s)
}

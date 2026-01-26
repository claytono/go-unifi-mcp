package main

import (
	"log"
	"os"

	"github.com/claytono/go-unifi-mcp/internal/config"
	"github.com/claytono/go-unifi-mcp/internal/server"
)

var loadConfig = config.Load
var newClient = server.NewClient
var newServer = server.New
var serve = server.Serve
var exit = os.Exit

func main() {
	if err := run(); err != nil {
		log.Printf("Error: %v", err)
		exit(1)
	}
}

func run() error {
	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Create UniFi client
	client, err := newClient(cfg)
	if err != nil {
		return err
	}

	// Create MCP server
	s, err := newServer(server.Options{
		Client: client,
	})
	if err != nil {
		return err
	}

	// Start serving
	return serve(s)
}

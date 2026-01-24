package main

import (
	"log"
	"os"

	"github.com/claytono/go-unifi-mcp/internal/config"
	"github.com/claytono/go-unifi-mcp/internal/server"
)

func main() {
	if err := run(); err != nil {
		log.Printf("Error: %v", err)
		os.Exit(1)
	}
}

func run() error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Create UniFi client
	client, err := server.NewClient(cfg)
	if err != nil {
		return err
	}

	// Create MCP server
	s, err := server.New(server.Options{
		Client: client,
	})
	if err != nil {
		return err
	}

	// Start serving
	return server.Serve(s)
}

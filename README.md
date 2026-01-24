# go-unifi-mcp

A Model Context Protocol (MCP) server for UniFi Network Controller, written in
Go.

## Overview

go-unifi-mcp provides an MCP interface to UniFi Network Controller, enabling AI
assistants and other MCP clients to interact with your UniFi infrastructure.

## Status

This project is under development. See the issue tracker for current progress.

## Development

### Prerequisites

- [Nix](https://nixos.org/download.html) with flakes enabled
- [direnv](https://direnv.net/) (optional but recommended)

### Developing

```bash
# Clone the repository
git clone https://github.com/claytono/go-unifi-mcp.git
cd go-unifi-mcp

# Enter the development environment
nix develop
# Or with direnv:
direnv allow

# Install pre-commit hooks
pre-commit install

# Run linters
task lint

# Run tests
task test

# Run tests with coverage
task coverage
```

### Available Tasks

```bash
task lint        # Run linters via pre-commit
task test        # Run tests
task coverage    # Run tests with coverage checks
task build       # Build the binary
task generate    # Run go generate
```

### Testing with mcp-cli

The development environment includes
[mcp-cli](https://github.com/philschmid/mcp-cli) for interactive testing of the
MCP server.

1. Create `.envrc.local` with your UniFi credentials (not tracked in git):

   ```bash
   export UNIFI_HOST="https://your-controller:443"
   export UNIFI_API_KEY="your-api-key"
   # Or use username/password:
   # export UNIFI_USERNAME="admin"
   # export UNIFI_PASSWORD="password"
   ```

2. Build and test:

   ```bash
   task build
   mcp-cli go-unifi-mcp/unifi_list_device '{}'
   ```

The `.mcp_servers.json` config is pre-configured to use the local binary.

## Credits

This project builds upon:

- [go-unifi](https://github.com/paultyng/go-unifi) - Go client library for UniFi
  Network Controller
- [unifi-network-mcp](https://github.com/sirkirby/unifi-network-mcp) - Python
  MCP server for UniFi that inspired this project
- [mcp-go](https://github.com/mark3labs/mcp-go) - Go SDK for Model Context
  Protocol

## License

MIT

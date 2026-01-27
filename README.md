# go-unifi-mcp

A Model Context Protocol (MCP) server for UniFi Network Controller, written in
Go.

## Overview

`go-unifi-mcp` provides an MCP interface to UniFi Network Controller, enabling
AI assistants and other MCP clients to interact with your UniFi infrastructure.

### Why this exists

I couldn’t find an MCP server that supported both v1 and v2 firewall rules and
IPv6, so I built one. This wraps the go-unifi library (which I trust from my
Terraform provider experience) and leans on its generated API surface. The
server is generated from the controller’s own API definitions, which makes it
much easier to keep tool coverage up to date as UniFi evolves.

### UniFi controller versioning

This project generates tools against the same UniFi Controller version pinned by
go-unifi. When go-unifi updates its supported controller version, we regenerate
our field definitions and tool metadata to match. We support the same controller
range; see their
[controller support range](https://github.com/filipowm/go-unifi/tree/main?tab=readme-ov-file#supported-unifi-controller-versions).

## Installation

### Binary (GitHub Releases)

Download pre-built binaries from the
[Releases page](https://github.com/claytono/go-unifi-mcp/releases). Binaries are
available for macOS and Linux (amd64/arm64).

```bash
# macOS (Apple Silicon)
curl -L https://github.com/claytono/go-unifi-mcp/releases/latest/download/go-unifi-mcp_darwin_arm64.tar.gz | tar xz
sudo mv go-unifi-mcp /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/claytono/go-unifi-mcp/releases/latest/download/go-unifi-mcp_darwin_amd64.tar.gz | tar xz
sudo mv go-unifi-mcp /usr/local/bin/

# Linux (amd64)
curl -L https://github.com/claytono/go-unifi-mcp/releases/latest/download/go-unifi-mcp_linux_amd64.tar.gz | tar xz
sudo mv go-unifi-mcp /usr/local/bin/

# Linux (arm64)
curl -L https://github.com/claytono/go-unifi-mcp/releases/latest/download/go-unifi-mcp_linux_arm64.tar.gz | tar xz
sudo mv go-unifi-mcp /usr/local/bin/
```

### Nix

```bash
# Run without installing
nix run github:claytono/go-unifi-mcp

# Install to your profile
nix profile install github:claytono/go-unifi-mcp
```

### Docker

Multi-architecture images (amd64/arm64) are published to GitHub Container
Registry.

```bash
# Latest (pinned to most recent release, rebuilt on base image updates)
docker pull ghcr.io/claytono/go-unifi-mcp:latest

# Edge (built from main on every merge, unstable)
docker pull ghcr.io/claytono/go-unifi-mcp:edge
```

### Go Install

```bash
go install github.com/claytono/go-unifi-mcp/cmd/go-unifi-mcp@latest
```

## Configuration

### UniFi Credentials

The server requires access to a UniFi Network Controller. Two authentication
methods are supported:

1. **API Key** (preferred): Create an API key in your UniFi controller under
   Settings > Control Plane > Integrations. Set `UNIFI_HOST` and
   `UNIFI_API_KEY`.

2. **Username/Password**: Use a local admin account. Set `UNIFI_HOST`,
   `UNIFI_USERNAME`, and `UNIFI_PASSWORD`.

### Claude Desktop

Add to your `claude_desktop_config.json`:

**Using the binary:**

```json
{
  "mcpServers": {
    "unifi": {
      "command": "/usr/local/bin/go-unifi-mcp",
      "env": {
        "UNIFI_HOST": "https://your-controller:443",
        "UNIFI_API_KEY": "your-api-key"
      }
    }
  }
}
```

**Using Docker:**

```json
{
  "mcpServers": {
    "unifi": {
      "command": "docker",
      "args": [
        "run",
        "-i",
        "--rm",
        "-e",
        "UNIFI_HOST",
        "-e",
        "UNIFI_API_KEY",
        "ghcr.io/claytono/go-unifi-mcp:latest"
      ],
      "env": {
        "UNIFI_HOST": "https://your-controller:443",
        "UNIFI_API_KEY": "your-api-key"
      }
    }
  }
}
```

### Claude Code

```bash
claude mcp add unifi -- go-unifi-mcp
```

Then set the required environment variables in your shell before running
`claude`.

### Environment Variables

| Variable           | Required | Default   | Description                     |
| ------------------ | -------- | --------- | ------------------------------- |
| `UNIFI_HOST`       | Yes      | —         | UniFi controller URL            |
| `UNIFI_API_KEY`    | \*       | —         | API key (preferred auth method) |
| `UNIFI_USERNAME`   | \*       | —         | Username for password auth      |
| `UNIFI_PASSWORD`   | \*       | —         | Password for password auth      |
| `UNIFI_SITE`       | No       | `default` | UniFi site name                 |
| `UNIFI_VERIFY_SSL` | No       | `true`    | Whether to verify SSL certs     |
| `UNIFI_TOOL_MODE`  | No       | `lazy`    | Tool registration mode          |

\* Either `UNIFI_API_KEY` or both `UNIFI_USERNAME` and `UNIFI_PASSWORD` must be
set.

### Tool Modes

The server supports two tool registration modes, following the pattern
established by
[unifi-network-mcp](https://github.com/sirkirby/unifi-network-mcp):

| Mode    | Tools | Context Size | Description                                     |
| ------- | ----- | ------------ | ----------------------------------------------- |
| `lazy`  | 3     | ~200 tokens  | Meta-tools only (default, recommended for LLMs) |
| `eager` | 242   | ~55K tokens  | All tools registered directly                   |

**Lazy mode** (default) registers only 3 meta-tools that provide access to 242
UniFi operations (generated from the controller API):

- `tool_index` - Search/filter the tool catalog by category or resource
- `execute` - Execute any tool by name with arguments
- `batch` - Execute multiple tools in parallel

This dramatically reduces context window usage while preserving full
functionality. The LLM first queries the index to find relevant tools, then
executes them via the dispatcher.

**Eager mode** registers all 242 tools directly, which may be useful for non-LLM
clients or debugging but consumes significant context.

**Update semantics:** Updates use a read-modify-write flow against the
controller API. We fetch the current resource, merge your fields, and submit the
full object. This avoids clearing unspecified fields, but it is not atomic and
concurrent updates can race (last write wins) because the UniFi API does not
expose etags or revision IDs. In practice this is unlikely to be an issue, but
it's something to be aware of.

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

2. Build the binary:

   ```bash
   task build
   ```

3. Test with mcp-cli:

   The `.mcp_servers.json` config provides two server entries:
   - `go-unifi-mcp` - eager mode (242 tools)
   - `go-unifi-mcp-lazy` - lazy mode (3 meta-tools)

   **Eager mode** (direct tool access):

   ```bash
   # List tools (shows all 242)
   mcp-cli go-unifi-mcp --list-tools

   # Call a tool directly
   mcp-cli go-unifi-mcp/list_device '{}'
   mcp-cli go-unifi-mcp/list_network '{"site": "default"}'
   ```

   **Lazy mode** (meta-tools):

   ```bash
   # List tools (shows only 3 meta-tools)
   mcp-cli go-unifi-mcp-lazy --list-tools

   # Query the tool index
   mcp-cli go-unifi-mcp-lazy/tool_index '{}'
   mcp-cli go-unifi-mcp-lazy/tool_index '{"category": "list"}'
   mcp-cli go-unifi-mcp-lazy/tool_index '{"resource": "network"}'

   # Execute a tool via the dispatcher
   mcp-cli go-unifi-mcp-lazy/execute '{"tool": "list_device", "arguments": {}}'

   # Batch execute multiple tools
   mcp-cli go-unifi-mcp-lazy/batch '{"calls": [{"tool": "list_network", "arguments": {}}, {"tool": "list_device", "arguments": {}}]}'
   ```

## Credits

This project builds upon:

- [go-unifi](https://github.com/paultyng/go-unifi) - Go client library for UniFi
  Network Controller
- [unifi-network-mcp](https://github.com/sirkirby/unifi-network-mcp) - Python
  MCP server for UniFi that inspired this project
- [mcp-go](https://github.com/mark3labs/mcp-go) - Go SDK for Model Context
  Protocol

## License

MPL-2.0

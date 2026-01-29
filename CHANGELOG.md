# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-01-29

### Added

- MCP server for UniFi Network Controller with 240+ auto-generated tools
- Lazy mode with 3 meta-tools (tool_index, execute, batch) for LLM-friendly
  operation
- Eager mode exposing all tools directly for MCP clients that handle large tool
  sets
- Docker multi-arch images (linux/amd64, linux/arm64) published to ghcr.io
- Homebrew tap install via `brew install claytono/tap/go-unifi-mcp`
- Nix flake support for reproducible builds and installation
- Pre-built binaries for linux and macOS (amd64 and arm64)
- Configurable site selection and authentication via environment variables

[0.1.0]: https://github.com/claytono/go-unifi-mcp/releases/tag/v0.1.0

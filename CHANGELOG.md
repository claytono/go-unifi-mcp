# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2026-02-03

### Added

- Add automatic ID reference resolution for tool responses — list, get, create,
  and update tools now resolve opaque `_id` fields to human-readable `_name`
  siblings using per-request caching. For example,
  `"network_id": "609fbf24e3ae433962e000de"` becomes
  `"network_id": "609fbf24e3ae433962e000de", "network_name": "IOT"`. Resolution
  is on by default; pass `"resolve": false` to disable (#63)
- Add configurable log level via `UNIFI_LOG_LEVEL` environment variable,
  defaulting to `error` to prevent go-unifi INFO messages from breaking piped
  JSON workflows (#62)
- Add `filter`, `fields`, and `search` post-processing parameters to all list
  operations for client-side filtering. `filter` supports exact match, contains,
  and regex operators (e.g. `"filter": {"name": {"contains": "office"}}`).
  `fields` projects the response to specific keys (e.g.
  `"fields": ["name", "ip"]`). `search` does a full-text search across all
  string values (#61)

### Changed

- Pin and update GitHub Actions dependencies

## [0.1.1] - 2026-01-30

### Added

- `--help` and `--version` CLI flags
- Automatic publishing to MCP Registry on release

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

[0.2.0]: https://github.com/claytono/go-unifi-mcp/releases/tag/v0.2.0
[0.1.1]: https://github.com/claytono/go-unifi-mcp/releases/tag/v0.1.1
[0.1.0]: https://github.com/claytono/go-unifi-mcp/releases/tag/v0.1.0

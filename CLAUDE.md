# go-unifi-mcp

MCP server for UniFi Network Controller written in Go.

## Development

This project uses Nix for development environment management. Run `nix develop`
or use direnv to enter the development shell.

### Common Commands

```bash
task lint        # Run linters
task test        # Run tests
task coverage    # Run tests with coverage checks
task build       # Build binary
task generate    # Run go generate
```

### Project Structure

- `/internal/` - Internal packages (not exported)
- `/pkg/` - Public packages (exported API)
- `/cmd/` - Main entry points

### Code Style

- Use `goimports` for import organization (local prefix:
  `github.com/claytono/go-unifi-mcp`)
- Follow standard Go conventions
- Maintain 85% total test coverage, 85% per file

### Type Generation

Types in this project are generated from the go-unifi library. Do not manually
edit generated type files. See Phase 2 documentation for sync process.

### Testing

- Write table-driven tests where applicable
- Use testify for assertions
- Mock external dependencies using mockery

## Issue Tracking

This project uses **bd** (beads) for issue tracking. Run `bd onboard` to get
started.

### Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --status in_progress  # Claim work
bd close <id>         # Complete work
bd sync               # Sync with git
```

### Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT
complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs
   follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**

- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds

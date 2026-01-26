# go-unifi-mcp

MCP server for UniFi Network Controller written in Go.

## ABSOLUTE PROHIBITIONS

**The following actions are FORBIDDEN without explicit user permission. There
are NO exceptions. Do not rationalize, justify, or work around these rules.**

### 1. DO NOT CREATE OR MERGE PULL REQUESTS WITHOUT PERMISSION

- Always ASK before creating a PR - never create one autonomously
- NEVER merge PRs - merging is the user's responsibility
- Even if tests pass, even if it looks ready, even if the user seems busy - ASK
  FIRST

### 2. DO NOT BYPASS PRE-COMMIT HOOKS

- Never use `--no-verify` on any git command
- Never disable or skip hooks "temporarily"
- If hooks fail, FIX THE ISSUE - do not bypass
- Hook failures exist to catch problems - respect them

### 3. DO NOT CHANGE CODE COVERAGE THRESHOLDS

- The coverage thresholds in `.testcoverage.yaml` are SACRED
- Never lower `total` or `file` thresholds for any reason
- If coverage is failing, write more tests - do not lower the bar
- This includes "temporary" changes - there is no such thing

### 4. DO NOT RUSH

- Rushing leads to sloppy work that wastes MORE time later
- Take the time to do things correctly the first time
- If something seems hard, that's a sign to slow down, not speed up
- Quality gates exist for a reason - do not circumvent them to "save time"

**If you find yourself tempted to violate any of these rules, STOP and ask the
user for guidance. The answer is almost certainly "no, do it correctly."**

---

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
- Maintain 95% total test coverage, 90% per file (DO NOT CHANGE THESE VALUES)

### Type Generation

Types in this project are generated from the go-unifi library. Do not manually
edit generated type files. See Phase 2 documentation for sync process.

### Testing

- Write table-driven tests where applicable
- Use testify for assertions
- Mock external dependencies using mockery

### Mocks

- Generate mocks with `go generate ./internal/server/mocks`
- Mock definitions live in `internal/server/mocks`
- Do not edit generated mocks manually

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

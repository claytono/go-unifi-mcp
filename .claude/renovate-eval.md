# Renovate Eval Context

## Role of This File

This file provides repository context, discovery hints, validation commands, and
action-menu behavior for Renovate PR evaluation. It does not redefine the shared
`renovate:safe`, `renovate:caution`, `renovate:breaking`, or `renovate:risk`
label semantics.

## Repo Layout

- Go application entry points: `cmd/`
- Internal packages: `internal/`
- Public Go packages: `pkg/`
- Generated UniFi API types and tool metadata: inspect generated-file headers
  before recommending manual edits
- Nix development environment and packaged tools: `flake.nix` and `flake.lock`
- Go module metadata: `go.mod` and `go.sum`
- Release automation: `.goreleaser.yml`, `RELEASE.md`, and release workflows
- Renovate configuration: `.github/renovate.json`

## Normal Validation Actions

- `nix develop --command task lint` runs repository linters. The CI lint job
  skips `go-test-coverage` there because coverage is checked by the test job.
- `nix develop --command task coverage` runs tests with coverage thresholds.
- `nix develop --command task build` builds the binary.
- `nix develop --command task generate` regenerates generated code when a
  dependency update changes generated UniFi API surfaces.
- Docker/release impact is validated by the CI Docker job with GoReleaser
  snapshot builds.
- Renovate PRs may trigger `.github/workflows/renovate-flake-hash-fix.yaml`,
  which fixes Nix hash mismatches in `flake.nix`.

## Config Discovery

- Go dependency updates are usually in `go.mod` and `go.sum`.
- Updates to the pinned UniFi controller surface may involve
  `.go-unifi-version`, generated files, and release notes from
  `filipowm/go-unifi`.
- Flake-managed tool updates usually touch `flake.nix` and may require hash
  refreshes.
- Docker and GitHub Actions updates are managed by workflow files and
  `.goreleaser.yml`.
- Coverage settings are controlled by `.testcoverage.yaml`; do not recommend
  lowering thresholds.

## Notes

- The repository enforces 95% total coverage and 90% per-file coverage.
- Generated types and generated mocks should not be edited manually.
- Releases publish binaries, Docker images, Homebrew tap updates, and MCP
  registry metadata; dependency changes can affect more than the Go test suite.

## Actions Menu

Include the default shared actions menu. Add a "Regenerate and test" action when
the update changes `filipowm/go-unifi`, `.go-unifi-version`, generated files, or
generation tooling, because those PRs may need `task generate` followed by the
normal lint, coverage, and build checks.

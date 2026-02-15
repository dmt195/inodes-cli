# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

```bash
make build          # Build binary to bin/inodes (version from git tags)
make test           # Run all tests with verbose output
make test-cover     # Run tests with HTML coverage report
make format         # Format all Go files with gofmt
make lint           # Check formatting + go vet
make install        # Install to $GOPATH/bin
```

Run a single test:
```bash
go test ./internal/client/ -run TestName -v
```

Always run `make format` (or `gofmt -w .`) before committing Go changes.

## Architecture

This is a CLI for the Image Nodes image processing API, built with Cobra and Go 1.26.

### Package Layout

- **`main.go`** — Root Cobra command, global flags (`--api-key`, `--base-url`), and exit code mapping from error strings (auth=2, API=3, network=4, file=5).
- **`internal/commands/`** — One file per subcommand (`configure`, `list`, `describe`, `run`, `upload`). Each follows the same pattern: load config → require API key → create client → call API → format output. `completions.go` provides shell tab-completion by fetching pipeline IDs from the API.
- **`internal/client/`** — HTTP API client. All API responses use a `{"error", "message", "data"}` JSON wrapper decoded by `decodeJSON()`, **except** the evaluate endpoint which returns `PipelineReport` directly. Auth is via `X-API-Key` header.
- **`internal/config/`** — Config resolution with priority: CLI flags > env vars (`INODES_API_KEY`, `INODES_BASE_URL`) > config file (`$XDG_CONFIG_HOME/imagenodes/config.json`). Default base URL: `https://imagenodes.com`.
- **`internal/output/`** — Terminal output formatting. `IsInteractive()` detects TTY for switching between pretty tables (lipgloss-styled) and JSON output. All commands support `--json` for machine-readable output.
- **`internal/tui/`** — Shared lipgloss styles and symbols (check/cross/arrow). Interactive prompts use charmbracelet/huh.

### Key Patterns

- The `run` command has dual modes: interactive (prompts for missing params via huh forms) and CI/CD (`--no-prompt` uses flags and defaults only).
- Image parameters can be local file paths (auto-uploaded as ephemeral assets) or existing asset UUIDs.
- Version is injected at build time via `-ldflags "-X main.version=$(VERSION)"`.
- `action.yaml` defines a GitHub Action composite step that installs the CLI and verifies auth.

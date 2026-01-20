# Implementation Progress

## Commit 1: Config + CLI surface for OpenCode backend

- Added OpenCode config fields to `internal/config/config.go`:
  - `OpenCodeServerURL` - URL for OpenCode server
  - `OpenCodeUsername` - Username for basic auth
  - `OpenCodePassword` - Password for basic auth
  - `OpenCodeModelID` - Model ID (default: glm-4.7)

- Added CLI flags and environment variable fallbacks in `cmd/ralph/main.go`:
  - `--backend` - Backend selection: cli or opencode
  - `--opencode-url` - Server URL (env: OPENCODE_SERVER_URL)
  - `--opencode-user` - Username (env: OPENCODE_SERVER_USERNAME, default: opencode)
  - `--opencode-pass` - Password (env: OPENCODE_SERVER_PASSWORD)
  - `--opencode-model` - Model ID (env: OPENCODE_MODEL_ID, default: glm-4.7)

- Default max calls set to 10 when backend is `opencode` (vs 3 for cli)

- Updated help text with new backend options section

## Commit 2: OpenCode server client + session persistence

- Created `internal/opencode` package with:
  - `client.go` - HTTP client wrapper for `/session` and `/session/:id/message` endpoints
  - `session.go` - Session persistence helpers using `.opencode_session_id` file
  - `runner.go` - Runner implementation that maps OpenCode responses to the existing output format

- Key features:
  - Basic auth headers for server authentication
  - Configurable request timeouts (default: 5 minutes)
  - Atomic file writes for session persistence
  - Event emission for TUI compatibility

- Unit tests (`client_test.go`, `session_test.go`) covering:
  - Auth header verification
  - Session creation and message sending
  - Error handling for HTTP errors
  - Session file persistence and cleanup

## Commit 3: Runner integration + TUI loop behavior

- Created `internal/runner` package with:
  - `Runner` interface for backend abstraction
  - `New()` factory function that selects backend based on config
  - `codexWrapper` and `openCodeWrapper` adapters

- Updated `internal/loop/controller.go`:
  - Now uses `runner.Runner` interface instead of concrete `codex.Runner`
  - Dynamically selects backend based on `config.Backend` setting
  - Log messages show correct backend name (Codex or OpenCode)

- Unit tests for runner package:
  - Backend selection tests (default, cli, opencode)
  - Callback setting verification

## Commit 4: Docs + examples for OpenCode usage

- Created `docs/opencode.md` with:
  - Server setup and environment variable configuration
  - Example usage with `--backend opencode`
  - Default model setting (Z.AI GLM 4.7) and override instructions
  - Quick-start examples for running TUI with 10-iteration loops
  - Configuration reference table
  - Troubleshooting guide
  - Architecture overview

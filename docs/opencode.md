# OpenCode Backend

Lisa supports the OpenCode server as an alternative AI backend, enabling integration with the Z.AI GLM 4.7 coding plan model.

## Overview

The OpenCode backend connects Lisa to an OpenCode server instance, providing:

- **HTTP-based API** - RESTful endpoints for session management and message passing
- **Z.AI GLM 4.7** - Default model optimized for coding tasks (provider: `zhipuai-coding-plan`)
- **Session persistence** - Automatic conversation continuity across loop iterations
- **10-iteration default** - Extended loop limit for more complex tasks

## Setup

### Server Requirements

You need a running OpenCode server instance. The server can be started with:

```bash
opencode serve
# or
opencode web
```

### Environment Variables

Configure the OpenCode backend using environment variables:

```bash
# Required: OpenCode server URL
export OPENCODE_SERVER_URL="http://localhost:8080"

# Required: Server password for authentication
export OPENCODE_SERVER_PASSWORD="your-password-here"

# Optional: Username (defaults to "opencode")
export OPENCODE_SERVER_USERNAME="opencode"

# Optional: Model ID (defaults to "glm-4.7")
export OPENCODE_MODEL_ID="glm-4.7"
```

Add to your shell profile for persistence:

```bash
# For bash
cat >> ~/.bashrc << 'EOF'
export OPENCODE_SERVER_URL="http://localhost:8080"
export OPENCODE_SERVER_PASSWORD="your-password-here"
EOF
source ~/.bashrc

# For zsh
cat >> ~/.zshrc << 'EOF'
export OPENCODE_SERVER_URL="http://localhost:8080"
export OPENCODE_SERVER_PASSWORD="your-password-here"
EOF
source ~/.zshrc
```

## Usage

### Basic Usage

Run Lisa with the OpenCode backend:

```bash
lisa run --backend opencode --monitor
```

### With All Options

Override environment variables via command-line flags:

```bash
lisa run \
  --backend opencode \
  --opencode-url http://localhost:8080 \
  --opencode-user opencode \
  --opencode-pass your-password \
  --opencode-model glm-4.7 \
  --calls 10 \
  --monitor
```

### Quick Start Examples

**10-iteration loop with TUI monitoring:**
```bash
export OPENCODE_SERVER_URL="http://localhost:8080"
export OPENCODE_SERVER_PASSWORD="secret"
lisa run --backend opencode --monitor
```

**Initialize and run with OpenCode:**
```bash
lisa init --backend opencode
```

**Override max calls (default is 10 for opencode):**
```bash
lisa run --backend opencode --calls 5 --monitor
```

## Configuration Reference

### CLI Flags

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--backend` | - | `cli` | Backend selection: `cli` or `opencode` |
| `--opencode-url` | `OPENCODE_SERVER_URL` | - | OpenCode server URL |
| `--opencode-user` | `OPENCODE_SERVER_USERNAME` | `opencode` | Basic auth username |
| `--opencode-pass` | `OPENCODE_SERVER_PASSWORD` | - | Basic auth password |
| `--opencode-model` | `OPENCODE_MODEL_ID` | `glm-4.7` | Model ID to use |
| `--calls` | - | `10` (opencode) | Max loop iterations |

### Session Persistence

The OpenCode backend persists session IDs in `.opencode_session_id` for conversation continuity. This file is automatically managed:

- Created when a new session starts
- Updated after each successful interaction
- Used to resume conversations across Lisa restarts

To start a fresh session, delete the file:

```bash
rm .opencode_session_id
```

## Model Configuration

### Default Model: Z.AI GLM 4.7

The default model (`glm-4.7`) is the Z.AI Coding Plan model, optimized for:

- Code generation and completion
- Implementation planning
- Refactoring tasks
- Bug fixing

### Alternative Models

To use a different model supported by your OpenCode server:

```bash
lisa run --backend opencode --opencode-model your-model-id
```

Or via environment variable:

```bash
export OPENCODE_MODEL_ID="your-model-id"
lisa run --backend opencode
```

## Troubleshooting

### Connection Errors

If you see "failed to create session" errors:

1. Verify the OpenCode server is running
2. Check the server URL is correct
3. Ensure the password is set correctly

```bash
# Test connection
curl -u opencode:your-password http://localhost:8080/health
```

### Authentication Errors

If you see 401 Unauthorized errors:

1. Check `OPENCODE_SERVER_PASSWORD` is set
2. Verify the username (default: `opencode`)
3. Ensure the server is configured with matching credentials

### Session Issues

If conversations aren't continuing:

1. Check `.opencode_session_id` exists and contains a valid ID
2. Verify the session hasn't expired on the server
3. Try clearing the session file to start fresh

## Architecture

The OpenCode integration consists of:

- `internal/opencode/client.go` - HTTP client for API endpoints
- `internal/opencode/session.go` - Session persistence helpers
- `internal/opencode/runner.go` - Runner implementation
- `internal/runner/runner.go` - Backend abstraction layer

The runner interface allows seamless switching between backends without changes to the loop controller.

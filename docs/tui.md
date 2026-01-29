# Lisa Codex TUI Documentation

## Overview

Lisa Codex includes a terminal user interface (TUI) powered by Charm libraries (Bubble Tea, Lip Gloss). The TUI provides real-time monitoring of the autonomous development loop with a modern, visually appealing interface.

## Starting the TUI

To start Lisa with the integrated TUI monitor:

```bash
lisa --command run --monitor
```

## Keybindings

### Navigation
- `q` / `Ctrl+C` - Quit Lisa Codex
- `?` - Toggle help screen

### Loop Control
- `r` - Run / Restart loop
- `p` - Pause / Resume loop

### Views
- `l` - Toggle log view
- `c` - Show circuit breaker status
- `R` - Reset circuit breaker

## Views

### Status View (Default)

The main status view displays:
- **Loop Counter**: Current iteration number
- **Spinner**: Animated indicator when Codex is running
- **Rate Limit Progress**: Visual progress bar showing API calls used vs. remaining
- **Circuit State**: Current circuit breaker state (CLOSED/HALF_OPEN/OPEN)
- **Codex Status**: Current execution status
- **Elapsed Time**: Time since loop started

### Circuit Breaker View

Press `c` to view circuit breaker status:
- **CLOSED** - Circuit is operational, normal execution allowed
- **HALF_OPEN** - Circuit is monitoring, may pause if no progress continues
- **OPEN** - Circuit is open! Execution halted due to repeated failures

Press `R` in this view to reset the circuit breaker to CLOSED state.

### Log View

Press `l` to toggle the log view:
- Shows the last 30 log entries (scrollable)
- Color-coded by log level (INFO, WARN, ERROR, SUCCESS)
- Press `l` again to return to status view

### Help View

Press `?` to view comprehensive help:
- Navigation keybindings
- Loop control commands
- View toggles
- CLI options
- Project options
- Rate limiting settings
- Project commands
- Troubleshooting tips

## CLI Options

### Backend Selection
```bash
--backend cli    # Use Claude Code CLI backend (default)
--backend sdk    # Use Claude Code SDK backend
```

### Monitoring
```bash
--monitor        # Enable integrated TUI monitoring
--verbose        # Enable verbose output
```

### Project Configuration
```bash
--project <path>  # Set project directory (default: .)
--prompt <file>   # Set prompt file (default: PROMPT.md)
```

### Rate Limiting
```bash
--calls <num>    # Max API calls per hour (default: 100)
--timeout <sec>   # Codex execution timeout (default: 600)
```

## Circuit Breaker States

### CLOSED (Green)
- Normal operation
- All loop iterations execute
- No stagnation detected

### HALF_OPEN (Yellow)
- Monitoring mode
- Some no-progress events detected
- Will transition to OPEN if stagnation continues
- Will transition back to CLOSED if progress resumes

### OPEN (Red)
- Loop execution halted
- Too many consecutive errors or no-progress events
- Press `R` to reset circuit breaker
- Check logs for details on what caused the halt

## Troubleshooting

### Loop Not Progressing

1. Check circuit breaker state with `c` key
2. If OPEN, reset with `R` key
3. View logs with `l` key for error details
4. Check that Codex is installed: `claude --version`

### Rate Limit Reached

The progress bar shows API calls used (█) vs. remaining (░):
```
Calls: 85/100 [██████████████████████████████░░░░░]
```

When full, wait for the hour to reset or increase limit:
```bash
lisa --command run --calls 200
```

### Codex Not Found

If you see errors about Codex not being found:
```bash
# Install Claude Code CLI
npm install -g @anthropic/claude-code

# Verify installation
claude --version
```

## Color Scheme

The TUI uses the following color scheme:
- **Purple** (#7D56F4) - Primary elements, headers
- **Blue** (#3B82F6) - Secondary elements
- **Green** (#10B981) - Success states, running loops
- **Amber** (#F59E0B) - Warnings, paused state
- **Red** (#EF4444) - Error states, circuit open
- **Dark Gray** (#1F2937) - Backgrounds
- **Light Gray** (#F9FAFB) - Primary text

## Building from Source

```bash
# Clone repository
git clone https://github.com/brainwhocodes/lisa-loop.git
cd lisa-loop

# Build binary
make build

# Or install to GOPATH
make install

# Run with TUI
./lisa --command run --monitor
```

## Testing

Run tests with Makefile:
```bash
# All tests
make test

# Verbose output
make test-verbose

# Coverage report
make test-coverage

# Linting
make lint
```

## Next Steps

- See [README.md](../README.md) for general usage
- See [docs/codex.md](codex.md) for Codex integration
- See [CLAUDE.md](../CLAUDE.md) for development guidelines

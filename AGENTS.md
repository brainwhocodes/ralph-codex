# AGENTS.md

This file provides guidance to AI coding agents (Codex, Claude Code, etc.) when working with code in this repository.

## Repository Overview

Ralph Codex is an autonomous AI development loop system written in **Go**. It enables continuous development cycles with intelligent exit detection, circuit breaker patterns, and rate limiting. Ralph executes Codex repeatedly in a managed loop, analyzing responses to determine when work is complete.

## Technology Stack

- **Language**: Go 1.21+
- **TUI Framework**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) (terminal UI)
- **CLI**: Native Go with goreleaser for cross-platform builds
- **AI Backend**: OpenAI Codex CLI (`codex exec --json`)
- **Build**: `go build` / `make`
- **Test**: `go test ./...`

## Project Structure

```
ralph-codex/
├── cmd/ralph/main.go          # CLI entry point
├── internal/
│   ├── analysis/              # Response analysis
│   │   └── response.go        # Parse RALPH_STATUS, detect completion
│   ├── circuit/               # Circuit breaker pattern
│   │   └── breaker.go         # CLOSED/HALF_OPEN/OPEN states
│   ├── codex/                 # Codex CLI execution
│   │   ├── config.go          # Configuration structs
│   │   └── runner.go          # Execute codex, parse JSONL output
│   ├── loop/                  # Main loop controller
│   │   ├── context.go         # Build loop context, load fix plan
│   │   ├── controller.go      # Orchestrate loop iterations
│   │   └── ratelimit.go       # API call rate limiting
│   ├── project/               # Project management
│   │   ├── import.go          # PRD/spec import to Ralph format
│   │   └── setup.go           # Create new Ralph projects
│   ├── state/                 # State persistence
│   │   └── files.go           # Atomic file I/O for state
│   └── tui/                   # Terminal UI
│       ├── model.go           # Bubble Tea model
│       ├── keybindings.go     # Keyboard shortcuts
│       ├── styles.go          # Lipgloss styling
│       └── views/             # Status, logs, help views
├── templates/                 # Project templates
│   ├── PROMPT.md
│   ├── fix_plan.md
│   └── AGENT.md
├── Makefile                   # Build automation
└── go.mod                     # Go module definition
```

## Key Commands

### Building
```bash
# Build the binary
go build -o ralph ./cmd/ralph

# Or use make
make build

# Run tests
go test ./...
make test

# Run with race detector
go test -race ./...
```

### Running Ralph
```bash
# Start the loop (from a Ralph-managed project directory)
./ralph

# With TUI monitoring
./ralph --monitor

# Custom rate limit
./ralph --calls 50

# Check status
./ralph --status

# Reset circuit breaker
./ralph --reset-circuit
```

### Creating a New Project
```bash
# Creates project structure with PROMPT.md, @fix_plan.md, @AGENT.md
./ralph setup my-project
cd my-project
```

### Importing a PRD
```bash
# Convert a PRD/spec document to Ralph format
./ralph import spec.md --name my-project
```

## Architecture Overview

### Loop Controller (`internal/loop/controller.go`)

The main orchestration loop:

```go
func (c *Controller) Run(ctx context.Context) error {
    for {
        // 1. Check rate limit
        if !c.rateLimiter.CanMakeCall() {
            return c.rateLimiter.WaitForReset(ctx)
        }
        
        // 2. Check circuit breaker
        if c.breaker.ShouldHalt() {
            return errors.New("circuit breaker OPEN")
        }
        
        // 3. Load prompt and build context
        prompt, _ := GetPrompt()
        ctx := BuildContext(loopNum, tasks, circuitState, prevOutput)
        
        // 4. Execute Codex
        output, threadID, _ := c.codexRunner.Run(prompt + ctx)
        
        // 5. Analyze response for exit conditions
        if c.ShouldContinue() {
            return nil  // All tasks complete
        }
        
        // 6. Record result in circuit breaker
        c.breaker.RecordResult(loopNum, filesChanged, hasErrors)
    }
}
```

### Circuit Breaker (`internal/circuit/breaker.go`)

Prevents runaway loops with three states:

| State | Description | Behavior |
|-------|-------------|----------|
| `CLOSED` | Normal operation | Loop executes normally |
| `HALF_OPEN` | Monitoring | Warning state, may transition to OPEN |
| `OPEN` | Halted | Loop stops, requires manual reset |

Triggers:
- **No progress**: 3+ loops with no file changes
- **Repeated errors**: 5+ loops with same error

### Response Analysis (`internal/analysis/response.go`)

Analyzes Codex output for completion signals:

```go
type RALPHStatus struct {
    Status         string  // WORKING, COMPLETE, BLOCKED
    TasksCompleted int
    FilesModified  int
    TestsStatus    string  // PASSING, FAILING, UNKNOWN
    WorkType       string  // feature, bugfix, test, docs
    ExitSignal     bool    // Explicit signal to exit loop
}
```

Exit detection uses dual verification:
1. `CompletionIndicators >= 2` (keyword detection)
2. `ExitSignal == true` (explicit confirmation)

### Codex Runner (`internal/codex/runner.go`)

Executes Codex CLI and parses JSONL output:

```go
// Build command
cmd := exec.Command("codex", "exec", "--json", "--skip-git-repo-check")
if sessionID != "" {
    cmd.Args = append(cmd.Args, "--resume", "--thread-id", sessionID)
}
cmd.Stdin = strings.NewReader(prompt)

// Parse JSONL stream for thread.started events and messages
threadID, message, events := ParseJSONLStream(output)
```

### State Persistence (`internal/state/files.go`)

All state is persisted to dot files with atomic writes:

| File | Purpose |
|------|---------|
| `.call_count` | API calls made this hour |
| `.last_reset` | Last rate limit reset time |
| `.codex_session_id` | Codex thread ID for continuity |
| `.ralph_session` | Ralph session metadata |
| `.circuit_breaker_state` | Circuit breaker state |
| `.exit_signals` | Recent exit signal history |

### TUI (`internal/tui/`)

Built with Bubble Tea for interactive monitoring:

- **Status view**: Loop number, rate limit progress, circuit state
- **Logs view**: Scrollable log history
- **Circuit view**: Detailed circuit breaker status
- **Keybindings**: `r` run, `p` pause, `l` logs, `?` help, `q` quit

## Ralph-Managed Project Files

Projects managed by Ralph use these control files:

| File | Purpose |
|------|---------|
| `PROMPT.md` | Main instructions for each loop iteration |
| `@fix_plan.md` | Prioritized task checklist (`- [ ]` / `- [x]`) |
| `@AGENT.md` | Build and run instructions |

The `@` prefix indicates Ralph-specific control files.

## Exit Conditions

Ralph exits when ANY of these conditions are met:

1. **All tasks complete**: Every item in `@fix_plan.md` is marked `[x]`
2. **Explicit exit signal**: `ExitSignal: true` in RALPH_STATUS block
3. **Circuit breaker open**: Too many no-progress or error loops
4. **Rate limit exhausted**: Max calls per hour reached (waits for reset)
5. **Completion indicators**: 2+ completion keywords AND explicit exit signal

## Configuration

### Rate Limiting
- Default: 100 calls/hour
- Configurable via `--calls` flag
- Auto-resets hourly

### Circuit Breaker Thresholds
- No progress threshold: 3 loops
- Same error threshold: 5 loops
- Half-open transitions to open after 2x threshold

## Development Guidelines

### Adding New Features

1. Write tests first (`*_test.go` files)
2. Implement in appropriate `internal/` package
3. Update CLI in `cmd/ralph/main.go` if needed
4. Run `go test ./...` to verify
5. Update this file if architecture changes

### Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Keep packages focused and small
- Prefer composition over inheritance
- Handle errors explicitly

### Testing

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package
go test ./internal/circuit/...

# Run with coverage
go test -cover ./...
```

## Integration Points

Ralph integrates with:
- **Codex CLI**: `codex exec --json` for AI execution
- **Git**: Expects projects to be git repositories
- **File system**: State files in project root

## Troubleshooting

### Circuit Breaker Stuck Open
```bash
./ralph --reset-circuit
```

### Session Issues
Delete `.codex_session_id` and `.ralph_session` to start fresh.

### Rate Limit Issues
Wait for hourly reset, or check `.call_count` and `.last_reset` files.

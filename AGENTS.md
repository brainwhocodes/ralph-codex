# AGENTS.md

This file provides guidance to AI coding agents (Codex, Claude Code, etc.) when working with code in this repository.

## Repository Overview

Lisa Codex is an autonomous AI development loop system written in **Go**. It enables continuous development cycles with intelligent exit detection, circuit breaker patterns, and loop management. Lisa executes Codex repeatedly in a managed loop, analyzing task completion to determine when work is complete.

## Technology Stack

- **Language**: Go 1.21+
- **TUI Framework**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) (terminal UI)
- **Styling**: [Lipgloss](https://github.com/charmbracelet/lipgloss)
- **AI Backend**: Codex CLI (`codex exec --json`)
- **Build**: `go build` / `make`
- **Test**: `go test ./...`

## Project Structure

```
lisa-loop/
├── cmd/
│   ├── ralph/main.go          # Main CLI entry point
│   └── test-loop/main.go      # Loop controller test harness (no TUI)
├── internal/
│   ├── analysis/              # Response analysis
│   │   └── response.go        # Parse RALPH_STATUS, detect completion
│   ├── circuit/               # Circuit breaker pattern
│   │   └── breaker.go         # CLOSED/HALF_OPEN/OPEN states
│   ├── codex/                 # Codex CLI execution + JSONL parsing
│   │   ├── config.go          # Config type alias to unified config
│   │   ├── events.go          # Unified event parsing
│   │   └── runner.go          # Execute codex, parse JSONL output
│   ├── config/                # Unified configuration
│   │   └── config.go          # Shared Config struct
│   ├── loop/                  # Main loop controller
│   │   ├── context.go         # Build loop context, load plan
│   │   ├── controller.go      # Orchestrate loop iterations
│   │   ├── events.go          # Typed constants for event types, log levels
│   │   └── ratelimit.go       # Loop iteration management
│   ├── project/               # Project management
│   │   ├── codexhelper.go     # Shared Codex invocation utilities
│   │   ├── import.go          # PRD/spec import to Lisa format
│   │   ├── init.go            # Generate plans and AGENTS.md
│   │   ├── mode.go            # Unified project mode detection
│   │   ├── prompts.go         # Prompt/template resolution
│   │   └── setup.go           # Create new Lisa projects
│   ├── state/                 # State persistence
│   │   └── files.go           # Generic LoadState[T]/SaveState[T] helpers
│   ├── stats/                 # Statistics interface
│   │   └── stats.go           # StatsProvider interface
│   └── tui/                   # Terminal UI
│       ├── model.go           # Bubble Tea model with integrated views
│       ├── views.go           # Rendering helpers
│       ├── messages.go        # Message types
│       ├── program.go         # Program wrapper
│       ├── keybindings.go     # Keyboard shortcuts
│       ├── styles.go          # Lipgloss styling, constants, helpers
│       └── theme.go           # Theme tokens
├── templates/                 # Project templates
│   ├── PROMPT.md
│   ├── fix_plan.md
│   └── AGENT.md
├── tests/                     # Bats unit/integration tests
├── Makefile                   # Build automation
└── go.mod                     # Go module definition
```

## Key Commands

### Building
```bash
# Build the main CLI
go build -o ralph ./cmd/ralph

# Or use make
make build

# Build the test harness (for loop testing without TUI)
go build -o ralph-test ./cmd/test-loop

# Run tests
go test ./...
make test

# Run with race detector
go test -race ./...
```

### Running Lisa
```bash
# Start the loop (from a Lisa-managed project directory)
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
# Convert a PRD/spec document to Lisa format
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
cmd := exec.Command("codex", "exec", "--json", "--skip-git-repo-check", "--sandbox", "danger-full-access")
if hasSession {
    cmd.Args = append(cmd.Args, "resume", "--last")
}
cmd.Stdin = strings.NewReader(prompt)

// Parse JSONL stream for events and messages
events := ParseJSONLStream(output)
```

### State Persistence (`internal/state/files.go`)

Uses generic helpers for type-safe state management:

```go
// Generic load/save helpers
func LoadState[T any](filename string, defaultVal T) (T, error)
func SaveState[T any](filename string, value T) error
```

State files:

| File | Purpose |
|------|---------|
| `.call_count` | Loop iterations completed |
| `.last_reset` | Last reset time |
| `.codex_session_id` | Codex session for continuity |
| `.ralph_session` | Lisa session metadata |
| `.circuit_breaker_state` | Circuit breaker state |
| `.exit_signals` | Recent exit signal history |

### TUI (`internal/tui/`)

Built with Bubble Tea for interactive monitoring:

- **Integrated status view**: Loop number, progress bar, circuit state, task checklist, and logs
- **Circuit view**: Detailed circuit breaker status
- **Keybindings**: `r` run, `p` pause, `l` logs, `c` circuit, `?` help, `q` quit
- **Styling**: Centralized in `styles.go` with constants for spinners, log levels, colors

## Lisa-Managed Project Files

Projects managed by Lisa use these control files:

| File | Purpose |
|------|---------|
| `PRD.md` | Requirements source for implementation mode |
| `IMPLEMENTATION_PLAN.md` | Implementation task plan (`- [ ]` / `- [x]`) |
| `REFACTOR.md` | Refactoring goals for refactor mode |
| `REFACTOR_PLAN.md` | Refactor task plan (`- [ ]` / `- [x]`) |
| `PROMPT.md` | Main instructions for fix mode |
| `@fix_plan.md` | Prioritized task checklist (`- [ ]` / `- [x]`) |
| `@AGENT.md` | Build and run instructions |

The `@` prefix indicates Lisa-specific control files.

## Exit Conditions

Lisa exits when ANY of these conditions are met:

1. **All tasks complete**: Every item in `@fix_plan.md` is marked `[x]`
2. **Explicit exit signal**: `ExitSignal: true` in RALPH_STATUS block
3. **Circuit breaker open**: Too many no-progress or error loops
4. **Rate limit exhausted**: Max calls per hour reached (waits for reset)
5. **Completion indicators**: 2+ completion keywords AND explicit exit signal

## Configuration

### Loop Limits
- Default: 3 iterations
- Configurable via `--calls` flag

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

Lisa integrates with:
- **Codex CLI**: `codex exec --json` for AI execution
- **Git**: Expects projects to be git repositories
- **File system**: State files in project root (dot files)

## Troubleshooting

### Circuit Breaker Stuck Open
```bash
./ralph --reset-circuit
```

### Session Issues
Delete `.codex_session_id` and `.ralph_session` to start fresh.

### Rate Limit Issues
Wait for hourly reset, or check `.call_count` and `.last_reset` files.

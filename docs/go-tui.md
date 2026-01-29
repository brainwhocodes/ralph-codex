# Lisa Codex Go TUI

This directory contains the Go implementation of Lisa using Charm TUI libraries.

## Structure

```
cmd/ralph/       - CLI entry point
internal/tui/     - Bubble Tea TUI implementation
internal/codex/   - Codex runner and JSONL parsing
```

## Building

```bash
go build ./cmd/ralph
```

## Running

```bash
./ralph
```

## Keybindings

- `q` / `Ctrl+c` - Quit
- `r` - Run/restart loop
- `p` - Pause
- `l` - Toggle log view
- `?` - Help

## Dependencies

- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Styling
- `github.com/charmbracelet/bubbles` - Components (planned)

## Status

This is a work-in-progress TUI implementation. The shell scripts (`ralph_loop.sh`, `setup.sh`, etc.) are still available as backup.

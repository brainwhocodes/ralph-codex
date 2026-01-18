# Ralph Codex Refactoring Plan

## Overview

This document outlines improvements identified in the codebase review. Issues are prioritized by impact and complexity.

---

## 1. Files to Remove (Unused)

| File | Reason |
|------|--------|
| `internal/tui/views/logs.go` | `LogsViewModel` never instantiated; functionality duplicated in `model.go` |
| `internal/tui/views/help.go` | `HelpViewModel` unused; help rendering done in `keybindings.go` |
| `internal/tui/views/status.go` | `StatusViewModel` unused; status rendering done in `model.go` |
| `internal/tui/views/status_test.go` | Tests unused view model |

**Action:** Delete `internal/tui/views/` directory entirely after confirming no imports.

---

## 2. Unused Imports to Remove

### `internal/tui/model.go`
```go
// REMOVE these imports (lines 10-11):
"github.com/charmbracelet/bubbles/spinner"
"github.com/charmbracelet/bubbles/viewport"
```

### `internal/tui/program.go`
```go
// REMOVE these imports:
"github.com/charmbracelet/bubbles/spinner"
"github.com/charmbracelet/bubbles/viewport"
```

**Reason:** Custom spinner frames are used instead. Viewport is declared but minimally used.

---

## 3. Dead Code to Remove

### `internal/loop/controller.go`

| Method | Lines | Reason |
|--------|-------|--------|
| `CheckExitConditions()` | 329-333 | Always returns false, never called |
| `HandleCircuitBreakerOpen()` | 335-338 | Never called |
| `HandleRateLimitExceeded()` | 340-343 | Never called |
| `UpdateProgress()` | 345-350 | Never called, prints to stdout |

### `internal/tui/model.go`

| Field | Line | Reason |
|-------|------|--------|
| `logViewport viewport.Model` | 106 | Only resized, never rendered |
| `taskSpinner spinner.Model` | 107 | Declared but custom frames used |

---

## 4. Duplicate Code to Consolidate

### 4.1 Styled Log Entry Functions

**Current State:**
- `internal/tui/model.go:119` - `styledLogEntry()` (private)
- `internal/tui/views/logs.go:130` - `StyledLogEntry()` (public)

**Solution:** Keep one in `styles.go` and export it:
```go
// internal/tui/styles.go
func StyledLogEntry(level, message string) string {
    switch level {
    case "INFO":
        return StyleHelpDesc.Render("ℹ️  " + message)
    case "WARN":
        return StyleCircuitHalfOpen.Render("⚠️  " + message)
    case "ERROR":
        return StyleErrorMsg.Render("❌ " + message)
    case "SUCCESS":
        return StyleCircuitClosed.Render("✅ " + message)
    default:
        return StyleHelpDesc.Render("   " + message)
    }
}
```

### 4.2 Spinner Frames

**Current State:** Defined in 3 places:
- `internal/tui/model.go:445`
- `internal/tui/model.go:551`
- `internal/tui/views/status.go:10`

**Solution:** Define once in `styles.go`:
```go
// internal/tui/styles.go
var SpinnerFrames = []string{">", ">>", ">>>", ">>", ">"}
var BrailleSpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
```

### 4.3 Config Structs

**Current State:**
- `internal/loop/Config`
- `internal/codex/Config`

Both have nearly identical fields.

**Solution:** Use single config in `internal/config/config.go`:
```go
package config

type Config struct {
    Backend      string
    ProjectPath  string
    PromptPath   string
    MaxCalls     int
    Timeout      int
    Verbose      bool
    ResetCircuit bool
}
```

---

## 5. Logic Simplification

### 5.1 State File Operations

**Current State:** Repeated pattern in `internal/state/files.go`:
```go
func LoadX() (Type, error) {
    data, err := ReadStateFile(filename)
    if err != nil {
        if os.IsNotExist(err) {
            return defaultValue, nil
        }
        return zero, err
    }
    var result Type
    json.Unmarshal(data, &result)
    return result, nil
}
```

**Solution:** Create generic helper:
```go
func LoadState[T any](filename string, defaultVal T) (T, error) {
    data, err := ReadStateFile(filename)
    if err != nil {
        if os.IsNotExist(err) {
            return defaultVal, nil
        }
        return defaultVal, err
    }
    var result T
    if err := json.Unmarshal(data, &result); err != nil {
        return defaultVal, err
    }
    return result, nil
}
```

### 5.2 Task Section Rendering

**Current State:** `model.go:561-591` has 3 nearly identical branches for task rendering.

**Solution:** Extract to helper:
```go
func (m Model) renderTaskLine(task Task, index int, isActive bool) string {
    indicator := "  "
    checkbox := "[ ]"
    style := StyleHelpDesc

    if task.Completed {
        checkbox = "[x]"
        style = StyleCircuitClosed
    } else if isActive && m.state == StateRunning {
        indicator = SpinnerFrames[m.tick%len(SpinnerFrames)] + " "
        style = StyleInfoMsg
    }

    return style.Render(fmt.Sprintf("%s%s %s", indicator, checkbox, task.Text))
}
```

---

## 6. Inconsistencies to Fix

### 6.1 State String Representation

**Current:** Mixed case - "CLOSED" vs "Running"

**Solution:** Use constants:
```go
// internal/tui/model.go
const (
    StateStringInit     = "INIT"
    StateStringRunning  = "RUNNING"
    StateStringPaused   = "PAUSED"
    StateStringComplete = "COMPLETE"
    StateStringError    = "ERROR"
)
```

### 6.2 Log Level Constants

**Current:** Hardcoded strings throughout

**Solution:** Define constants:
```go
// internal/tui/styles.go
const (
    LogLevelInfo    = "INFO"
    LogLevelWarn    = "WARN"
    LogLevelError   = "ERROR"
    LogLevelSuccess = "SUCCESS"
)
```

### 6.3 Error Wrapping Format

**Current:** Inconsistent error messages

**Solution:** Use consistent format:
```go
fmt.Errorf("operation failed: %w", err)
```

---

## 7. Missing Abstractions

### 7.1 Stats Interface

**Current:** Three components define similar `GetStats()` methods.

**Solution:**
```go
// internal/stats/stats.go
type StatsProvider interface {
    GetStats() map[string]interface{}
}
```

### 7.2 Event Callback Interface

**Current:** Both `Runner` and `Controller` define callback types.

**Solution:**
```go
// internal/events/events.go
type Event struct {
    Type    string
    Level   string
    Message string
    Data    map[string]interface{}
}

type EventHandler func(Event)
```

---

## 8. Implementation Priority

### Phase 1: Quick Wins (Low Risk)
1. [ ] Remove unused `internal/tui/views/` directory
2. [ ] Remove unused imports from `model.go` and `program.go`
3. [ ] Remove dead methods from `controller.go`
4. [ ] Remove unused fields from `Model` struct

### Phase 2: Consolidation (Medium Risk)
5. [ ] Move `styledLogEntry` to `styles.go`
6. [ ] Define spinner frames as constants
7. [ ] Define log level constants
8. [ ] Fix state string consistency

### Phase 3: Refactoring (Higher Risk)
9. [ ] Unify Config structs
10. [ ] Create generic state file helpers
11. [ ] Extract task rendering helper
12. [ ] Create Stats interface

---

## 9. Verification Steps

After each phase:
1. Run `go build ./...` to verify compilation
2. Run `go test ./...` to verify tests pass
3. Run `ralph --monitor` in test project to verify TUI works
4. Run `./test-loop` to verify headless mode works

---

## 10. Files Changed Summary

| Action | Files |
|--------|-------|
| DELETE | `internal/tui/views/*.go` (4 files) |
| MODIFY | `internal/tui/model.go` |
| MODIFY | `internal/tui/program.go` |
| MODIFY | `internal/tui/styles.go` |
| MODIFY | `internal/loop/controller.go` |
| CREATE | `internal/config/config.go` (optional) |

---

*Generated: 2026-01-18*

# Refactor Plan

## Overview
Reduce duplication in Codex invocation, improve loop completion reliability across all modes, align OpenCode API usage with the current server spec, and tighten parsing/error handling in `internal/project` without changing user-facing behavior or breaking existing tests.

## Current State Analysis
- Codex execution is duplicated between `internal/project/init.go:generateWithCodex` and `internal/project/setup.go:generateTemplatesWithCodex`, with different prompt delivery and output parsing.
- `internal/project/setup.go:executeCommand` is a stub, so `initGitRepo` reports success without running git commands.
- Template resolution in `internal/project/prompts.go:loadPromptTemplate` relies on global state (`TemplateDir`) and implicit search order, which makes tests and overrides brittle.
- `internal/project/import.go:parseSourceContent` resets section builders on headings, which can drop earlier content when multiple headings appear.
- Error messages are inconsistent and often omit context (which file/action failed), making failures harder to diagnose.
- Loop iteration always executes before checking completion, so already-complete plans still trigger a backend call.
- `RateLimiter.RecordCall` is only invoked on error, so successful calls do not count toward the limit.
- Task parsing only recognizes lines starting with `- [ ]`, which can silently yield zero tasks for other checklist formats.
- RALPH_STATUS expectations differ between prompts, context injection, and analysis parsing, leading to inconsistent completion signals.
- OpenCode client uses `/event` SSE and `model_id` payload fields, while the server spec documents `/global/event` and message bodies with `model` and other fields.

## Phase 1: Low-Risk Foundation
### Goals
- Make command execution real and observable.
- Add tests around the most fragile helpers before refactoring behavior.

### Tasks
- [ ] Task 1: Implement `internal/project/setup.go:executeCommand` using `os/exec`, returning errors that include the command and exit status.
  - Acceptance: failing commands surface exit status and command string; success executes the command.
  - Tests: unit test in `internal/project/setup_test.go` using a mock `CommandRunner`.
- [ ] Task 2: Add a `CommandRunner` interface in `internal/project/runner.go` and inject it into `internal/project/setup.go:initGitRepo` for test doubles.
  - Acceptance: `initGitRepo` uses injected runner; default path still uses real exec.
  - Tests: unit test validating runner invocation and error propagation.
- [ ] Task 3: Add unit tests for `internal/project/init.go:readSpecsFolder` covering empty folders, non-markdown files, and multi-file ordering.
  - Acceptance: non-markdown files ignored; multiple files concatenated deterministically.
  - Tests: table-driven tests with temp dirs and mixed file types.
- [ ] Task 4: Add unit tests for `internal/project/prompts.go:loadPromptTemplate` using temp dirs to verify custom, home, and default resolution order.
  - Acceptance: resolution order is deterministic and matches docs.
  - Tests: tests create temp dirs for each tier and assert content selection.

### Verification
- [ ] All tests pass (`go test ./...`)
- [ ] No functionality changes in generated files for existing inputs
- [ ] Code review completed

## Phase 2: Loop and Completion Reliability
### Goals
- Reduce unnecessary backend calls.
- Make task detection and completion signals consistent across modes.

### Tasks
- [ ] Task 1: Add a preflight completion check in `internal/loop/controller.go` to short-circuit before `ExecuteLoop` when the plan is already complete, the breaker is open, or rate limit is exhausted.
  - Acceptance: no backend call occurs when any preflight stop condition is met.
  - Tests: controller test verifies no runner invocation for complete plan and breaker-open cases.
- [ ] Task 2: Record rate limit usage for successful runs (`RateLimiter.RecordCall`) and centralize call accounting in `internal/loop/controller.go`.
  - Acceptance: successful runs increment call count; errors still increment.
  - Tests: unit test with fake runner verifying call counts.
- [ ] Task 3: Expand `internal/loop/context.go:parseTasksFromPlan` to recognize `* [ ]`, `1. [ ]`, and indented checklists; treat "no tasks found" as a warning state; add tests.
  - Acceptance: common checklist formats are parsed; empty result triggers warning in logs.
  - Tests: table-driven parsing tests with different list styles.
- [ ] Task 4: Normalize RALPH_STATUS schema across `internal/loop/context.go`, `internal/project/setup.go` prompt templates, and `internal/analysis/response.go` parsing; add parsing tests for missing/extra fields.
  - Acceptance: required fields are consistent across all templates; parser ignores unknown fields.
  - Tests: parsing tests covering canonical, missing, extra, and shuffled fields.
- [ ] Task 5: Wire optional task auto-sync using `internal/loop/tasksync.go` (feature flag) and log evidence without auto-marking unless confidence is high; add tests for evidence detection.
  - Acceptance: task auto-sync is off by default; when on, only high-confidence items are auto-marked.
  - Tests: unit tests for evidence detection and "no auto-mark" path.

### Verification
- [ ] All tests pass (`go test ./...`)
- [ ] Loop exits without a backend call when all tasks are already complete
- [ ] Task parsing covers common checklist formats

## Phase 3: UX & Prompt Consistency
### Goals
- Make loop state obvious (why a loop runs or stops).
- Ensure prompts, context injection, and analysis parse the same status schema.
- Standardize plan/task formatting for reliable parsing.

### Tasks
- [ ] Task 1: Add a preflight summary event in `internal/loop/controller.go` with mode, plan file, remaining count, breaker state, and rate-limit status.
  - Acceptance: event payload includes all fields; emitted before any backend call.
  - Tests: unit test in `internal/loop/controller_test.go` (or new file) asserting emitted event order and payload.
- [ ] Task 2: Extend `internal/loop/events.go` with a `preflight` event type and add a strongly-typed payload in `internal/loop/controller.go`.
  - Acceptance: new event type compiled into TUI and test harness without breaking existing events.
  - Tests: compile-time coverage via existing tests plus a new event-type assertion.
- [ ] Task 3: Render the preflight summary in the TUI (`internal/tui/model.go`, `internal/tui/views.go`) and show the reason when the loop is skipped (complete, rate-limited, breaker open).
  - Acceptance: TUI shows “Skipped: <reason>” and preflight details before any loop output.
  - Tests: snapshot-style view test that includes the preflight block and skip reason.
- [ ] Task 4: Add a “Next Tasks” panel in the TUI that always shows the first N remaining tasks and the total remaining count (`internal/tui/views.go`).
  - Acceptance: panel shows up to N items, plus “+X more” for overflow, and total count.
  - Tests: view test with 0, 1, N, and N+1 tasks.
- [ ] Task 5: Emit a single “loop outcome” event at the end of each iteration with tasks completed, files modified, tests status, and exit signal (`internal/loop/controller.go`).
  - Acceptance: outcome event emitted on both success and error paths (with best-effort data).
  - Tests: controller tests assert outcome event presence for success and failure.
- [ ] Task 6: Normalize the RALPH_STATUS schema in `internal/loop/context.go`, `internal/project/setup.go:defaultPromptTemplate`, and `internal/analysis/response.go` (include CURRENT_TASK consistently; deprecate fields not parsed).
  - Acceptance: all templates show identical fields and ordering; analysis accepts any order.
  - Tests: response parsing tests with canonical ordering and shuffled ordering.
- [ ] Task 7: Add prompt rules to the default prompt: “pick topmost unchecked task,” “do not create new tasks in responses,” and a BLOCKED template for missing inputs (`internal/project/setup.go`).
  - Acceptance: generated PROMPT.md includes the rules verbatim and appears in new projects.
  - Tests: unit test for `defaultPromptTemplate()` string content.
- [ ] Task 8: Standardize plan checklist format to `- [ ]` in `internal/project/setup.go:defaultFixPlanTemplate` and document the requirement in generated prompts.
  - Acceptance: generated @fix_plan.md uses only `- [ ]` and no other checkbox formats.
  - Tests: unit test asserting checklist format for default plan template.
- [ ] Task 9: Add unit tests for RALPH_STATUS parsing (missing fields, extra fields, mixed order) in `internal/analysis/response_test.go`.
  - Acceptance: parser returns defaults for missing fields and ignores unknown fields.
  - Tests: table-driven tests covering all three cases.
- [ ] Task 10: Add TUI rendering tests for preflight and next-tasks panels (snapshot-style) in `internal/tui` where feasible.
  - Acceptance: snapshot tests cover default, empty, and overflow states.
  - Tests: view tests with fixed-width rendering and deterministic output.

### Verification
- [ ] All tests pass (`go test ./...`)
- [ ] Preflight summary shows mode/plan/reason before a loop runs
- [ ] RALPH_STATUS parsing succeeds for canonical and reordered blocks

## Phase 4: OpenCode API Alignment
### Goals
- Align OpenCode client requests and event streaming with the documented server spec.
- Improve reliability and reduce unnecessary retries.

### Tasks
- [ ] Task 1: Update `internal/opencode/client.go` to use the documented SSE endpoint (`/global/event`) and add a fallback if the legacy `/event` path is required.
  - Acceptance: streaming succeeds via `/global/event`; fallback activates only on failure.
  - Tests: mock server tests for both endpoints.
- [ ] Task 2: Align message payloads with the server spec: send `model` objects (providerID/modelID) and support `noReply`, `system`, and `agent` fields where applicable; add tests with a mock server.
  - Acceptance: request JSON matches spec fields; old fields removed or deprecated.
  - Tests: request-body assertions in client tests.
- [ ] Task 3: Add a sync-path fallback using `POST /session/:id/message` when streaming fails, and surface clearer errors when `session.status` indicates retry/error.
  - Acceptance: fallback path returns content on SSE failure; errors include status type and message.
  - Tests: mock SSE failure triggers sync fallback; error formatting tests.
- [ ] Task 4: On context cancellation, call `POST /session/:id/abort` and close SSE cleanly.
  - Acceptance: abort is invoked on ctx cancel; SSE goroutine exits without leaks.
  - Tests: cancellation test ensures abort endpoint hit and goroutine ends.
- [ ] Task 5: Add a preflight `/global/health` check when connecting to an external server to fail fast with a helpful error.
  - Acceptance: invalid server URL fails before any session call; error includes URL.
  - Tests: mock health endpoint returns non-200 and surfaces error.

### Verification
- [ ] All tests pass (`go test ./...`)
- [ ] Streaming works against the documented SSE endpoint
- [ ] Failure modes produce actionable errors (status + endpoint)

## Phase 5: Core Refactoring
### Goals
- Consolidate Codex execution and JSONL parsing into a single helper.
- Normalize output assembly and error handling across init/setup flows.

### Tasks
- [ ] Task 1: Extract `internal/project/codex.go:RunCodex(prompt string, opts CodexOptions)` and route `internal/project/init.go:generateWithCodex` through it.
  - Acceptance: both init and setup call the shared helper; outputs unchanged.
  - Tests: unit test with fake runner to assert prompt wiring.
- [ ] Task 2: Replace `internal/project/setup.go:generateTemplatesWithCodex` with the shared helper while preserving its working directory behavior.
  - Acceptance: templates are generated in the expected directory; no path regressions.
  - Tests: temp-dir test asserting file creation location.
- [ ] Task 3: Add `internal/project/codex.go:ParseCodexJSONL(r io.Reader)` with a larger scanner buffer and use it in all Codex callers.
  - Acceptance: large JSONL lines parse without scanner errors.
  - Tests: parsing test with oversized JSONL line.
- [ ] Task 4: Add tests for JSONL parsing covering `message`, `content_block_delta`, and `assistant` content array events.
  - Acceptance: all supported event types yield expected output text.
  - Tests: table-driven JSONL parsing tests.

### Verification
- [ ] All tests pass (`go test ./...`)
- [ ] Performance unchanged for large prompts (spot check with verbose output)
- [ ] Init and Setup flows produce identical files to pre-refactor runs

## Phase 6: Performance & Consolidation
### Goals
- Reduce repeated filesystem work per loop.
- Consolidate duplicated helpers and normalize shared behaviors.
- Improve streaming efficiency without changing output.

### Tasks
- [ ] Task 1: Cache mode/plan/tasks once per loop in `internal/loop/controller.go` and pass through to `ExecuteLoop`/`ShouldContinue`.
  - Acceptance: plan file is read once per loop; no behavior change.
  - Tests: controller test verifies single plan read with a spy loader.
- [ ] Task 2: Rename `LoadFixPlanWithFile` to `LoadPlanWithFile` and reuse it from `internal/loop/tasksync.go`.
  - Acceptance: function name reflects cross-mode behavior; all call sites updated.
  - Tests: compile + existing tests; new test in `internal/loop/context_test.go` for refactor/implement/fix modes.
- [ ] Task 3: Consolidate project-root discovery into a single helper to remove duplicate `GetProjectRoot` in loop/setup paths.
  - Acceptance: only one root resolver remains; no import cycles.
  - Tests: root detection tests still pass in `internal/loop/context_test.go`.
- [ ] Task 4: Make OpenCode stream assembly deterministic by preserving part order (avoid map iteration) in `internal/opencode/client.go`.
  - Acceptance: response text order matches server part sequence.
  - Tests: streaming unit test with ordered parts assertion.
- [ ] Task 5: Add a lightweight “no-change” gate before calling the backend (git diff or file timestamp heuristic) in `internal/loop/controller.go`.
  - Acceptance: when no files change since last loop and tasks unchanged, loop skips backend call with a log reason.
  - Tests: controller test with mocked diff provider.
- [ ] Task 6: Centralize plan/prompt resolution helpers into a shared package or file to reduce duplication between `internal/loop/context.go` and `internal/project/mode.go`.
  - Acceptance: plan/prompt resolution logic lives in one place; callers delegate.
  - Tests: existing mode tests pass; add a new unit test for shared helper.

### Verification
- [ ] All tests pass (`go test ./...`)
- [ ] Loop reads plan files once per iteration
- [ ] OpenCode streaming output remains identical for the same SSE input

## Phase 7: Cleanup and Polish
### Goals
- Preserve all imported content.
- Make template resolution explicit and configurable without breaking defaults.

### Tasks
- [ ] Task 1: Update `internal/project/import.go:parseSourceContent` to accumulate section content instead of resetting on each heading; add tests with interleaved headings.
  - Acceptance: repeated headings keep earlier content; output preserves all sections.
  - Tests: import parsing tests with interleaved headings.
- [ ] Task 2: Introduce `internal/project/prompts.go:TemplateResolver` and route `Get*Prompt` helpers through it while keeping `TemplateDir` as a backward-compatible default.
  - Acceptance: resolver supports explicit directories and legacy globals.
  - Tests: resolver tests for custom, home, and default paths.
- [ ] Task 3: Normalize errors in `internal/project/init.go`, `internal/project/setup.go`, and `internal/project/import.go` to include file paths and action context.
  - Acceptance: errors include action + path (e.g., “read PRD.md”).
  - Tests: error message assertions in unit tests.
- [ ] Task 4: Remove dead or duplicated helpers uncovered by consolidation (e.g., legacy prompt-loading paths).
  - Acceptance: no unused helpers remain; package build passes with `-deadcode` tooling if used.
  - Tests: `go test ./...` passes; optional staticcheck/unused check if configured.

### Verification
- [ ] All tests pass (`go test ./...`)
- [ ] Documentation updated if public behavior or configuration changes
- [ ] No dead code remaining

## E2E Testing
### Goals
- Validate end-to-end looping across all modes using a fake project.
- Ensure the loop stops and reports progress when tasks are completed.
- Confirm TUI rendering is verifiable in a headless/PTY test.

### Tasks
- [ ] Task 1: Add an e2e harness that generates a temp project for each mode (fix/implement/refactor), writes minimal plan files, and runs the loop with a fake runner that marks tasks complete.
  - Acceptance: each mode exits with “complete” without calling real Codex/OpenCode.
  - Tests: `tests/e2e_loop_test.go` (Go) or `tests/e2e_loop.bats` if preferred.
- [ ] Task 2: Add a CLI e2e test that runs `./ralph --status` and `./ralph --calls 1` against the fake project, verifying progress output and clean exit.
  - Acceptance: CLI output shows plan file, remaining count, and exit condition.
  - Tests: Bats test in `tests/` with fixtures.
- [ ] Task 3: Add a headless TUI e2e test using Bubble Tea `WithoutRenderer()` and assert on `View()` snapshots during loop execution.
  - Acceptance: rendered output includes preflight block, next-tasks panel, and “complete” status.
  - Tests: Go test using `tea.NewProgram(..., tea.WithoutRenderer())`.
- [ ] Task 4: Add a PTY-based TUI e2e test that runs `./ralph --monitor` and captures terminal output until completion.
  - Acceptance: PTY capture includes preflight block, next-tasks panel, and “complete” status.
  - Tests: Go test using `github.com/creack/pty` (or Bats if preferred).
- [ ] Task 5: Add fixtures for each mode (minimal PRD/REFACTOR/PROMPT + plan) in `tests/fixtures/` and a helper to copy them into temp dirs.
  - Acceptance: fixtures are stable and used by all e2e tests.
  - Tests: helper tests for fixture setup/teardown.

### Verification
- [ ] All e2e tests pass on CI and locally
- [ ] TUI e2e test proves the loop can be observed end-to-end without manual interaction

## Rollback Plan
- Revert the commit for the impacted phase; each task is independently committable.
- Restore original Codex execution functions if the shared helper changes output.
- Revert OpenCode client changes if API alignment breaks streaming; keep tests to guard regressions.
- Keep new tests even if code changes are reverted to preserve coverage.

## Success Criteria
- [ ] `internal/project/setup.go:initGitRepo` performs real git initialization with actionable errors on failure.
- [ ] All Codex invocations use one helper with consistent JSONL parsing.
- [ ] Template resolution is testable without relying on global state.
- [ ] Import parsing preserves all sections with repeated headings.
- [ ] Unit tests cover specs reading, template resolution, and JSONL parsing.
- [ ] Loop exits early when plans are already complete and rate limits are enforced.
- [ ] OpenCode streaming and message payloads match the documented server spec.
- [ ] TUI shows preflight status and next tasks; RALPH_STATUS parsing is consistent across templates.
- [ ] Loop and TUI e2e tests pass for fix, implement, and refactor modes.

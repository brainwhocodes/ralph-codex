package loop

import (
	stdcontext "context"
	"fmt"
	"strings"
	"time"

	"github.com/brainwhocodes/ralph-codex/internal/analysis"
	"github.com/brainwhocodes/ralph-codex/internal/circuit"
	"github.com/brainwhocodes/ralph-codex/internal/codex"
	"github.com/brainwhocodes/ralph-codex/internal/config"
	"github.com/brainwhocodes/ralph-codex/internal/runner"
	"github.com/brainwhocodes/ralph-codex/internal/state"
)

// Config is an alias to the unified config type
type Config = config.Config

// LoopEvent represents an event from the loop controller
type LoopEvent struct {
	Type         EventType
	LoopNumber   int
	CallsUsed    int
	Status       string
	LogMessage   string
	LogLevel     LogLevel
	CircuitState string

	// Codex output streaming fields
	OutputLine    string // Raw output line
	OutputType    OutputType
	ReasoningText string // Reasoning/thinking text
	ToolName      string // Tool being called
	ToolTarget    string // File path or command
	ToolStatus    ToolStatus

	// Analysis result fields (from RALPH_STATUS block)
	AnalysisStatus     string  // WORKING, COMPLETE, BLOCKED
	CurrentTask        string  // Current task being worked on or just completed
	TasksCompleted     int     // Tasks completed this loop
	FilesModified      int     // Files modified this loop
	TestsStatus        string  // PASSING, FAILING, UNKNOWN
	ExitSignal         bool    // Whether exit was signaled
	ConfidenceScore    float64 // Confidence in completion (0-1)

	// Context tracking fields
	ContextUsagePercent  float64 // Current context window usage (0-1)
	ContextTotalTokens   int     // Total tokens used
	ContextLimit         int     // Context window limit
	ContextThreshold     bool    // True if threshold reached
	ContextWasCompacted  bool    // True if OpenCode compacted the session
}

// EventCallback is called when the controller has an update
type EventCallback func(event LoopEvent)

// Controller manages the main Ralph loop
type Controller struct {
	config        ControllerConfig
	rateLimiter   *RateLimiter
	breaker       *circuit.Breaker
	runner        runner.Runner
	loopNum       int
	lastOutput    string
	shouldStop    bool
	eventCallback EventCallback
	paused        bool
	backend       string
}

// ControllerConfig holds configuration for the loop controller
type ControllerConfig struct {
	MaxLoops      int
	MaxDuration   time.Duration
	CheckInterval time.Duration
}

// NewController creates a new loop controller
func NewController(cfg Config, rateLimiter *RateLimiter, breaker *circuit.Breaker) *Controller {
	// Create runner based on backend selection
	r := runner.New(cfg)

	c := &Controller{
		config: ControllerConfig{
			MaxLoops:      cfg.MaxCalls,
			MaxDuration:   time.Duration(cfg.Timeout) * time.Second,
			CheckInterval: 5 * time.Second,
		},
		rateLimiter:   rateLimiter,
		breaker:       breaker,
		runner:        r,
		loopNum:       0,
		lastOutput:    "",
		shouldStop:    false,
		eventCallback: nil,
		paused:        false,
		backend:       cfg.Backend,
	}

	// Set up output callback for streaming
	r.SetOutputCallback(func(event runner.Event) {
		c.handleCodexEvent(codex.Event(event))
	})

	return c
}

// SetEventCallback sets the callback for loop events
func (c *Controller) SetEventCallback(cb EventCallback) {
	c.eventCallback = cb
}

// emit sends an event to the callback if set
func (c *Controller) emit(event LoopEvent) {
	if c.eventCallback != nil {
		c.eventCallback(event)
	}
}

// emitLog sends a log event
func (c *Controller) emitLog(level LogLevel, message string) {
	c.emit(LoopEvent{
		Type:       EventTypeLog,
		LogMessage: message,
		LogLevel:   level,
	})
}

// emitUpdate sends a loop update event
func (c *Controller) emitUpdate(status string) {
	c.emit(LoopEvent{
		Type:         EventTypeLoopUpdate,
		LoopNumber:   c.loopNum,
		CallsUsed:    c.rateLimiter.CallsMade(),
		Status:       status,
		CircuitState: c.breaker.GetState().String(),
	})
}

// emitCodexOutput sends a codex output event
func (c *Controller) emitCodexOutput(line string, outputType OutputType) {
	c.emit(LoopEvent{
		Type:       EventTypeCodexOutput,
		OutputLine: line,
		OutputType: outputType,
	})
}

// emitCodexReasoning sends a codex reasoning event
func (c *Controller) emitCodexReasoning(text string) {
	c.emit(LoopEvent{
		Type:          EventTypeCodexReasoning,
		ReasoningText: text,
	})
}

// emitCodexTool sends a codex tool call event
func (c *Controller) emitCodexTool(toolName, target string, status ToolStatus) {
	c.emit(LoopEvent{
		Type:       EventTypeCodexTool,
		ToolName:   toolName,
		ToolTarget: target,
		ToolStatus: status,
	})
}

// emitAnalysis sends analysis results from RALPH_STATUS block
func (c *Controller) emitAnalysis(result *analysis.Analysis) {
	if result == nil {
		return
	}

	event := LoopEvent{
		Type:            EventTypeAnalysis,
		LoopNumber:      c.loopNum,
		ExitSignal:      result.ExitSignal,
		ConfidenceScore: result.ConfidenceScore,
	}

	if result.Status != nil {
		event.AnalysisStatus = result.Status.Status
		event.CurrentTask = result.Status.CurrentTask
		event.TasksCompleted = result.Status.TasksCompleted
		event.FilesModified = result.Status.FilesModified
		event.TestsStatus = result.Status.TestsStatus
	}

	c.emit(event)
}

// emitContextUsage sends context window usage event
func (c *Controller) emitContextUsage(usagePercent float64, totalTokens, limit int, thresholdReached, wasCompacted bool) {
	c.emit(LoopEvent{
		Type:                 EventTypeContextUsage,
		LoopNumber:           c.loopNum,
		ContextUsagePercent:  usagePercent,
		ContextTotalTokens:   totalTokens,
		ContextLimit:         limit,
		ContextThreshold:     thresholdReached,
		ContextWasCompacted:  wasCompacted,
	})
}

// Pause pauses the loop
func (c *Controller) Pause() {
	c.paused = true
	c.emitLog(LogLevelInfo, "Loop paused")
}

// Resume resumes the loop
func (c *Controller) Resume() {
	c.paused = false
	c.emitLog(LogLevelInfo, "Loop resumed")
}

// IsPaused returns whether the loop is paused
func (c *Controller) IsPaused() bool {
	return c.paused
}

// Stop signals the loop to stop
func (c *Controller) Stop() {
	c.shouldStop = true
}

// Run executes the main loop
func (c *Controller) Run(ctx stdcontext.Context) error {
	c.emitLog(LogLevelInfo, fmt.Sprintf("Starting Ralph Codex loop (max %d calls)", c.config.MaxLoops))
	c.emitUpdate("starting")

	for {
		// Check if paused
		if c.paused {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		if c.shouldStop {
			c.emitLog(LogLevelSuccess, "Loop stopped")
			c.emitUpdate("stopped")
			return nil
		}

		select {
		case <-ctx.Done():
			c.emitLog(LogLevelWarn, "Loop cancelled")
			c.emitUpdate("cancelled")
			return ctx.Err()
		default:
			c.emitUpdate("running")

			// Execute one iteration
			err := c.ExecuteLoop(ctx)

			if err != nil {
				c.emitLog(LogLevelError, fmt.Sprintf("Loop iteration error: %v", err))
				c.emitUpdate("error")
				// Don't return on error - start a new loop iteration instead
				// This handles message.error and other transient failures
				c.emitLog(LogLevelInfo, "Waiting 5s before retrying...")
				time.Sleep(5 * time.Second)
				c.emitLog(LogLevelInfo, "Starting new loop iteration after error...")
				c.loopNum++
				continue
			}

			// Check if we should stop
			if c.ShouldContinue() {
				c.emitLog(LogLevelSuccess, fmt.Sprintf("Ralph Codex loop complete after %d iterations", c.loopNum))
				c.emitUpdate("complete")
				return nil
			}

			c.loopNum++
		}
	}
}

// ExecuteLoop executes a single loop iteration
func (c *Controller) ExecuteLoop(ctx stdcontext.Context) error {
	// Check if paused
	if c.paused {
		time.Sleep(100 * time.Millisecond)
		return nil
	}

	c.emitUpdate("executing")

	// Check rate limit
	if !c.rateLimiter.CanMakeCall() {
		c.emitLog(LogLevelWarn, fmt.Sprintf("Rate limit reached. Calls remaining: %d", c.rateLimiter.CallsRemaining()))
		c.emitUpdate("rate_limited")
		return c.rateLimiter.WaitForReset(ctx)
	}

	// Check circuit breaker
	if c.breaker.ShouldHalt() {
		c.emitLog(LogLevelError, "Circuit breaker is OPEN, halting execution")
		c.emitUpdate("circuit_open")
		return fmt.Errorf("circuit breaker is OPEN, halting execution")
	}

	// Load prompt and fix plan
	prompt, err := GetPrompt()
	if err != nil {
		c.emitLog(LogLevelError, fmt.Sprintf("Failed to load prompt: %v", err))
		c.emitUpdate("error")
		return fmt.Errorf("failed to load prompt: %w", err)
	}

	tasks, planFile, err := LoadFixPlanWithFile()
	if err != nil {
		c.emitLog(LogLevelError, fmt.Sprintf("Failed to load fix plan: %v", err))
		c.emitUpdate("error")
		return fmt.Errorf("failed to load fix plan: %w", err)
	}

	// Build context
	circuitState := c.breaker.GetState().String()
	remainingTasks := []string{}
	for _, task := range tasks {
		if !strings.HasPrefix(task, "[x]") {
			remainingTasks = append(remainingTasks, task)
		}
	}

	loopContext, err := BuildContextWithPlanFile(c.loopNum+1, remainingTasks, circuitState, c.lastOutput, planFile)
	if err != nil {
		c.emitLog(LogLevelError, fmt.Sprintf("Failed to build context: %v", err))
		c.emitUpdate("error")
		return fmt.Errorf("failed to build context: %w", err)
	}

	promptWithContext := InjectContext(prompt, loopContext)

	// Execute runner (Codex CLI or OpenCode)
	backendName := "Codex"
	if c.backend == "opencode" {
		backendName = "OpenCode"
	}
	c.emitLog(LogLevelInfo, fmt.Sprintf("Loop %d: Executing %s", c.loopNum+1, backendName))
	c.emitUpdate("codex_running")
	c.emitCodexOutput(fmt.Sprintf("Starting %s execution (loop %d)...", backendName, c.loopNum+1), OutputTypeRaw)
	c.emitCodexOutput(fmt.Sprintf("Prompt size: %d bytes", len(promptWithContext)), OutputTypeRaw)
	output, _, err := c.runner.Run(promptWithContext)

	if err != nil {
		// Don't pass error messages as prevSummary - they confuse the AI
		// Clear lastOutput so the next loop gets a clean start
		c.lastOutput = ""
		if rlErr := c.rateLimiter.RecordCall(); rlErr != nil {
			c.emitLog(LogLevelWarn, fmt.Sprintf("Failed to record call: %v", rlErr))
		}

		// Record error in circuit breaker
		if cbErr := c.breaker.RecordError(err.Error()); cbErr != nil {
			c.emitLog(LogLevelWarn, fmt.Sprintf("Failed to record error in circuit breaker: %v", cbErr))
		}
		c.emitLog(LogLevelError, fmt.Sprintf("Codex execution failed: %v", err))
		c.emitUpdate("execution_error")
		return err
	}

	// Store a clean summary of the output for the next loop
	// Truncate to ~200 chars at word boundary to avoid confusing partial text
	summary := output
	if len(summary) > 200 {
		summary = summary[:200]
		// Find last space to avoid mid-word cutoff
		if lastSpace := strings.LastIndex(summary, " "); lastSpace > 100 {
			summary = summary[:lastSpace]
		}
		summary += "..."
	}
	c.lastOutput = summary
	c.emitLog(LogLevelSuccess, fmt.Sprintf("Loop %d completed successfully", c.loopNum+1))
	c.emitUpdate("execution_complete")

	// Analyze output for exit conditions using the analysis package
	exitSignals, _ := state.LoadExitSignals()
	analysisResult, err := analysis.Analyze(output, exitSignals)
	if err != nil {
		c.emitLog(LogLevelWarn, fmt.Sprintf("Output analysis failed: %v", err))
	}

	// Determine hasErrors and filesChanged from analysis
	hasErrors := false
	filesChanged := 0
	if analysisResult != nil {
		hasErrors = analysisResult.HasErrors
		if analysisResult.Status != nil {
			filesChanged = analysisResult.Status.FilesModified
		}

		// Emit analysis results to UI
		c.emitAnalysis(analysisResult)

		// If exit signal detected, persist it and signal stop
		if analysisResult.ExitSignal {
			c.emitLog(LogLevelSuccess, "✓ EXIT_SIGNAL: true - Work complete!")
			exitSignals = append(exitSignals, fmt.Sprintf("loop_%d", c.loopNum+1))
			_ = state.SaveExitSignals(exitSignals)
			c.shouldStop = true
		}

		// Check for completion based on confidence
		if analysisResult.ConfidenceScore >= 0.9 && analysisResult.Status != nil && analysisResult.Status.Status == "COMPLETE" {
			c.emitLog(LogLevelSuccess, "✓ High-confidence completion detected (STATUS: COMPLETE)")
			c.shouldStop = true
		}
	}

	// Record result in circuit breaker
	err = c.breaker.RecordResult(c.loopNum, filesChanged, hasErrors)
	if err != nil {
		c.emitLog(LogLevelError, fmt.Sprintf("Failed to record result: %v", err))
		c.emitUpdate("error")
		return err
	}

	return nil
}

// ShouldContinue checks if the loop should continue
func (c *Controller) ShouldContinue() bool {
	tasks, err := LoadFixPlan()
	if err != nil {
		return false
	}

	// Check if all tasks are complete
	allComplete := true
	for _, task := range tasks {
		if !strings.HasPrefix(task, "[x]") {
			allComplete = false
			break
		}
	}

	if allComplete {
		c.shouldStop = true
		return true
	}

	// Check circuit breaker
	if c.breaker.ShouldHalt() {
		c.shouldStop = true
		return true
	}

	// Check rate limit
	if !c.rateLimiter.CanMakeCall() {
		c.shouldStop = true
		return true
	}

	// Check max loops
	if c.loopNum >= c.config.MaxLoops {
		c.shouldStop = true
		return true
	}

	return false
}

// GracefulExit performs cleanup before exiting
func (c *Controller) GracefulExit() error {
	fmt.Println("\n🧹 Performing graceful exit...")

	c.shouldStop = true

	// Stop the runner (shuts down managed servers if any)
	if c.runner != nil {
		if err := c.runner.Stop(); err != nil {
			fmt.Printf("Warning: failed to stop runner: %v\n", err)
		}
	}

	// Reset circuit breaker
	if err := c.breaker.Reset(); err != nil {
		fmt.Printf("Warning: failed to reset circuit breaker: %v\n", err)
	}

	// Reset session
	if err := codex.NewSession(); err != nil {
		return fmt.Errorf("failed to reset session: %w", err)
	}

	fmt.Println("✅ Graceful exit complete")
	return nil
}

// GetStats returns controller statistics
func (c *Controller) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"loop_num":        c.loopNum,
		"should_stop":     c.shouldStop,
		"rate_limiter":    c.rateLimiter.GetStats(),
		"circuit_breaker": c.breaker.GetStats(),
		"last_output":     c.lastOutput,
	}
}

// handleCodexEvent processes streaming events from codex and emits them to TUI
// Uses the unified event parser from the codex package
func (c *Controller) handleCodexEvent(event codex.Event) {
	// Debug: log raw event type
	eventType, _ := event["type"].(string)
	if eventType != "" {
		c.emitLog(LogLevelDebug, fmt.Sprintf("SSE event: %s", eventType))
	}

	// Handle context usage events directly (not parsed by codex parser)
	if eventType == "context.usage" {
		usagePercent, _ := event["usage_percent"].(float64)
		totalTokens, _ := event["total_tokens"].(float64)
		limit, _ := event["context_limit"].(float64)
		thresholdReached, _ := event["threshold_reached"].(bool)
		wasCompacted, _ := event["was_compacted"].(bool)
		c.emitContextUsage(usagePercent, int(totalTokens), int(limit), thresholdReached, wasCompacted)
		return
	}

	parsed := codex.ParseEvent(event)
	if parsed == nil {
		return
	}

	// Debug: log parsed type
	if parsed.Text != "" || parsed.ToolName != "" {
		c.emitLog(LogLevelDebug, fmt.Sprintf("Parsed: type=%s text=%d chars", parsed.Type, len(parsed.Text)))
	}

	// Emit based on parsed event type
	switch parsed.Type {
	case "reasoning":
		if parsed.Text != "" {
			c.emitCodexReasoning(parsed.Text)
		}

	case "message", "delta":
		if parsed.Text != "" {
			c.emitCodexOutput(parsed.Text, OutputTypeAgentMessage)
		}

	case "tool_call", "tool_result":
		if parsed.ToolName != "" {
			status := ToolStatusStarted
			if parsed.ToolStatus == "completed" {
				status = ToolStatusCompleted
			}
			c.emitCodexTool(parsed.ToolName, parsed.ToolTarget, status)
		}

	case "lifecycle":
		// Lifecycle events (start, stop, etc.) - just show the type
		if parsed.RawType != "" {
			c.emitCodexOutput(fmt.Sprintf(">>> %s", parsed.RawType), OutputTypeRaw)
		}

	default:
		// Unknown event with text
		if parsed.Text != "" {
			c.emitCodexOutput(parsed.Text, OutputTypeRaw)
		}
	}
}

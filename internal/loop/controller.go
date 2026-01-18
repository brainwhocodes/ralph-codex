package loop

import (
	stdcontext "context"
	"fmt"
	"strings"
	"time"

	"github.com/brainwhocodes/ralph-codex/internal/circuit"
	"github.com/brainwhocodes/ralph-codex/internal/codex"
)

// Config holds configuration for the loop
type Config struct {
	Backend      string
	ProjectPath  string
	PromptPath   string
	MaxCalls     int
	Timeout      int
	Verbose      bool
	ResetCircuit bool
}

// LoopEvent represents an event from the loop controller
type LoopEvent struct {
	Type        string // "loop_update", "log", "state_change", "status"
	LoopNumber  int
	CallsUsed   int
	Status      string
	LogMessage  string
	LogLevel    string // INFO, WARN, ERROR, SUCCESS
	CircuitState string
}

// EventCallback is called when the controller has an update
type EventCallback func(event LoopEvent)

// Controller manages the main Ralph loop
type Controller struct {
	config        ControllerConfig
	rateLimiter   *RateLimiter
	breaker       *circuit.Breaker
	codexRunner   *codex.Runner
	loopNum       int
	lastOutput    string
	shouldStop    bool
	eventCallback EventCallback
	paused        bool
}

// ControllerConfig holds configuration for the loop controller
type ControllerConfig struct {
	MaxLoops      int
	MaxDuration   time.Duration
	CheckInterval time.Duration
}

// NewController creates a new loop controller
func NewController(config Config, rateLimiter *RateLimiter, breaker *circuit.Breaker) *Controller {
	codexConfig := codex.Config{
		Backend:      config.Backend,
		ProjectPath:  config.ProjectPath,
		PromptPath:   config.PromptPath,
		MaxCalls:     config.MaxCalls,
		Timeout:      config.Timeout,
		Verbose:      config.Verbose,
		ResetCircuit: config.ResetCircuit,
	}
	codexRunner := codex.NewRunner(codexConfig)

	return &Controller{
		config: ControllerConfig{
			MaxLoops:      config.MaxCalls,
			MaxDuration:   time.Duration(config.Timeout) * time.Second,
			CheckInterval: 5 * time.Second,
		},
		rateLimiter:   rateLimiter,
		breaker:       breaker,
		codexRunner:   codexRunner,
		loopNum:       0,
		lastOutput:    "",
		shouldStop:    false,
		eventCallback: nil,
		paused:        false,
	}
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
func (c *Controller) emitLog(level, message string) {
	c.emit(LoopEvent{
		Type:       "log",
		LogMessage: message,
		LogLevel:   level,
	})
}

// emitUpdate sends a loop update event
func (c *Controller) emitUpdate(status string) {
	c.emit(LoopEvent{
		Type:         "loop_update",
		LoopNumber:   c.loopNum,
		CallsUsed:    c.rateLimiter.CallsMade(),
		Status:       status,
		CircuitState: c.breaker.GetState().String(),
	})
}

// Pause pauses the loop
func (c *Controller) Pause() {
	c.paused = true
	c.emitLog("INFO", "Loop paused")
}

// Resume resumes the loop
func (c *Controller) Resume() {
	c.paused = false
	c.emitLog("INFO", "Loop resumed")
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
	c.emitLog("INFO", fmt.Sprintf("Starting Ralph Codex loop (max %d calls)", c.config.MaxLoops))
	c.emitUpdate("starting")

	for {
		// Check if paused
		if c.paused {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		if c.shouldStop {
			c.emitLog("SUCCESS", "Loop stopped")
			c.emitUpdate("stopped")
			return nil
		}

		select {
		case <-ctx.Done():
			c.emitLog("WARN", "Loop cancelled")
			c.emitUpdate("cancelled")
			return ctx.Err()
		default:
			c.emitUpdate("running")

			// Execute one iteration
			err := c.ExecuteLoop(ctx)

			if err != nil {
				c.emitLog("ERROR", fmt.Sprintf("Loop iteration error: %v", err))
				c.emitUpdate("error")
				return err
			}

			// Check if we should stop
			if c.ShouldContinue() {
				c.emitLog("SUCCESS", fmt.Sprintf("Ralph Codex loop complete after %d iterations", c.loopNum))
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
		c.emitLog("WARN", fmt.Sprintf("Rate limit reached. Calls remaining: %d", c.rateLimiter.CallsRemaining()))
		c.emitUpdate("rate_limited")
		return c.rateLimiter.WaitForReset(ctx)
	}

	// Check circuit breaker
	if c.breaker.ShouldHalt() {
		c.emitLog("ERROR", "Circuit breaker is OPEN, halting execution")
		c.emitUpdate("circuit_open")
		return fmt.Errorf("circuit breaker is OPEN, halting execution")
	}

	// Load prompt and fix plan
	prompt, err := GetPrompt()
	if err != nil {
		c.emitLog("ERROR", fmt.Sprintf("Failed to load prompt: %v", err))
		c.emitUpdate("error")
		return fmt.Errorf("failed to load prompt: %w", err)
	}

	tasks, err := LoadFixPlan()
	if err != nil {
		c.emitLog("ERROR", fmt.Sprintf("Failed to load fix plan: %v", err))
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

	loopContext, err := BuildContext("", c.loopNum+1, remainingTasks, circuitState, c.lastOutput)
	if err != nil {
		c.emitLog("ERROR", fmt.Sprintf("Failed to build context: %v", err))
		c.emitUpdate("error")
		return fmt.Errorf("failed to build context: %w", err)
	}

	promptWithContext := InjectContext(prompt, loopContext)

	// Execute Codex
	c.emitLog("INFO", fmt.Sprintf("Loop %d: Executing Codex", c.loopNum+1))
	c.emitUpdate("codex_running")
	output, _, err := c.codexRunner.Run(promptWithContext)

	if err != nil {
		c.lastOutput = fmt.Sprintf("Error: %v", err)
		c.rateLimiter.RecordCall()

		// Record error in circuit breaker
		c.breaker.RecordError(err.Error())
		c.emitLog("ERROR", fmt.Sprintf("Codex execution failed: %v", err))
		c.emitUpdate("execution_error")
		return err
	}

	c.lastOutput = fmt.Sprintf("Success: %s", output[:min(200, len(output))])
	c.emitLog("SUCCESS", fmt.Sprintf("Loop %d completed successfully", c.loopNum+1))
	c.emitUpdate("execution_complete")

	// Analyze output for exit conditions
	// TODO: This will be implemented in response analysis package

	// Record result in circuit breaker
	filesChanged := 0
	if strings.Contains(output, "Modified") || strings.Contains(output, "Created") {
		filesChanged = 1
	}

	hasErrors := strings.Contains(output, "Error") || strings.Contains(output, "failed")

	err = c.breaker.RecordResult(c.loopNum, filesChanged, hasErrors)
	if err != nil {
		c.emitLog("ERROR", fmt.Sprintf("Failed to record result: %v", err))
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

	// Reset circuit breaker
	c.breaker.Reset()

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

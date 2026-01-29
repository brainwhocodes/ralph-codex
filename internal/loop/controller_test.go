package loop

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/brainwhocodes/lisa-loop/internal/circuit"
)

func TestRunPreflight(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Create a test plan file
	planContent := `
- [ ] First task
- [ ] Second task
- [x] Third task (done)
`
	os.WriteFile("@fix_plan.md", []byte(planContent), 0644)
	os.WriteFile("PROMPT.md", []byte("Test prompt"), 0644)

	// Create controller with test config
	rateLimiter := NewRateLimiter(10, 1)
	breaker := circuit.NewBreaker(3, 5)

	cfg := Config{
		MaxCalls: 5,
		Backend:  "cli",
	}

	controller := NewController(cfg, rateLimiter, breaker)

	t.Run("normal preflight with remaining tasks", func(t *testing.T) {
		summary, shouldSkip := controller.RunPreflight()

		if shouldSkip {
			t.Errorf("RunPreflight() shouldSkip = true, want false")
		}

		if summary.TotalTasks != 3 {
			t.Errorf("RunPreflight() TotalTasks = %d, want 3", summary.TotalTasks)
		}

		if summary.RemainingCount != 2 {
			t.Errorf("RunPreflight() RemainingCount = %d, want 2", summary.RemainingCount)
		}

		if summary.PlanFile != "@fix_plan.md" {
			t.Errorf("RunPreflight() PlanFile = %s, want @fix_plan.md", summary.PlanFile)
		}

		if !summary.RateLimitOK {
			t.Errorf("RunPreflight() RateLimitOK = false, want true")
		}

		if summary.CallsRemaining != 10 {
			t.Errorf("RunPreflight() CallsRemaining = %d, want 10", summary.CallsRemaining)
		}
	})

	t.Run("skip when all tasks complete", func(t *testing.T) {
		// Invalidate cache to force reload
		controller.cacheValid = false

		// Update plan to mark all tasks complete
		completePlan := `
- [x] First task
- [x] Second task
- [x] Third task (done)
`
		os.WriteFile("@fix_plan.md", []byte(completePlan), 0644)

		summary, shouldSkip := controller.RunPreflight()

		if !shouldSkip {
			t.Errorf("RunPreflight() shouldSkip = false, want true")
		}

		if summary.SkipReason != "All tasks complete" {
			t.Errorf("RunPreflight() SkipReason = %s, want 'All tasks complete'", summary.SkipReason)
		}

		// Restore plan for other tests
		os.WriteFile("@fix_plan.md", []byte(planContent), 0644)
	})
}

func TestRunPreflight_MaxLoops(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Create a test plan file with remaining tasks
	planContent := `- [ ] First task`
	os.WriteFile("@fix_plan.md", []byte(planContent), 0644)
	os.WriteFile("PROMPT.md", []byte("Test prompt"), 0644)

	rateLimiter := NewRateLimiter(10, 1)
	breaker := circuit.NewBreaker(3, 5)

	cfg := Config{
		MaxCalls: 1,
		Backend:  "cli",
	}

	controller := NewController(cfg, rateLimiter, breaker)
	controller.loopNum = 1 // Set loop number to max

	summary, shouldSkip := controller.RunPreflight()

	if !shouldSkip {
		t.Errorf("RunPreflight() shouldSkip = false, want true when max loops reached")
	}

	if summary.SkipReason != "Max loops reached (1)" {
		t.Errorf("RunPreflight() SkipReason = %s, want 'Max loops reached (1)'", summary.SkipReason)
	}
}

func TestRunPreflight_NoPlanFile(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// No plan file

	rateLimiter := NewRateLimiter(10, 1)
	breaker := circuit.NewBreaker(3, 5)

	cfg := Config{
		MaxCalls: 5,
		Backend:  "cli",
	}

	controller := NewController(cfg, rateLimiter, breaker)

	summary, shouldSkip := controller.RunPreflight()

	if !shouldSkip {
		t.Errorf("RunPreflight() shouldSkip = false, want true when no plan file")
	}

	if summary.SkipReason != "No plan file found" {
		t.Errorf("RunPreflight() SkipReason = %s, want 'No plan file found'", summary.SkipReason)
	}
}

func TestPreflightSummary_TasksToShow(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Create a test plan file with many tasks
	planContent := ""
	for i := 1; i <= 10; i++ {
		planContent += "- [ ] Task " + string(rune('0'+i)) + "\n"
	}
	os.WriteFile("@fix_plan.md", []byte(planContent), 0644)
	os.WriteFile("PROMPT.md", []byte("Test prompt"), 0644)

	rateLimiter := NewRateLimiter(10, 1)
	breaker := circuit.NewBreaker(3, 5)

	cfg := Config{
		MaxCalls: 5,
		Backend:  "cli",
	}

	controller := NewController(cfg, rateLimiter, breaker)

	summary, _ := controller.RunPreflight()

	// Should only show first 5 tasks
	if len(summary.RemainingTasks) != 5 {
		t.Errorf("RunPreflight() RemainingTasks length = %d, want 5", len(summary.RemainingTasks))
	}

	if summary.RemainingCount != 10 {
		t.Errorf("RunPreflight() RemainingCount = %d, want 10", summary.RemainingCount)
	}
}

func TestEmitPreflight(t *testing.T) {
	rateLimiter := NewRateLimiter(10, 1)
	breaker := circuit.NewBreaker(3, 5)

	cfg := Config{
		MaxCalls: 5,
		Backend:  "cli",
	}

	controller := NewController(cfg, rateLimiter, breaker)

	// Set up event capture
	var capturedEvent *LoopEvent
	controller.SetEventCallback(func(event LoopEvent) {
		if event.Type == EventTypePreflight {
			capturedEvent = &event
		}
	})

	summary := &PreflightSummary{
		Mode:           "fix",
		PlanFile:       "@fix_plan.md",
		TotalTasks:     5,
		RemainingCount: 3,
		CircuitState:   "CLOSED",
		RateLimitOK:    true,
		ShouldSkip:     false,
	}

	controller.emitPreflight(summary)

	if capturedEvent == nil {
		t.Fatal("emitPreflight() did not capture event")
	}

	if capturedEvent.Type != EventTypePreflight {
		t.Errorf("emitPreflight() event.Type = %s, want %s", capturedEvent.Type, EventTypePreflight)
	}

	if capturedEvent.Preflight == nil {
		t.Fatal("emitPreflight() event.Preflight is nil")
	}

	if capturedEvent.Preflight.Mode != "fix" {
		t.Errorf("emitPreflight() Preflight.Mode = %s, want fix", capturedEvent.Preflight.Mode)
	}
}

func TestEmitOutcome(t *testing.T) {
	rateLimiter := NewRateLimiter(10, 1)
	breaker := circuit.NewBreaker(3, 5)

	cfg := Config{
		MaxCalls: 5,
		Backend:  "cli",
	}

	controller := NewController(cfg, rateLimiter, breaker)

	// Set up event capture
	var capturedEvent *LoopEvent
	controller.SetEventCallback(func(event LoopEvent) {
		if event.Type == EventTypeOutcome {
			capturedEvent = &event
		}
	})

	outcome := &LoopOutcome{
		Success:        true,
		TasksCompleted: 2,
		FilesModified:  3,
		TestsStatus:    "PASSING",
		ExitSignal:     false,
	}

	controller.emitOutcome(outcome)

	if capturedEvent == nil {
		t.Fatal("emitOutcome() did not capture event")
	}

	if capturedEvent.Type != EventTypeOutcome {
		t.Errorf("emitOutcome() event.Type = %s, want %s", capturedEvent.Type, EventTypeOutcome)
	}

	if capturedEvent.Outcome == nil {
		t.Fatal("emitOutcome() event.Outcome is nil")
	}

	if !capturedEvent.Outcome.Success {
		t.Errorf("emitOutcome() Outcome.Success = false, want true")
	}

	if capturedEvent.Outcome.TasksCompleted != 2 {
		t.Errorf("emitOutcome() Outcome.TasksCompleted = %d, want 2", capturedEvent.Outcome.TasksCompleted)
	}
}

// Test that EventTypePreflight and EventTypeOutcome constants exist
func TestEventTypes(t *testing.T) {
	if EventTypePreflight != "preflight" {
		t.Errorf("EventTypePreflight = %s, want 'preflight'", EventTypePreflight)
	}

	if EventTypeOutcome != "outcome" {
		t.Errorf("EventTypeOutcome = %s, want 'outcome'", EventTypeOutcome)
	}
}

func TestRateLimiter_RecordSuccessfulCall(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Create minimal test files
	os.MkdirAll(".git", 0755)
	os.WriteFile(".git/config", []byte("test"), 0644)
	os.WriteFile("@fix_plan.md", []byte("- [ ] Task"), 0644)
	os.WriteFile("PROMPT.md", []byte("Test"), 0644)

	// Create state files for rate limiter
	os.MkdirAll(filepath.Join(tmpDir, ".ralph"), 0755)

	rateLimiter := NewRateLimiter(10, 1)
	// Initial state
	if rateLimiter.CallsMade() != 0 {
		t.Errorf("Initial CallsMade = %d, want 0", rateLimiter.CallsMade())
	}

	// Simulate recording a successful call
	err := rateLimiter.RecordCall()
	if err != nil {
		t.Errorf("RecordCall() error = %v", err)
	}

	if rateLimiter.CallsMade() != 1 {
		t.Errorf("After RecordCall, CallsMade = %d, want 1", rateLimiter.CallsMade())
	}
}
